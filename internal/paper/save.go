package paper

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/elecbug/crawlp/internal/app/client"
)

func UniqueOutputPath(
	outputDir string,
	filename string,
	title string,
	identifier string,
) string {
	outputPath := filepath.Join(
		outputDir,
		filename,
	)

	if _, err := os.Stat(outputPath); errors.Is(
		err,
		os.ErrNotExist,
	) {
		return outputPath
	}

	safeIdentifier := client.SafeFilename(identifier)

	base := fmt.Sprintf(
		"%s-%s",
		client.SafeFilename(title),
		safeIdentifier,
	)

	candidate := filepath.Join(
		outputDir,
		base+".pdf",
	)

	if _, err := os.Stat(candidate); errors.Is(
		err,
		os.ErrNotExist,
	) {
		return candidate
	}

	for index := 2; ; index++ {
		candidate = filepath.Join(
			outputDir,
			fmt.Sprintf(
				"%s-%d.pdf",
				base,
				index,
			),
		)

		if _, err := os.Stat(candidate); errors.Is(
			err,
			os.ErrNotExist,
		) {
			return candidate
		}
	}
}

func SaveVerifiedPDF(
	resp *http.Response,
	outputPath string,
) error {
	defer resp.Body.Close()

	tempFile, err := os.CreateTemp(
		filepath.Dir(outputPath),
		"ieee-*.tmp",
	)
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}

	tempPath := tempFile.Name()
	keepFile := false

	defer func() {
		_ = tempFile.Close()
		if !keepFile {
			_ = os.Remove(tempPath)
		}
	}()

	if _, err := io.Copy(tempFile, resp.Body); err != nil {
		return fmt.Errorf("failed to save HTTP response: %w", err)
	}

	if err := tempFile.Sync(); err != nil {
		return fmt.Errorf("failed to flush temporary file: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}

	file, err := os.Open(tempPath)
	if err != nil {
		return fmt.Errorf("failed to reopen temporary file: %w", err)
	}

	signature := make([]byte, 5)
	_, readErr := io.ReadFull(file, signature)
	_ = file.Close()

	if readErr != nil || string(signature) != "%PDF-" {
		return fmt.Errorf(
			"the server returned a non-PDF response; content type: %s; "+
				"final URL: %s; verify campus network, VPN, proxy, or subscription access",
			resp.Header.Get("Content-Type"),
			resp.Request.URL.String(),
		)
	}

	if err := os.Rename(tempPath, outputPath); err != nil {
		return fmt.Errorf(
			"failed to move the completed PDF: %w",
			err,
		)
	}

	keepFile = true
	return nil
}
