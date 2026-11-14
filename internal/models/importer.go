package models

// ImportFormat identifies supported API import formats.
type ImportFormat string

const (
	ImportFormatUnknown ImportFormat = ""
	ImportFormatCurl    ImportFormat = "curl"
	ImportFormatPostman ImportFormat = "postman"
	ImportFormatOpenAPI ImportFormat = "openapi"
	ImportFormatBruno   ImportFormat = "bruno"
)

// FormatDisplayName returns the human-readable badge label for fmtName.
func (f ImportFormat) DisplayName() string {
	switch f {
	case ImportFormatCurl:
		return "cURL"
	case ImportFormatPostman:
		return "Postman v2.1"
	case ImportFormatOpenAPI:
		return "OpenAPI 3.1"
	case ImportFormatBruno:
		return "Bruno (.bru)"
	default:
		return "Unknown"
	}
}

// ImportPreview captures the parsed collection and summary metrics ready to confirm import.
type ImportPreview struct {
	Format       ImportFormat `json:"format"`
	FormatLabel  string       `json:"format_label"`
	Collection   Collection   `json:"collection"`
	RequestCount int          `json:"request_count"`
	FolderCount  int          `json:"folder_count"`
	Warnings     []string     `json:"warnings,omitempty"`
}

// ImporterService parses external API formats into Boomerang collections.
type ImporterService interface {
	DetectFormat(input string) ImportFormat
	Parse(input string, sourceHint string) (ImportPreview, error)
	ParseFile(path string) (ImportPreview, error)
}
