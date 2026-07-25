package app

import (
	"fmt"
	"path/filepath"

	"github.com/elecbug/crawlp/internal/app/client"
	"github.com/elecbug/crawlp/internal/ieee"
	"github.com/elecbug/crawlp/internal/paper"
)

func Run(opts Options) error {
	if opts.Interactive {
		PrintBanner()
		fmt.Println("This program uses your current network access rights.")
		fmt.Println("It does not bypass IEEE Xplore authentication or licensing.")
		fmt.Println()
	}

	doi, err := paper.NormalizeDOI(opts.DOI)
	if err != nil {
		return err
	}

	client, err := client.NewHTTPClient(opts.Timeout)
	if err != nil {
		return fmt.Errorf("failed to create HTTP client: %w", err)
	}

	fmt.Printf("Resolving DOI: %s\n", doi)

	info, err := ieee.ResolveIEEEDocument(client, doi)
	if err != nil {
		return err
	}

	fmt.Printf("Title: %s\n", info.Title)
	fmt.Printf("Article number: %s\n", info.ArticleNo)
	fmt.Printf("Landing page: %s\n", info.LandingURL)

	outputPath, err := ieee.DownloadIEEEPDF(client, info, opts.OutputDir)
	if err != nil {
		return err
	}

	absolutePath, err := filepath.Abs(outputPath)
	if err == nil {
		outputPath = absolutePath
	}

	fmt.Println()
	fmt.Printf("Download completed: %s\n", outputPath)
	return nil
}
