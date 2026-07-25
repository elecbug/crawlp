package provider

import (
	"errors"
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

	var probeErrors []error

	for _, downloader := range r.downloaders {
		prober, ok := downloader.(Prober)
		if !ok {
			continue
		}

		matched, err := prober.Probe(cli, doi)
		if err != nil {
			probeErrors = append(
				probeErrors,
				fmt.Errorf(
					"%s probe failed: %w",
					downloader.Name(),
					err,
				),
			)

			continue
		}

		if matched {
			return downloader, nil
		}
	}

	baseErr := fmt.Errorf(
		"no downloader is registered for DOI: %s",
		doi,
	)

	if len(probeErrors) == 0 {
		return nil, baseErr
	}

	return nil, errors.Join(
		append(
			[]error{baseErr},
			probeErrors...,
		)...,
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
