package main

import (
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/allex/fconv/pkgs/util"
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
  fconv [-help | -h | -version | -v]
  fconv [-host HOST] [-port PORT]

Description:
  Runs an HTTP server for file format conversions using LibreOffice and pluggable converters.

Options:
  -host HOST            Host/IP to bind (default 0.0.0.0)
  -port PORT            Port to listen on (default 8080)
  -help, -h             Show help info
  -version, -v          Show version

Environment:
  FCONV_LISTEN_ADDR     Server listen address (default ":8080"). Examples: ":8081", "127.0.0.1:8080", "0.0.0.0:9090"
  FCONV_PORT            Shortcut to set port (e.g. 8081). Ignored if FCONV_LISTEN_ADDR is set
  FCONV_AUTH_KEY        Optional bearer token required for requests (Authorization: Bearer <key>)
  FCONV_TIMEOUT         Conversion timeout (default 10m). Examples: "30s", "5m", "1h"
  FCONV_TMPDIR          Override temporary working directory
  FCONV_ENABLE_SHA256   If true, include X-Content-SHA256 header in binary responses (default true)
  GIN_MODE              One of: debug | test | release (default release)

HTTP API:
  GET  /healthz
       Health check endpoint. Returns "ok" when the server is healthy.

  POST /api/v1/doc2docx
       Multipart form upload with a single file field named "file".
       - Converts the uploaded .doc to .docx
       - Default response: application/vnd.openxmlformats-officedocument.wordprocessingml.document
       - JSON response: set header "Accept: application/json" or query "?format=json" to receive {"base64": "..."}

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
	defer handlePanic()

	// Configure flag usage to show the custom help text
	flag.Usage = func() {
		fmt.Print(helpText)
	}

	// Flags
	var (
		hostFlag    string
		portFlag    int
		showHelp    bool
		showVersion bool
	)

	flag.StringVar(&hostFlag, "host", "", "Host/IP to bind (default 0.0.0.0)")
	flag.IntVar(&portFlag, "port", 0, "Port to listen on (default 8080)")
	flag.BoolVar(&showHelp, "help", false, "Show help")
	flag.BoolVar(&showHelp, "h", false, "Show help (shorthand)")
	flag.BoolVar(&showVersion, "version", false, "Show version")
	flag.BoolVar(&showVersion, "v", false, "Show version (shorthand)")
	flag.Parse()

	if showHelp {
		fmt.Print(helpText)
		return
	}
	if showVersion {
		fmt.Printf("fconv %s (rev %s) %s\n", appVersion, gitCommit, buildTime)
		return
	}

	// If either host or port provided, construct FCONV_LISTEN_ADDR accordingly
	if hostFlag != "" || portFlag != 0 {
		if !util.IsValidPort(portFlag) {
			fmt.Fprintln(os.Stderr, "error: -port must be a positive integer (1-65535)")
			os.Exit(2)
		}
		addr := ""
		if hostFlag != "" && portFlag != 0 {
			addr = net.JoinHostPort(hostFlag, fmt.Sprintf("%d", portFlag))
		} else if hostFlag != "" { // host only => default to 8080
			addr = net.JoinHostPort(hostFlag, "8080")
		} else { // port only
			addr = fmt.Sprintf(":%d", portFlag)
		}
		_ = os.Setenv("FCONV_LISTEN_ADDR", addr)
	}

	if err := server.Start(); err != nil {
		panic(fmt.Errorf("error: %w", err))
	}
}

func handlePanic() {
	if r := recover(); r != nil {
		fmt.Fprintln(os.Stderr, r)
		os.Exit(1)
	}
}
