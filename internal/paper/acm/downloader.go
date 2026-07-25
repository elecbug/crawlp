package acm

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/elecbug/crawlp/internal/provider"
)

const acmDOIPrefix = "10.1145/"

type Downloader struct{}

func NewDownloader() *Downloader {
	return &Downloader{}
}

func (d *Downloader) ID() string {
	return "acm"
}

func (d *Downloader) Name() string {
	return "ACM Digital Library"
}

func (d *Downloader) MatchDOI(doi string) bool {
	normalized := strings.ToLower(
		strings.TrimSpace(doi),
	)

	return strings.HasPrefix(
		normalized,
		acmDOIPrefix,
	)
}

func (d *Downloader) Download(
	cli *http.Client,
	doi string,
	outputDir string,
) (provider.Result, error) {
	info, err := resolveACMDocument(cli, doi)
	if err != nil {
		return provider.Result{}, fmt.Errorf(
			"failed to resolve ACM document: %w",
			err,
		)
	}

	outputPath, err := downloadACMPDF(
		cli,
		info,
		outputDir,
	)
	if err != nil {
		return provider.Result{}, fmt.Errorf(
			"failed to download ACM document: %w",
			err,
		)
	}

	return provider.Result{
		ProviderID:   d.ID(),
		ProviderName: d.Name(),
		Title:        info.Title,
		Identifier:   info.Identifier,
		LandingURL:   info.LandingURL,
		OutputPath:   outputPath,
	}, nil
}
