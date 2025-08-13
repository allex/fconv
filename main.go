package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/allex/fconv/server"
)

// Version information populated via -ldflags during build
var (
	appVersion = "dev"
	gitCommit  = "unknown"
	buildTime  = ""
)

const helpText = `fconv - File conversion server

Usage:
  fconv [--help | -h | help | --version | -v | version]

Description:
  Runs an HTTP server for file format conversions using LibreOffice and pluggable converters.

HTTP API:
  GET  /healthz
       Health check endpoint. Returns "ok" when the server is healthy.

  POST /api/v1/doc2docx
       Multipart form upload with a single file field named "file".
       - Converts the uploaded .doc to .docx
       - Default response: application/vnd.openxmlformats-officedocument.wordprocessingml.document
       - JSON response: set header "Accept: application/json" or query "?format=json" to receive {"base64": "..."}

Environment:
  FCONV_LISTEN_ADDR   Server listen address (default ":8080")
  FCONV_AUTH_KEY      Optional bearer token required for requests (Authorization: Bearer <key>)
  FCONV_TIMEOUT       Conversion timeout (default 10m). Examples: "30s", "5m", "1h"
  FCONV_TMPDIR        Override temporary working directory
  FCONV_ENABLE_SHA256 If true, include X-Content-SHA256 header in binary responses (default true)
  GIN_MODE               One of: debug | test | release (default release)

Examples:
  Binary DOCX response:
    curl -sS -X POST 'http://localhost:8080/api/v1/doc2docx' \
      -F 'file=@/path/to/input.doc' \
      -o output.docx

  JSON response (base64 -> docx):
    curl -sS -H 'Accept: application/json' -X POST 'http://localhost:8080/api/v1/doc2docx' \
      -F 'file=@/path/to/input.doc' | jq -r .base64 | base64 --decode > output.docx
`

func main() {
	if len(os.Args) > 1 {
		arg := os.Args[1]
		if arg == "--help" || arg == "-h" || arg == "help" {
			fmt.Print(helpText)
			return
		}
	}

	s := server.Start()
	if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(fmt.Errorf("server error: %w", err))
	}
}
