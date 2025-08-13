package converter

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/allex/fconv/common"
)

// LibreOfficeConverter implements conversions via libreoffice/soffice.
type LibreOfficeConverter struct {
	// allowedTargets is a set of allowed output extensions (lowercase, without dot).
	allowedTargets map[string]struct{}
}

func NewLibreOfficeConverter() *LibreOfficeConverter {
	return &LibreOfficeConverter{
		allowedTargets: map[string]struct{}{
			"docx": {},
			"pdf":  {},
			"odt":  {},
			"rtf":  {},
			"txt":  {},
			"html": {},
		},
	}
}

func (l *LibreOfficeConverter) Name() string { return "libreoffice" }

func (l *LibreOfficeConverter) Accepts(inputName string, targetExt string) bool {
	if _, ok := l.allowedTargets[strings.ToLower(strings.TrimPrefix(targetExt, "."))]; !ok {
		return false
	}
	// LibreOffice supports many input formats. We do a light filter to common doc types, but allow broader usage.
	inExt := strings.ToLower(strings.TrimPrefix(filepath.Ext(inputName), "."))
	supportedInputs := map[string]struct{}{
		"doc": {}, "docx": {}, "rtf": {}, "odt": {}, "txt": {}, "html": {}, "htm": {}, "wps": {}, "wpd": {}, "xml": {},
	}
	if _, ok := supportedInputs[inExt]; ok {
		return true
	}
	// fallback: try anyway
	return true
}

func (l *LibreOfficeConverter) Convert(ctx context.Context, outDir string, inputPath string, targetExt string) (string, string, error) {
	outPath, err := runLibreOffice(ctx, outDir, inputPath, targetExt)
	if err != nil {
		return "", "", err
	}
	return outPath, extToContentType(targetExt), nil
}

func extToContentType(ext string) string {
	switch strings.ToLower(strings.TrimPrefix(ext, ".")) {
	case "docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case "pdf":
		return "application/pdf"
	case "odt":
		return "application/vnd.oasis.opendocument.text"
	case "rtf":
		return "application/rtf"
	case "txt":
		return "text/plain; charset=utf-8"
	case "html":
		return "text/html; charset=utf-8"
	default:
		return "application/octet-stream"
	}
}

func runLibreOffice(ctx context.Context, outDir string, inputPath string, targetExt string) (string, error) {
	bin := "libreoffice"
	if _, err := exec.LookPath("soffice"); err == nil {
		bin = "soffice"
	}
	args := []string{"--headless", "--convert-to", targetExt, "--outdir", outDir, inputPath}
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Stdout = &bytes.Buffer{}
	cmd.Stderr = &bytes.Buffer{}
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("libreoffice run: %w; stdout=%s stderr=%s", err, cmd.Stdout, cmd.Stderr)
	}
	// Determine output path based on requested extension
	base := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	return filepath.Join(outDir, base+"."+strings.ToLower(strings.TrimPrefix(targetExt, "."))), nil
}

func init() {
	common.RegisterConverter(NewLibreOfficeConverter())
}
