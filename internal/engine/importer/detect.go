package importer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fblazt/boomerang/internal/engine/storage"
	"github.com/fblazt/boomerang/internal/models"
)

// Service implements models.ImporterService for OpenAPI 3.1, Postman v2.1, Bruno, and cURL.
type Service struct{}

// NewService constructs a universal API importer service.
func NewService() *Service {
	return &Service{}
}

// DetectFormat inspects raw text input and determines whether it matches cURL, Postman v2.1, OpenAPI 3.x, or Bruno.
func (s *Service) DetectFormat(input string) models.ImportFormat {
	return DetectFormat(input)
}

// DetectFormat inspects raw text input and returns its detected format.
func DetectFormat(input string) models.ImportFormat {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return models.ImportFormatUnknown
	}
	lower := strings.ToLower(trimmed)

	// 1. cURL command
	if strings.HasPrefix(lower, "curl ") || strings.HasPrefix(lower, "curl\t") || strings.HasPrefix(lower, "curl\\") {
		return models.ImportFormatCurl
	}

	// 2. Postman Collection v2.1 (JSON containing schema.getpostman.com or postman_id + item)
	if strings.HasPrefix(trimmed, "{") &&
		(strings.Contains(lower, "schema.getpostman.com") ||
			strings.Contains(lower, "_postman_id") ||
			(strings.Contains(lower, `"info"`) && strings.Contains(lower, `"item"`))) {
		return models.ImportFormatPostman
	}

	// 3. OpenAPI 3.x / Swagger JSON or YAML
	if (strings.Contains(lower, `"openapi"`) && strings.Contains(lower, `"paths"`)) ||
		(strings.Contains(lower, "openapi:") && strings.Contains(lower, "paths:")) {
		return models.ImportFormatOpenAPI
	}

	// 4. Bruno (.bru) DSL blocks
	if strings.Contains(lower, "meta {") ||
		hasAnyLinePrefix(lower, []string{"get {", "post {", "put {", "patch {", "delete {", "headers {", "body:json {"}) {
		return models.ImportFormatBruno
	}

	return models.ImportFormatUnknown
}

// Parse parses raw input (file contents or pasted snippet) into an ImportPreview.
func (s *Service) Parse(input string, sourceHint string) (models.ImportPreview, error) {
	format := DetectFormat(input)
	if format == models.ImportFormatUnknown && strings.HasSuffix(strings.ToLower(sourceHint), ".bru") {
		format = models.ImportFormatBruno
	}
	if format == models.ImportFormatUnknown {
		return models.ImportPreview{}, fmt.Errorf("could not detect format (supported: OpenAPI 3.1, Postman v2.1, Bruno .bru, cURL)")
	}

	var (
		col      models.Collection
		warnings []string
		err      error
	)

	now := time.Now().UTC()
	switch format {
	case models.ImportFormatCurl:
		col, warnings, err = parseCurlCollection(input, sourceHint, now)
	case models.ImportFormatPostman:
		col, warnings, err = parsePostmanCollection(input, now)
	case models.ImportFormatOpenAPI:
		col, warnings, err = parseOpenAPICollection(input, sourceHint, now)
	case models.ImportFormatBruno:
		col, warnings, err = parseBrunoCollection(input, sourceHint, now)
	}
	if err != nil {
		return models.ImportPreview{}, err
	}

	if col.ID == "" {
		col.ID = storage.NewID("col")
	}
	if strings.TrimSpace(col.Name) == "" {
		col.Name = "Imported Collection"
	}
	if col.CreatedAt.IsZero() {
		col.CreatedAt = now
	}
	col.UpdatedAt = now

	folders, reqs := countTreeTotals(col)
	return models.ImportPreview{
		Format:       format,
		FormatLabel:  format.DisplayName(),
		Collection:   col,
		RequestCount: reqs,
		FolderCount:  folders,
		Warnings:     warnings,
	}, nil
}

// ParseFile reads a file or Bruno directory from disk and returns its parsed ImportPreview.
func (s *Service) ParseFile(path string) (models.ImportPreview, error) {
	info, err := os.Stat(path)
	if err != nil {
		return models.ImportPreview{}, err
	}
	if info.IsDir() {
		return parseBrunoDirectory(path)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return models.ImportPreview{}, err
	}
	return s.Parse(string(raw), filepath.Base(path))
}

func hasAnyLinePrefix(lower string, prefixes []string) bool {
	for _, line := range strings.Split(lower, "\n") {
		t := strings.TrimSpace(line)
		for _, p := range prefixes {
			if strings.HasPrefix(t, p) {
				return true
			}
		}
	}
	return false
}

func countTreeTotals(col models.Collection) (folders int, requests int) {
	requests += len(col.Items)
	fCount, rCount := countFolderTree(col.Folders)
	return folders + fCount, requests + rCount
}

func countFolderTree(folders []models.Folder) (int, int) {
	totalF := len(folders)
	totalR := 0
	for _, f := range folders {
		totalR += len(f.Items)
		subF, subR := countFolderTree(f.Folders)
		totalF += subF
		totalR += subR
	}
	return totalF, totalR
}
