package provider

import (
	"fmt"
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

type Router struct {
	downloaders []Downloader
}

func NewRouter(downloaders ...Downloader) *Router {
	copied := make([]Downloader, len(downloaders))
	copy(copied, downloaders)

	return &Router{
		downloaders: copied,
	}
}

func (r *Router) Resolve(
	cli *http.Client,
	doi string,
) (Downloader, error) {
	// First, try deterministic DOI-prefix matches.
	for _, downloader := range r.downloaders {
		if downloader.MatchDOI(doi) {
			return downloader, nil
		}
	}

	// Then, try providers that require a network lookup.
	for _, downloader := range r.downloaders {
		prober, ok := downloader.(Prober)
		if !ok {
			continue
		}

		matched, err := prober.Probe(cli, doi)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to probe %s: %w",
				downloader.Name(),
				err,
			)
		}

		if matched {
			return downloader, nil
		}
	}

	return nil, fmt.Errorf(
		"no downloader is registered for DOI: %s",
		doi,
	)
}
