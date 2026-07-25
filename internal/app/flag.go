package app

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/elecbug/crawlp/internal/config"
)

type Flags struct {
	MainArgs       []string
	OutputDir      string
	TimeoutSeconds int
	AutoProvider   bool
	NArg           int
}

func ParseFlags() *Flags {
	outputDirF := flag.String(
		"o",
		config.DEFAULT_OUTPUT_DIR,
		"Output directory for downloaded PDF files",
	)

	timeoutSecondsF := flag.Int(
		"timeout",
		config.DEFAULT_TIMEOUT,
		"HTTP timeout in seconds",
	)

	autoProviderF := flag.Bool(
		"auto-provider",
		false,
		"Use the automatically detected provider without confirmation",
	)

	flag.Usage = printUsage
	flag.Parse()

	return &Flags{
		MainArgs:       flag.Args(),
		OutputDir:      *outputDirF,
		TimeoutSeconds: *timeoutSecondsF,
		AutoProvider:   *autoProviderF,
		NArg:           flag.NArg(),
	}
}

func printUsage() {
	executable := filepath.Base(os.Args[0])

	fmt.Fprintf(
		flag.CommandLine.Output(),
		`%s

Usage:
  %s [options] <DOI>
  %s [options]

Modes:
  Command mode:
    Provide a DOI as a command-line argument.

  Interactive mode:
    Run the program without a DOI argument. The program will prompt for
    a DOI, an output directory, and an HTTP timeout.

Provider selection:
  By default, the program detects a provider from the DOI and asks you
  to confirm or change the selected provider.

  Use -auto-provider to accept the automatically detected provider
  without displaying a provider selection prompt.

Options:
`,
		config.APP_NAME,
		executable,
		executable,
	)

	flag.PrintDefaults()

	fmt.Fprintf(
		flag.CommandLine.Output(),
		`
Examples:
  Interactive mode:
    %s

  Download using provider confirmation:
    %s 10.1109/icbc67748.2026.11575499

  Automatically use the detected provider:
    %s -auto-provider 10.1109/icbc67748.2026.11575499

  Specify an output directory:
    %s -o papers 10.1109/icbc67748.2026.11575499

  Specify an HTTP timeout:
    %s -timeout 120 -o papers 10.1109/icbc67748.2026.11575499

  Use a complete DOI URL:
    %s -auto-provider https://doi.org/10.1109/icbc67748.2026.11575499
`,
		executable,
		executable,
		executable,
		executable,
		executable,
		executable,
	)
}
