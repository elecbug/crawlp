package springer

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/elecbug/crawlp/internal/provider"
)

type Downloader struct{}

func NewDownloader() *Downloader {
	return &Downloader{}
}

func (d *Downloader) ID() string {
	return "springer"
}

func (d *Downloader) Name() string {
	return "Springer Nature Link"
}

func (d *Downloader) MatchDOI(doi string) bool {
	normalized := strings.ToLower(
		strings.TrimSpace(doi),
	)

	for _, prefix := range springerDOIPrefixies {

		if strings.HasPrefix(normalized, prefix) {
			return true
		}
	}

	return false
}

func (d *Downloader) Download(
	cli *http.Client,
	doi string,
	outputDir string,
) (provider.Result, error) {
	info, err := resolveSpringerDocument(cli, doi)
	if err != nil {
		return provider.Result{}, fmt.Errorf(
			"failed to resolve Springer document: %w",
			err,
		)
	}

	outputPath, err := downloadSpringerPDF(
		cli,
		info,
		outputDir,
	)
	if err != nil {
		return provider.Result{}, fmt.Errorf(
			"failed to download Springer document: %w",
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
