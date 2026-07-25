package app

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

type Options struct {
	DOI           string
	OutputDir     string
	Timeout       time.Duration
	Interactive   bool
	PauseOnFinish bool
}

func ParseOptions(flags *Flags) (Options, error) {
	var opts Options

	opts.OutputDir = flags.OutputDir
	opts.Timeout = time.Duration(flags.TimeoutSeconds) * time.Second

	if flags.TimeoutSeconds <= 0 {
		return opts, errors.New("timeout must be greater than zero")
	}

	switch flags.NArg {
	case 0:
		opts.Interactive = true
		opts.PauseOnFinish = true
		return readInteractiveOptions(opts)
	case 1:
		opts.DOI = flags.MainArgs[0]
		return opts, nil
	default:
		return opts, errors.New("only one DOI may be specified")
	}
}

func readInteractiveOptions(opts Options) (Options, error) {
	reader := bufio.NewReader(os.Stdin)

	doi, err := promptRequired(reader, "DOI")
	if err != nil {
		return opts, err
	}
	opts.DOI = doi

	outputDir, err := promptDefault(reader, "Output directory", opts.OutputDir)
	if err != nil {
		return opts, err
	}
	opts.OutputDir = outputDir

	timeoutText, err := promptDefault(
		reader,
		"HTTP timeout in seconds",
		strconv.Itoa(int(opts.Timeout.Seconds())),
	)
	if err != nil {
		return opts, err
	}

	timeoutSeconds, err := strconv.Atoi(timeoutText)
	if err != nil || timeoutSeconds <= 0 {
		return opts, errors.New("HTTP timeout must be a positive integer")
	}
	opts.Timeout = time.Duration(timeoutSeconds) * time.Second

	fmt.Println()
	return opts, nil
}

func promptRequired(reader *bufio.Reader, label string) (string, error) {
	for {
		fmt.Printf("%s: ", label)

		value, err := readLine(reader)
		if err != nil {
			return "", err
		}
		if value != "" {
			return value, nil
		}

		fmt.Printf("%s is required.\n", label)
	}
}

func promptDefault(
	reader *bufio.Reader,
	label string,
	defaultValue string,
) (string, error) {
	fmt.Printf("%s [%s]: ", label, defaultValue)

	value, err := readLine(reader)
	if err != nil {
		return "", err
	}
	if value == "" {
		return defaultValue, nil
	}

	return value, nil
}

func readLine(reader *bufio.Reader) (string, error) {
	value, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}

	return strings.TrimSpace(value), nil
}
