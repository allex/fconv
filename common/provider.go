package common

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
)

// Converter abstracts a conversion engine implementation.
type Converter interface {
	// Name returns a unique name for the converter implementation.
	Name() string
	// Accepts returns whether this converter can handle the given input file name and target extension (without dot).
	Accepts(inputName string, targetExt string) bool
	// Convert performs the conversion. targetExt is without dot (e.g. "docx", "pdf").
	// It must write the output file to outDir and return its absolute path and content-type.
	Convert(ctx context.Context, outDir string, inputPath string, targetExt string) (string, string, error)
}

var (
	providerMu sync.RWMutex
	providers  []Converter
)

// RegisterConverter adds a converter to the global registry.
func RegisterConverter(converter Converter) {
	providerMu.Lock()
	defer providerMu.Unlock()
	providers = append(providers, converter)
}

// SelectConverter returns the first converter that accepts the given input and target.
func SelectConverter(inputName string, targetExt string) (Converter, error) {
	providerMu.RLock()
	defer providerMu.RUnlock()
	for _, provider := range providers {
		if provider.Accepts(inputName, targetExt) {
			return provider, nil
		}
	}
	return nil, fmt.Errorf("no converter available for %q -> %q", filepath.Ext(inputName), targetExt)
}

// ListConverters returns registered converter names in order.
func ListConverters() []string {
	providerMu.RLock()
	defer providerMu.RUnlock()
	names := make([]string, 0, len(providers))
	for _, p := range providers {
		names = append(names, p.Name())
	}
	return names
}
