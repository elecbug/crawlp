package acm

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/elecbug/crawlp/internal/app/client"
	"github.com/elecbug/crawlp/internal/paper"
)

const (
	acmBaseURL      = "https://dl.acm.org"
	crossrefBaseURL = "https://api.crossref.org/v1"
)

type crossrefResponse struct {
	Status  string       `json:"status"`
	Message crossrefWork `json:"message"`
}

type crossrefWork struct {
	DOI       string         `json:"DOI"`
	Title     []string       `json:"title"`
	URL       string         `json:"URL"`
	Publisher string         `json:"publisher"`
	Link      []crossrefLink `json:"link"`
}

type crossrefLink struct {
	URL                 string `json:"URL"`
	ContentType         string `json:"content-type"`
	ContentVersion      string `json:"content-version"`
	IntendedApplication string `json:"intended-application"`
}

func resolveACMDocument(
	cli *http.Client,
	doi string,
) (paper.DocumentInfo, error) {
	crossrefURL := fmt.Sprintf(
		"%s/works/%s",
		crossrefBaseURL,
		paper.EscapeDOIPath(doi),
	)

	resp, err := client.DoGET(
		cli,
		crossrefURL,
		"application/json",
		"",
	)
	if err != nil {
		return paper.DocumentInfo{}, fmt.Errorf(
			"failed to request Crossref metadata: %w",
			err,
		)
	}

	body, err := client.ReadAndClose(resp, 8<<20)
	if err != nil {
		return paper.DocumentInfo{}, fmt.Errorf(
			"failed to read Crossref metadata response: %w",
			err,
		)
	}

	var result crossrefResponse

	if err := json.Unmarshal(body, &result); err != nil {
		return paper.DocumentInfo{}, fmt.Errorf(
			"failed to decode Crossref metadata: %w",
			err,
		)
	}

	if result.Status != "ok" {
		return paper.DocumentInfo{}, fmt.Errorf(
			"Crossref returned an unsuccessful status: %s",
			result.Status,
		)
	}

	if !strings.EqualFold(result.Message.DOI, doi) {
		return paper.DocumentInfo{}, fmt.Errorf(
			"Crossref returned a different DOI: %s",
			result.Message.DOI,
		)
	}

	title := fallbackTitle(doi)

	if len(result.Message.Title) > 0 {
		candidate := strings.TrimSpace(result.Message.Title[0])
		if candidate != "" {
			title = candidate
		}
	}

	landingURL := strings.TrimSpace(result.Message.URL)

	if landingURL == "" {
		landingURL = fmt.Sprintf(
			"%s/doi/%s",
			acmBaseURL,
			paper.EscapeDOIPath(doi),
		)
	}

	metadataPDF := findPDFLink(result.Message.Link)

	return paper.DocumentInfo{
		DOI:         doi,
		Identifier:  doi,
		LandingURL:  landingURL,
		Title:       title,
		MetadataPDF: metadataPDF,
	}, nil
}

func downloadACMPDF(
	cli *http.Client,
	info paper.DocumentInfo,
	outputDir string,
) (string, error) {
	candidates := buildPDFCandidates(info)

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

func findPDFLink(links []crossrefLink) string {
	for _, link := range links {
		contentType := strings.ToLower(
			strings.TrimSpace(link.ContentType),
		)

		if contentType != "application/pdf" {
			continue
		}

		target := strings.TrimSpace(link.URL)
		if target != "" {
			return target
		}
	}

	return ""
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

func buildPDFCandidates(info paper.DocumentInfo) []string {
	candidates := make([]string, 0, 3)

	if info.MetadataPDF != "" {
		candidates = append(
			candidates,
			info.MetadataPDF,
		)
	}

	escapedDOI := paper.EscapeDOIPath(info.DOI)

	candidates = append(
		candidates,
		fmt.Sprintf(
			"%s/doi/pdf/%s",
			acmBaseURL,
			escapedDOI,
		),
		fmt.Sprintf(
			"%s/doi/pdf/%s?download=true",
			acmBaseURL,
			escapedDOI,
		),
	)

	return uniqueStrings(candidates)
}
