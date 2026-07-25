package acm

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/elecbug/crawlp/internal/app/client"
	"github.com/elecbug/crawlp/internal/paper"
)

const acmBaseURL = "https://dl.acm.org"

func resolveACMDocument(
	cli *http.Client,
	doi string,
) (paper.DocumentInfo, error) {
	canonicalURL := fmt.Sprintf(
		"%s/doi/%s",
		acmBaseURL,
		paper.EscapeDOIPath(doi),
	)

	landingResp, err := client.DoGET(
		cli,
		canonicalURL,
		"text/html,application/xhtml+xml",
		"",
	)
	if err != nil {
		return paper.DocumentInfo{}, fmt.Errorf(
			"failed to request ACM Digital Library landing page: %w",
			err,
		)
	}

	landingBody, err := client.ReadAndClose(landingResp, 32<<20)
	if err != nil {
		return paper.DocumentInfo{}, fmt.Errorf(
			"failed to read ACM Digital Library landing page: %w",
			err,
		)
	}

	finalURL := landingResp.Request.URL.String()
	finalHost := strings.ToLower(
		strings.TrimSuffix(
			landingResp.Request.URL.Hostname(),
			".",
		),
	)

	if finalHost != "dl.acm.org" {
		return paper.DocumentInfo{}, fmt.Errorf(
			"ACM DOI did not resolve to the ACM Digital Library: %s",
			finalURL,
		)
	}

	landingHTML := string(landingBody)

	title := firstNonEmpty(
		client.ExtractMetaContent(
			landingHTML,
			"citation_title",
		),
		client.ExtractMetaContent(
			landingHTML,
			"dc.Title",
		),
		fallbackTitle(doi),
	)

	metadataPDF := firstNonEmpty(
		client.ExtractMetaContent(
			landingHTML,
			"citation_pdf_url",
		),
	)

	if metadataPDF != "" {
		metadataPDF, err = client.AbsoluteURL(
			acmBaseURL,
			metadataPDF,
		)
		if err != nil {
			metadataPDF = ""
		}
	}

	return paper.DocumentInfo{
		DOI:         doi,
		Identifier:  doi,
		LandingURL:  finalURL,
		Title:       title,
		MetadataPDF: metadataPDF,
	}, nil
}

func downloadACMPDF(
	cli *http.Client,
	info paper.DocumentInfo,
	outputDir string,
) (string, error) {
	candidates := make([]string, 0, 2)

	if info.MetadataPDF != "" {
		candidates = append(
			candidates,
			info.MetadataPDF,
		)
	}

	candidates = append(
		candidates,
		fmt.Sprintf(
			"%s/doi/pdf/%s",
			acmBaseURL,
			paper.EscapeDOIPath(info.DOI),
		),
	)

	candidates = uniqueStrings(candidates)

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", fmt.Errorf(
			"failed to create output directory: %w",
			err,
		)
	}

	var lastErr error

	for index, pdfURL := range candidates {
		fmt.Printf(
			"Trying ACM PDF endpoint %d of %d...\n",
			index+1,
			len(candidates),
		)

		resp, err := client.DoGET(
			cli,
			pdfURL,
			"application/pdf,application/octet-stream;q=0.9,*/*;q=0.8",
			info.LandingURL,
		)
		if err != nil {
			lastErr = err
			fmt.Fprintf(
				os.Stderr,
				"Attempt failed: %v\n",
				err,
			)
			continue
		}

		filename := client.ContentDispositionFilename(resp)

		if filename == "" {
			filename = client.SafeFilename(info.Title) + ".pdf"
		} else if !strings.HasSuffix(
			strings.ToLower(filename),
			".pdf",
		) {
			filename += ".pdf"
		}

		outputPath := paper.UniqueOutputPath(
			outputDir,
			filename,
			info.Title,
			info.Identifier,
		)

		err = paper.SaveVerifiedPDF(
			resp,
			outputPath,
		)
		if err == nil {
			return outputPath, nil
		}

		lastErr = err

		fmt.Fprintf(
			os.Stderr,
			"Attempt failed: %v\n",
			err,
		)
	}

	if lastErr == nil {
		lastErr = errors.New("unknown download error")
	}

	return "", fmt.Errorf(
		"all ACM PDF download endpoints failed: %w",
		lastErr,
	)
}

func fallbackTitle(doi string) string {
	suffix := strings.TrimPrefix(
		strings.ToLower(strings.TrimSpace(doi)),
		"10.1145/",
	)

	if suffix == "" {
		suffix = doi
	}

	return "ACM-" + client.SafeFilename(suffix)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}

	return ""
}

func uniqueStrings(values []string) []string {
	seen := make(
		map[string]struct{},
		len(values),
	)

	result := make(
		[]string,
		0,
		len(values),
	)

	for _, value := range values {
		if value == "" {
			continue
		}

		if _, exists := seen[value]; exists {
			continue
		}

		seen[value] = struct{}{}
		result = append(result, value)
	}

	return result
}
