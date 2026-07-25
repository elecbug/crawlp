package provider

import (
	"net/http"
)

type Result struct {
	ProviderID   string
	ProviderName string
	Title        string
	Identifier   string
	LandingURL   string
	OutputPath   string
}

type Downloader interface {
	ID() string
	Name() string
	MatchDOI(doi string) bool

	Download(
		cli *http.Client,
		doi string,
		outputDir string,
	) (Result, error)
}

// Prober is implemented by providers that cannot be identified
// using a deterministic DOI prefix.
type Prober interface {
	Probe(
		cli *http.Client,
		doi string,
	) (bool, error)
}
