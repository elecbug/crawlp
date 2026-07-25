package app

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/elecbug/crawlp/internal/provider"
)

var ErrProviderSelectionCancelled = errors.New(
	"provider selection was cancelled",
)

func selectDownloaderInteractively(
	router *provider.Router,
	detected provider.Downloader,
	detectionErr error,
	autoProvier bool,
) (provider.Downloader, error) {
	reader := bufio.NewReader(os.Stdin)
	if detected != nil {
		fmt.Printf(
			"Automatically detected provider: %s (%s)\n",
			detected.Name(),
			detected.ID(),
		)

		var confirmed bool
		var err error

		if autoProvier {
			confirmed = true
		} else {
			confirmed, err = promptYesNo(
				reader,
				"Use this provider?",
				true,
			)
		}
		if err != nil {
			return nil, err
		}

		if confirmed {
			return detected, nil
		}

		fmt.Println()
	} else {
		fmt.Println("Automatic provider detection failed.")

		if detectionErr != nil {
			fmt.Printf("Reason: %v\n", detectionErr)
		}

		fmt.Println()
	}

	return promptDownloader(reader, router)
}

func promptDownloader(
	reader *bufio.Reader,
	router *provider.Router,
) (provider.Downloader, error) {
	downloaders := router.Downloaders()

	if len(downloaders) == 0 {
		return nil, errors.New(
			"no download providers are registered",
		)
	}

	fmt.Println("Available providers:")

	for index, downloader := range downloaders {
		fmt.Printf(
			"  %d. %s (%s)\n",
			index+1,
			downloader.Name(),
			downloader.ID(),
		)
	}

	fmt.Println()
	fmt.Println(
		"Enter a provider number or provider ID. Enter q to cancel.",
	)

	for {
		fmt.Print("Provider: ")

		input, err := readPromptLine(reader)
		if err != nil {
			return nil, err
		}

		input = strings.TrimSpace(input)

		if input == "" {
			fmt.Println("A provider selection is required.")
			continue
		}

		if strings.EqualFold(input, "q") ||
			strings.EqualFold(input, "quit") ||
			strings.EqualFold(input, "cancel") {
			return nil, ErrProviderSelectionCancelled
		}

		if index, err := strconv.Atoi(input); err == nil {
			if index >= 1 && index <= len(downloaders) {
				selected := downloaders[index-1]

				fmt.Printf(
					"Manually selected provider: %s (%s)\n",
					selected.Name(),
					selected.ID(),
				)

				return selected, nil
			}

			fmt.Printf(
				"Provider number must be between 1 and %d.\n",
				len(downloaders),
			)

			continue
		}

		if selected, found := router.FindByID(input); found {
			fmt.Printf(
				"Manually selected provider: %s (%s)\n",
				selected.Name(),
				selected.ID(),
			)

			return selected, nil
		}

		fmt.Printf("Unknown provider: %s\n", input)
	}
}

func promptYesNo(
	reader *bufio.Reader,
	message string,
	defaultValue bool,
) (bool, error) {
	defaultHint := "y/N"

	if defaultValue {
		defaultHint = "Y/n"
	}

	for {
		fmt.Printf("%s [%s]: ", message, defaultHint)

		input, err := readPromptLine(reader)
		if err != nil {
			return false, err
		}

		switch strings.ToLower(strings.TrimSpace(input)) {
		case "":
			return defaultValue, nil

		case "y", "yes":
			return true, nil

		case "n", "no":
			return false, nil

		default:
			fmt.Println("Please enter yes or no.")
		}
	}
}

func readPromptLine(
	reader *bufio.Reader,
) (string, error) {
	value, err := reader.ReadString('\n')

	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf(
			"failed to read console input: %w",
			err,
		)
	}

	if errors.Is(err, io.EOF) && value == "" {
		return "", io.EOF
	}

	return strings.TrimSpace(value), nil
}
