package app

import (
	"fmt"
	"path/filepath"

	httpclient "github.com/elecbug/crawlp/internal/app/client"
	"github.com/elecbug/crawlp/internal/paper"
	"github.com/elecbug/crawlp/internal/registry"
)

func Run(opts Options) error {
	if opts.Interactive {
		PrintBanner()
		fmt.Println("This program uses your current network access rights.")
		fmt.Println("It does not bypass publisher authentication or licensing.")
		fmt.Println()
	}

	doi, err := paper.NormalizeDOI(opts.DOI)
	if err != nil {
		return err
	}

	httpClient, err := httpclient.NewHTTPClient(opts.Timeout)
	if err != nil {
		return fmt.Errorf(
			"failed to create HTTP client: %w",
			err,
		)
	}

	fmt.Printf("Resolving DOI: %s\n", doi)

	router := registry.NewDefaultRouter()

	detected, detectionErr := router.Resolve(
		httpClient,
		doi,
	)

	downloader, err := selectDownloaderInteractively(
		router,
		detected,
		detectionErr,
		opts.AutoProvider,
	)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("Selected provider: %s\n", downloader.Name())
	fmt.Printf(
		"Forwarding request to the %s downloader.\n",
		downloader.Name(),
	)

	result, err := downloader.Download(
		httpClient,
		doi,
		opts.OutputDir,
	)
	if err != nil {
		return fmt.Errorf(
			"%s downloader failed: %w",
			downloader.Name(),
			err,
		)
	}

	outputPath := result.OutputPath

	absolutePath, err := filepath.Abs(outputPath)
	if err == nil {
		outputPath = absolutePath
	}

	fmt.Printf("Title: %s\n", result.Title)

	if result.Identifier != "" {
		fmt.Printf(
			"Provider identifier: %s\n",
			result.Identifier,
		)
	}

	if result.LandingURL != "" {
		fmt.Printf(
			"Landing page: %s\n",
			result.LandingURL,
		)
	}

	fmt.Println()
	fmt.Printf("Download completed: %s\n", outputPath)

	return nil
}
