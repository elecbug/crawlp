package elsevier

import (
	"errors"
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
	return "elsevier"
}

func (d *Downloader) Name() string {
	return "Elsevier ScienceDirect"
}

func (d *Downloader) MatchDOI(doi string) bool {
	normalized := strings.ToLower(
		strings.TrimSpace(doi),
	)

	return strings.HasPrefix(
		normalized,
		elsevierDOIPrefix,
	)
}

func (d *Downloader) Probe(
	cli *http.Client,
	doi string,
) (bool, error) {
	_, err := resolveElsevierDocument(cli, doi)

	if err == nil {
		return true, nil
	}

	if errors.Is(err, ErrNotElsevier) {
		return false, nil
	}

	return false, err
}

func (d *Downloader) Download(
	cli *http.Client,
	doi string,
	outputDir string,
) (provider.Result, error) {
	info, err := resolveElsevierDocument(cli, doi)
	if err != nil {
		return provider.Result{}, fmt.Errorf(
			"failed to resolve Elsevier document: %w",
			err,
		)
	}

	outputPath, err := downloadElsevierPDF(
		cli,
		info,
		outputDir,
	)
	if err != nil {
		return provider.Result{}, fmt.Errorf(
			"failed to download Elsevier document: %w",
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
