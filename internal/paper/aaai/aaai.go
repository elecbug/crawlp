package aaai

import (
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	appclient "github.com/elecbug/crawlp/internal/app/client"
	"github.com/elecbug/crawlp/internal/paper"
)

const (
	aaaiBaseURL   = "https://ojs.aaai.org"
	aaaiDOIPrefix = "10.1609/"
)

var (
	articleIDPattern = regexp.MustCompile(
		`(?i)/article/view/([0-9]+)(?:[/?#]|$)`,
	)

	hrefPattern = regexp.MustCompile(
		`(?is)<a[^>]+href\s*=\s*["']([^"']+)["']`,
	)

	ojsPDFViewPattern = regexp.MustCompile(
		`(?i)/article/view/[0-9]+/[0-9]+(?:[/?#]|$)`,
	)
)

func resolveAAAIDocument(
	cli *http.Client,
	doi string,
) (paper.DocumentInfo, error) {
	resolverURL := paper.DoiResolver + paper.EscapeDOIPath(doi)

	resolverResp, err := appclient.DoGET(
		cli,
		resolverURL,
		"text/html,application/xhtml+xml",
		"",
	)
	if err != nil {
		return paper.DocumentInfo{}, fmt.Errorf(
			"failed to resolve AAAI DOI: %w",
			err,
		)
	}

	resolverBody, err := appclient.ReadAndClose(
		resolverResp,
		32<<20,
	)
	if err != nil {
		return paper.DocumentInfo{}, fmt.Errorf(
			"failed to read AAAI DOI response: %w",
			err,
		)
	}

	finalURL := resolverResp.Request.URL.String()
	finalHost := strings.ToLower(
		strings.TrimSuffix(
			resolverResp.Request.URL.Hostname(),
			".",
		),
	)

	if finalHost != "ojs.aaai.org" {
		return paper.DocumentInfo{}, fmt.Errorf(
			"DOI did not resolve to the AAAI publication system: %s",
			finalURL,
		)
	}

	landingHTML := string(resolverBody)

	title := firstNonEmpty(
		appclient.ExtractMetaContent(
			landingHTML,
			"citation_title",
		),
		appclient.ExtractMetaContent(
			landingHTML,
			"dc.Title",
		),
		fallbackTitle(doi),
	)

	metadataPDF := firstNonEmpty(
		appclient.ExtractMetaContent(
			landingHTML,
			"citation_pdf_url",
		),
		extractOJSDownloadURL(
			landingHTML,
			finalURL,
		),
	)

	if metadataPDF != "" {
		metadataPDF, err = appclient.AbsoluteURL(
			finalURL,
			metadataPDF,
		)
		if err != nil {
			metadataPDF = ""
		}
	}

	identifier := extractArticleID(finalURL)

	if identifier == "" {
		identifier = doi
	}

	return paper.DocumentInfo{
		DOI:         doi,
		Identifier:  identifier,
		LandingURL:  finalURL,
		Title:       title,
		MetadataPDF: metadataPDF,
	}, nil
}

func downloadAAAIPDF(
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

	// Some DOI registrants support PDF content negotiation.
	// This is only a fallback and is verified using PDF magic bytes.
	candidates = append(
		candidates,
		paper.DoiResolver+paper.EscapeDOIPath(info.DOI),
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
			"Trying AAAI PDF endpoint %d of %d...\n",
			index+1,
			len(candidates),
		)

		resp, err := appclient.DoGET(
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

		filename := appclient.ContentDispositionFilename(resp)

		if filename == "" {
			filename = appclient.SafeFilename(
				info.Title,
			) + ".pdf"
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

		if err := paper.SaveVerifiedPDF(
			resp,
			outputPath,
		); err != nil {
			lastErr = err

			fmt.Fprintf(
				os.Stderr,
				"Attempt failed: %v\n",
				err,
			)

			continue
		}

		return outputPath, nil
	}

	if lastErr == nil {
		lastErr = errors.New(
			"no usable AAAI PDF endpoint was found",
		)
	}

	return "", fmt.Errorf(
		"all AAAI PDF download endpoints failed: %w",
		lastErr,
	)
}

func extractArticleID(landingURL string) string {
	match := articleIDPattern.FindStringSubmatch(
		landingURL,
	)

	if len(match) < 2 {
		return ""
	}

	return match[1]
}

func extractOJSDownloadURL(
	body string,
	baseURL string,
) string {
	body = html.UnescapeString(body)

	matches := hrefPattern.FindAllStringSubmatch(
		body,
		-1,
	)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		candidate := strings.TrimSpace(match[1])
		if candidate == "" {
			continue
		}

		absolute, err := resolveURL(
			baseURL,
			candidate,
		)
		if err != nil {
			continue
		}

		if !isAllowedAAAIURL(absolute) {
			continue
		}

		parsed, err := url.Parse(absolute)
		if err != nil {
			continue
		}

		path := parsed.Path
		lowerPath := strings.ToLower(path)

		if strings.Contains(
			lowerPath,
			"/article/download/",
		) {
			return absolute
		}

		if ojsPDFViewPattern.MatchString(path) {
			return absolute
		}
	}

	return ""
}

func resolveURL(
	baseURL string,
	reference string,
) (string, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	ref, err := url.Parse(reference)
	if err != nil {
		return "", err
	}

	return base.ResolveReference(ref).String(), nil
}

func isAllowedAAAIURL(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}

	if parsed.Scheme != "https" &&
		parsed.Scheme != "http" {
		return false
	}

	host := strings.ToLower(
		strings.TrimSuffix(
			parsed.Hostname(),
			".",
		),
	)

	return host == "ojs.aaai.org"
}

func fallbackTitle(doi string) string {
	suffix := strings.TrimPrefix(
		strings.ToLower(
			strings.TrimSpace(doi),
		),
		aaaiDOIPrefix,
	)

	if suffix == "" {
		suffix = doi
	}

	return "AAAI-" + appclient.SafeFilename(suffix)
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
		value = strings.TrimSpace(value)
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
