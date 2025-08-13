package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultListenAddr = ":8080"
	defaultTimeout    = 10 * time.Minute
	formFileFieldName = "file"
	headerAuth        = "Authorization"
	contentTypeJSON   = "application/json"
	contentTypeDOCX   = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
)

// config represents runtime configuration derived from env vars
// DOC2DOCX_AUTH_KEY: optional bearer token required for requests
// DOC2DOCX_LISTEN_ADDR: server listen address, default :8080
// DOC2DOCX_TIMEOUT_MS: conversion timeout in milliseconds (default 600000)
// DOC2DOCX_RESPONSE: "binary" (default) or "json" for base64 response
// DOC2DOCX_TMPDIR: override temp working directory
//
// Request can also control response via Accept: application/json or query ?format=json

type config struct {
	listenAddr      string
	authKey         string
	timeout         time.Duration
	defaultRespJSON bool
	tmpDir          string
}

func loadConfig() config {
	cfg := config{
		listenAddr:      getenvDefault("DOC2DOCX_LISTEN_ADDR", defaultListenAddr),
		authKey:         os.Getenv("DOC2DOCX_AUTH_KEY"),
		tmpDir:          os.Getenv("DOC2DOCX_TMPDIR"),
		defaultRespJSON: strings.EqualFold(os.Getenv("DOC2DOCX_RESPONSE"), "json"),
		timeout:         defaultTimeout,
	}
	if v := os.Getenv("DOC2DOCX_TIMEOUT_MS"); v != "" {
		if ms, err := time.ParseDuration(v + "ms"); err == nil {
			cfg.timeout = ms
		}
	}
	return cfg
}

func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	cfg := loadConfig()
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = io.WriteString(w, "ok")
	})
	mux.HandleFunc("/convert/doc2docx", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if cfg.authKey != "" {
			if !validateBearer(r.Header.Get(headerAuth), cfg.authKey) {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
		}

		if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB
			writeError(w, http.StatusBadRequest, fmt.Sprintf("parse multipart: %v", err))
			return
		}
		file, header, err := r.FormFile(formFileFieldName)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("missing form file field '%s'", formFileFieldName))
			return
		}
		defer file.Close()

		inputName := header.Filename
		if !strings.HasSuffix(strings.ToLower(inputName), ".doc") {
			// allow .DOC as well
			writeError(w, http.StatusBadRequest, "uploaded file must have .doc extension")
			return
		}

		// prepare working dir
		workDir := cfg.tmpDir
		if workDir == "" {
			workDir = os.TempDir()
		}
		sessionDir, err := os.MkdirTemp(workDir, "doc2docx-*")
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("create temp dir: %v", err))
			return
		}
		defer os.RemoveAll(sessionDir)

		inputPath := filepath.Join(sessionDir, filepath.Base(inputName))
		outPath := strings.TrimSuffix(inputPath, filepath.Ext(inputPath)) + ".docx"

		inBytes := &bytes.Buffer{}
		if _, err := io.Copy(inBytes, file); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("read upload: %v", err))
			return
		}

		if err := os.WriteFile(inputPath, inBytes.Bytes(), 0600); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("write temp: %v", err))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), cfg.timeout)
		defer cancel()

		if err := runLibreOffice(ctx, sessionDir, inputPath); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("convert failed: %v", err))
			return
		}

		docxBytes, err := os.ReadFile(outPath)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("read output: %v", err))
			return
		}

		respondJSON := cfg.defaultRespJSON || strings.Contains(r.Header.Get("Accept"), contentTypeJSON) || r.URL.Query().Get("format") == "json"
		if respondJSON {
			payload := map[string]string{
				"docxBase64": base64.StdEncoding.EncodeToString(docxBytes),
			}
			w.Header().Set("Content-Type", contentTypeJSON)
			_ = json.NewEncoder(w).Encode(payload)
			return
		}

		w.Header().Set("Content-Type", contentTypeDOCX)
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", safeOutName(inputName)))
		w.Header().Set("X-Content-SHA256", fmt.Sprintf("%x", sha256.Sum256(docxBytes)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(docxBytes)
	})

	server := &http.Server{
		Addr:              cfg.listenAddr,
		Handler:           logMiddleware(mwRecover(mwNoCache(mux))),
		ReadHeaderTimeout: 30 * time.Second,
	}
	log.Printf("doc2docx listening on %s", cfg.listenAddr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server error: %v", err)
	}
}

func runLibreOffice(ctx context.Context, outDir string, inputPath string) error {
	bin := "libreoffice"
	if _, err := exec.LookPath("soffice"); err == nil {
		bin = "soffice"
	}
	args := []string{"--headless", "--convert-to", "docx", "--outdir", outDir, inputPath}
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Stdout = &bytes.Buffer{}
	cmd.Stderr = &bytes.Buffer{}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("libreoffice run: %w; stdout=%s stderr=%s", err, cmd.Stdout, cmd.Stderr)
	}
	return nil
}

func validateBearer(authHeader string, expected string) bool {
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(authHeader)), "bearer ") {
		return false
	}
	token := strings.TrimSpace(authHeader[len("Bearer "):])
	return token == expected
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func safeOutName(in string) string {
	base := filepath.Base(in)
	base = strings.TrimSuffix(base, filepath.Ext(base)) + ".docx"
	return base
}

func mwNoCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		next.ServeHTTP(w, r)
	})
}

func mwRecover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				writeError(w, http.StatusInternalServerError, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &respWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(rw, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, rw.status, time.Since(start))
	})
}

type respWriter struct {
	http.ResponseWriter
	status int
}

func (w *respWriter) WriteHeader(statusCode int) {
	w.status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}
