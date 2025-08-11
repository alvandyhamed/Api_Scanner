package models

type ScanResponse struct {
	URL          string   `json:"url"`
	Resources    []string `json:"resources"`
	UniquePaths  []string `json:"unique_paths"`
	AllScripts   []string `json:"script_urls"`
	Errors       []string `json:"errors,omitempty"`
	ProcessedAt  string   `json:"processed_at"`
	PageDuration string   `json:"page_duration"`
}
