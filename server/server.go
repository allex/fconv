package server

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/allex/fconv/common"
	_ "github.com/allex/fconv/converter/libreoffice"
	"github.com/allex/fconv/pkgs/util"
)

func Start() *http.Server {
	cfg := loadConfig()

	server := newServer(cfg)
	fmt.Printf("Starting server on %s\n", cfg.listenAddr)

	return server
}

func newServer(cfg config) *http.Server {
	// Gin setup
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("GIN_MODE")))
	switch mode {
	case "debug":
		gin.SetMode(gin.DebugMode)
	case "test":
		gin.SetMode(gin.TestMode)
	default:
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	fmt.Printf("Gin mode: %s\n", gin.Mode())

	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(noCacheMiddleware())

	r.GET("/healthz", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// Generic conversion endpoint for doc->docx. Usage: POST /api/v1/convert/doc2docx with form file field "file".
	r.POST(apiPrefix+"/convert/doc2docx", authMiddleware(cfg), makeConvertHandler(cfg, "docx"))

	return &http.Server{
		Addr:              cfg.listenAddr,
		Handler:           r,
		ReadHeaderTimeout: 30 * time.Second,
	}
}

func makeConvertHandler(cfg config, forcedTargetExt string) gin.HandlerFunc {
	return func(c *gin.Context) {
		fileHeader, err := c.FormFile(formFileFieldName)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("missing form file field '%s'", formFileFieldName)})
			return
		}

		// detect target extension, default to docx (allow shortcut via forcedTargetExt)
		rawTarget := forcedTargetExt
		if rawTarget == "" {
			rawTarget = c.DefaultQuery("to", "docx")
		}
		targetExt := strings.ToLower(strings.TrimPrefix(rawTarget, "."))
		if targetExt == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "missing target format: provide ?to=<ext>"})
			return
		}

		// prepare working dir
		workDir := cfg.tmpDir
		if workDir == "" {
			workDir = os.TempDir()
		}
		sessionDir, err := os.MkdirTemp(workDir, "doc2docx-*")
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("create temp dir: %v", err)})
			return
		}
		defer os.RemoveAll(sessionDir)

		inputName := fileHeader.Filename
		inputPath := filepath.Join(sessionDir, filepath.Base(inputName))
		if err := c.SaveUploadedFile(fileHeader, inputPath); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("save upload: %v", err)})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), cfg.timeout)
		defer cancel()

		conv, err := common.SelectConverter(inputName, targetExt)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		outPath, outCT, err := conv.Convert(ctx, sessionDir, inputPath, targetExt)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("convert failed: %v", err)})
			return
		}

		outBytes, err := os.ReadFile(outPath)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("read output: %v", err)})
			return
		}

		respondJSON := strings.Contains(c.GetHeader("Accept"), contentTypeJSON) || c.Query("format") == "json"
		if respondJSON {
			c.Header("Content-Type", contentTypeJSON)
			c.JSON(http.StatusOK, gin.H{"base64": base64.StdEncoding.EncodeToString(outBytes)})
			return
		}

		c.Header("Content-Type", outCT)
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", util.SafeOutNameWithExt(inputName, targetExt)))
		if cfg.enableSHA256Header {
			c.Header("X-Content-SHA256", fmt.Sprintf("%x", sha256.Sum256(outBytes)))
		}
		c.Status(http.StatusOK)
		_, _ = c.Writer.Write(outBytes)
	}
}

func noCacheMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")
		c.Next()
	}
}

func authMiddleware(cfg config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cfg.authKey != "" {
			if !util.ValidateBearer(c.GetHeader(headerAuth), cfg.authKey) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
				return
			}
		}
		c.Next()
	}
}
