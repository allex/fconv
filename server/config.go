package server

import (
	"os"
	"regexp"
	"time"

	"github.com/allex/fconv/pkgs/util"
)

// config represents runtime configuration derived from env vars
// FCONV_AUTH_KEY: optional bearer token required for requests
// FCONV_LISTEN_ADDR: server listen address, default :8080
// FCONV_TIMEOUT: conversion timeout (default 10m), Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
// FCONV_TMPDIR: override temp working directory
// FCONV_ENABLE_SHA256: if true, include X-Content-SHA256 header for binary responses (defaults to true)
//
// Response format can be controlled via Accept: application/json or query ?format=json

const (
	apiPrefix         = "/api/v1"
	defaultListenAddr = ":8080"
	defaultTimeout    = 10 * time.Minute // 10m
	formFileFieldName = "file"
	headerAuth        = "Authorization"
	contentTypeJSON   = "application/json"
)

type config struct {
	listenAddr         string
	authKey            string
	timeout            time.Duration
	tmpDir             string
	enableSHA256Header bool
}

// load server config based on env
func loadConfig() config {
	addr := util.Getenv("FCONV_LISTEN_ADDR", defaultListenAddr)
	if regexp.MustCompile(`^\d+$`).MatchString(addr) {
		addr = ":" + addr
	}
	cfg := config{
		listenAddr:         addr,
		authKey:            os.Getenv("FCONV_AUTH_KEY"),
		tmpDir:             os.Getenv("FCONV_TMPDIR"),
		timeout:            defaultTimeout,
		enableSHA256Header: util.EnvBool("FCONV_ENABLE_SHA256", true),
	}
	if v := os.Getenv("FCONV_TIMEOUT"); v != "" {
		if ms, err := time.ParseDuration(v); err == nil {
			cfg.timeout = ms
		}
	}
	return cfg
}
