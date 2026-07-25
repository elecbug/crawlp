package provider

import (
	"fmt"
	"net/http"
	"strings"
)

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
	for _, downloader := range r.downloaders {
		if downloader.MatchDOI(doi) {
			return downloader, nil
		}
	}

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

// Downloaders returns a copy of all registered downloaders.
func (r *Router) Downloaders() []Downloader {
	result := make([]Downloader, len(r.downloaders))
	copy(result, r.downloaders)

	return result
}

// FindByID returns a downloader using its provider ID.
func (r *Router) FindByID(id string) (Downloader, bool) {
	normalized := strings.ToLower(strings.TrimSpace(id))

	for _, downloader := range r.downloaders {
		if strings.EqualFold(downloader.ID(), normalized) {
			return downloader, true
		}
	}

	return nil, false
}
