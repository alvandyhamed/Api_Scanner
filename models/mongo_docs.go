package models

import "time"

type SiteDoc struct {
	ID           string    `bson:"_id"`
	DisplayURL   string    `bson:"display_url,omitempty"`
	Hosts        []string  `bson:"hosts,omitempty"`
	CreatedAt    time.Time `bson:"created_at,omitempty"`
	UpdatedAt    time.Time `bson:"updated_at,omitempty"`
	LastScanAt   time.Time `bson:"last_scan_at,omitempty"`
	PagesCount   int64     `bson:"pages_count,omitempty"`
	EndpointsCnt int64     `bson:"endpoints_count,omitempty"`
}

type PageDoc struct {
	SiteID     string    `bson:"site_id"`
	URL        string    `bson:"url"`
	URLNorm    string    `bson:"url_norm"`
	Scheme     string    `bson:"scheme"`
	Host       string    `bson:"host"`
	Path       string    `bson:"path"`
	Resources  []string  `bson:"resources,omitempty"`
	ScriptURLs []string  `bson:"script_urls,omitempty"`
	Endpoints  []string  `bson:"endpoints,omitempty"`
	ScannedAt  time.Time `bson:"scanned_at"`
	CreatedAt  time.Time `bson:"created_at,omitempty"`
}

type EndpointDoc struct {
	SiteID     string    `bson:"site_id"`
	Endpoint   string    `bson:"endpoint"`
	FirstSeen  time.Time `bson:"first_seen,omitempty"`
	LastSeen   time.Time `bson:"last_seen,omitempty"`
	Hosts      []string  `bson:"hosts,omitempty"`
	SourceURLs []string  `bson:"source_urls,omitempty"`
	SeenCount  int64     `bson:"seen_count,omitempty"`
}
