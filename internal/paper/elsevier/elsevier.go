package elsevier

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	appclient "github.com/elecbug/crawlp/internal/app/client"
	"github.com/elecbug/crawlp/internal/paper"
)

const (
	scienceDirectBaseURL = "https://www.sciencedirect.com"
	elsevierDOIPrefix    = "10.1016/"
)

var (
	ErrNotElsevier = errors.New(
		"DOI does not resolve to a supported Elsevier platform",
	)

	ErrPIINotFound = errors.New(
		"failed to extract the Elsevier publication item identifier",
	)

	piiURLPatterns = []*regexp.Regexp{
		regexp.MustCompile(
			`(?i)/retrieve/pii/([a-z0-9]+)`,
		),
		regexp.MustCompile(
			`(?i)/science/article/pii/([a-z0-9]+)`,
		),
	}

	piiBodyPatterns = []*regexp.Regexp{
		regexp.MustCompile(
			`(?i)"pii"\s*:\s*"([a-z0-9]+)"`,
		),
		regexp.MustCompile(
			`(?i)"publicationItemIdentifier"\s*:\s*"([a-z0-9]+)"`,
		),
		regexp.MustCompile(
			`(?i)data-pii\s*=\s*["']([a-z0-9]+)["']`,
		),
	}
)

func resolveElsevierDocument(
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
			"failed to resolve Elsevier DOI: %w",
			err,
		)
	}

	resolverBody, err := appclient.ReadAndClose(
		resolverResp,
		32<<20,
	)
	if err != nil {
		return paper.DocumentInfo{}, fmt.Errorf(
			"failed to read Elsevier DOI response: %w",
			err,
		)
	}

	finalURL := resolverResp.Request.URL.String()
	finalHost := normalizeHost(
		resolverResp.Request.URL.Hostname(),
	)

	knownPrefix := hasElsevierDOIPrefix(doi)

	if !knownPrefix && !isElsevierHost(finalHost) {
		return paper.DocumentInfo{}, fmt.Errorf(
			"%w: %s",
			ErrNotElsevier,
			finalURL,
		)
	}

	resolverHTML := string(resolverBody)

	pii := extractPII(
		finalURL,
		resolverHTML,
	)

	if pii == "" {
		return paper.DocumentInfo{}, fmt.Errorf(
			"%w for DOI %s",
			ErrPIINotFound,
			doi,
		)
	}

	landingURL := buildLandingURL(pii)
	landingHTML := resolverHTML

	// The linking hub often provides enough metadata. A direct
	// ScienceDirect request is attempted only to obtain richer metadata.
	landingResp, landingErr := appclient.DoGET(
		cli,
		landingURL,
		"text/html,application/xhtml+xml",
		finalURL,
	)

	if landingErr == nil {
		landingBody, readErr := appclient.ReadAndClose(
			landingResp,
			32<<20,
		)

		if readErr == nil {
			landingHTML = string(landingBody)
			landingURL = landingResp.Request.URL.String()
		}
	}

	title := firstNonEmpty(
		appclient.ExtractMetaContent(
			landingHTML,
			"citation_title",
		),
		appclient.ExtractMetaContent(
			resolverHTML,
			"citation_title",
		),
		appclient.ExtractMetaContent(
			landingHTML,
			"dc.Title",
		),
		appclient.ExtractMetaContent(
			landingHTML,
			"DC.title",
		),
		appclient.ExtractMetaContent(
			resolverHTML,
			"dc.Title",
		),
		fallbackTitle(pii),
	)

	metadataPDF := firstNonEmpty(
		appclient.ExtractMetaContent(
			landingHTML,
			"citation_pdf_url",
		),
		appclient.ExtractMetaContent(
			resolverHTML,
			"citation_pdf_url",
		),
	)

	if metadataPDF != "" {
		absolutePDF, absoluteErr := appclient.AbsoluteURL(
			landingURL,
			metadataPDF,
		)

		if absoluteErr == nil {
			metadataPDF = absolutePDF
		} else {
			metadataPDF = ""
		}
	}

	return paper.DocumentInfo{
		DOI:         doi,
		Identifier:  pii,
		LandingURL:  landingURL,
		Title:       title,
		MetadataPDF: metadataPDF,
	}, nil
}

