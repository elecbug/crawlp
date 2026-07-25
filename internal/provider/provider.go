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
		client *http.Client,
		doi string,
		outputDir string,
	) (Result, error)
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

func (r *Router) Resolve(doi string) (Downloader, error) {
	for _, downloader := range r.downloaders {
		if downloader.MatchDOI(doi) {
			return downloader, nil
		}
	}

	return nil, fmt.Errorf(
		"no downloader is registered for DOI: %s",
		doi,
	)
}
