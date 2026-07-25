package ieee

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/elecbug/crawlp/internal/provider"
)

const doiPrefix = "10.1109/"

type Downloader struct{}

func NewDownloader() *Downloader {
	return &Downloader{}
}

func (d *Downloader) ID() string {
	return "ieee"
}

func (d *Downloader) Name() string {
	return "IEEE Xplore"
}

func (d *Downloader) MatchDOI(doi string) bool {
	normalized := strings.ToLower(strings.TrimSpace(doi))
	return strings.HasPrefix(normalized, doiPrefix)
}

func (d *Downloader) Download(
	client *http.Client,
	doi string,
	outputDir string,
) (provider.Result, error) {
	info, err := resolveIEEEDocument(client, doi)
	if err != nil {
		return provider.Result{}, fmt.Errorf(
			"failed to resolve IEEE Xplore document: %w",
			err,
		)
	}

	outputPath, err := downloadIEEEPDF(client, info, outputDir)
	if err != nil {
		return provider.Result{}, fmt.Errorf(
			"failed to download IEEE Xplore document: %w",
			err,
		)
	}

	return provider.Result{
		ProviderID:   d.ID(),
		ProviderName: d.Name(),
		Title:        info.Title,
		Identifier:   info.ArticleNo,
		LandingURL:   info.LandingURL,
		OutputPath:   outputPath,
	}, nil
}