func downloadElsevierPDF(
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
			"Trying Elsevier PDF endpoint %d of %d...\n",
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
			"no usable Elsevier PDF endpoint was found",
		)
	}

	return "", fmt.Errorf(
		"all Elsevier PDF download endpoints failed: %w",
		lastErr,
	)
}

func buildPDFCandidates(
	info paper.DocumentInfo,
) []string {
	candidates := make([]string, 0, 3)

	if info.MetadataPDF != "" {
		candidates = append(
			candidates,
			info.MetadataPDF,
		)
	}

	escapedPII := url.PathEscape(info.Identifier)

	candidates = append(
		candidates,
		fmt.Sprintf(
			"%s/science/article/pii/%s/pdfft?isDTMRedir=true&download=true",
			scienceDirectBaseURL,
			escapedPII,
		),
		fmt.Sprintf(
			"%s/science/article/pii/%s/pdfft?isDTMRedir=true",
			scienceDirectBaseURL,
			escapedPII,
		),
	)

	return uniqueStrings(candidates)
}

func buildLandingURL(pii string) string {
	return fmt.Sprintf(
		"%s/science/article/pii/%s",
		scienceDirectBaseURL,
		url.PathEscape(pii),
	)
}

func extractPII(
	finalURL string,
	body string,
) string {
	for _, pattern := range piiURLPatterns {
		match := pattern.FindStringSubmatch(finalURL)

		if len(match) >= 2 {
			return normalizePII(match[1])
		}
	}

	metaNames := []string{
		"citation_pii",
		"pii",
		"dc.identifier",
	}

	for _, name := range metaNames {
		value := appclient.ExtractMetaContent(
			body,
			name,
		)

		if pii := extractPIIValue(value); pii != "" {
			return pii
		}
	}

	for _, pattern := range piiBodyPatterns {
		match := pattern.FindStringSubmatch(body)

		if len(match) >= 2 {
			return normalizePII(match[1])
		}
	}

	for _, pattern := range piiURLPatterns {
		match := pattern.FindStringSubmatch(body)

		if len(match) >= 2 {
			return normalizePII(match[1])
		}
	}

	return ""
}

func extractPIIValue(value string) string {
	value = strings.TrimSpace(value)

	if value == "" {
		return ""
	}

	for _, pattern := range piiURLPatterns {
		match := pattern.FindStringSubmatch(value)

		if len(match) >= 2 {
			return normalizePII(match[1])
		}
	}

	return normalizePII(value)
}

func normalizePII(value string) string {
	value = strings.TrimSpace(value)

	if value == "" {
		return ""
	}

	for _, character := range value {
		isDigit := character >= '0' && character <= '9'
		isUpper := character >= 'A' && character <= 'Z'
		isLower := character >= 'a' && character <= 'z'

		if !isDigit && !isUpper && !isLower {
			return ""
		}
	}

	if len(value) < 8 {
		return ""
	}

	return strings.ToUpper(value)
}

func hasElsevierDOIPrefix(doi string) bool {
	normalized := strings.ToLower(
		strings.TrimSpace(doi),
	)

	return strings.HasPrefix(
		normalized,
		elsevierDOIPrefix,
	)
}

func isElsevierHost(host string) bool {
	host = normalizeHost(host)

	switch host {
	case
		"linkinghub.elsevier.com",
		"sciencedirect.com",
		"www.sciencedirect.com",
		"elsevier.com",
		"www.elsevier.com",
		"cell.com",
		"www.cell.com",
		"thelancet.com",
		"www.thelancet.com":
		return true

	default:
		return false
	}
}

func normalizeHost(host string) string {
	return strings.ToLower(
		strings.TrimSuffix(
			strings.TrimSpace(host),
			".",
		),
	)
}

func fallbackTitle(pii string) string {
	if pii == "" {
		return "Elsevier-paper"
	}

	return "Elsevier-" + appclient.SafeFilename(pii)
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
