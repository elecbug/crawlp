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
	NArg           int
}

func ParseFlags() *Flags {
	outputDirF := flag.String("o", config.DEFAULT_OUTPUT_DIR, "Output directory for downloaded PDF files")
	timeoutSecondsF := flag.Int("timeout", config.DEFAULT_TIMEOUT, "HTTP timeout in seconds")
	flag.Usage = printUsage
	flag.Parse()

	return &Flags{
		MainArgs:       flag.Args(),
		OutputDir:      *outputDirF,
		TimeoutSeconds: *timeoutSecondsF,
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
  %s

Modes:
  Command mode:
    Provide a DOI as a command-line argument.

  Interactive mode:
    Run the program without arguments. The program will prompt for a DOI,
    an output directory, and a timeout.

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
  %s 10.1109/JSSC.2018.2824300
  %s -o papers 10.1109/JSSC.2018.2824300
  %s -timeout 120 -o papers https://doi.org/10.1109/JSSC.2018.2824300
`,
		executable,
		executable,
		executable,
	)
}
