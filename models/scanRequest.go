package models

type ScanRequest struct {
	URL            string `json:"url"`
	WaitSec        int    `json:"wait_sec,omitempty"`
	JSFetchTimeout int    `json:"js_fetch_timeout,omitempty"`
}
