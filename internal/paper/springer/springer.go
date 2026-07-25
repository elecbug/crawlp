package springer

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	appclient "github.com/elecbug/crawlp/internal/app/client"
	"github.com/elecbug/crawlp/internal/paper"
)

const springerBaseURL = "https://link.springer.com"

var springerDOIPrefixies = []string{
	"10.1007",
	"10.1023",
	"10.1057",
	"10.1065",
	"10.1114",
	"10.1134",
	"10.1140",
	"10.1186",
	"10.1208",
	"10.1245",
	"10.1251",
	"10.1361",
	"10.1365",
	"10.1379",
	"10.1381",
	"10.1385",
	"10.1617",
	"10.17269",
	"10.2165",
	"10.2991",
	"10.3758",
	"10.4076",
	"10.4098",
	"10.4333",
	"10.5052",
	"10.5819",
	"10.7603",
}

var (
	ErrUnsupportedDocumentType = errors.New(
		"the Springer DOI does not identify a supported article or chapter",
	)

	ErrBookLevelDOI = errors.New(
		"the DOI identifies an entire Springer book or proceedings volume, not an individual paper",
	)
)

func resolveSpringerDocument(
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
			"failed to resolve Springer DOI: %w",
			err,
		)
	}

	resolverBody, err := appclient.ReadAndClose(
		resolverResp,
		32<<20,
	)
	if err != nil {
		return paper.DocumentInfo{}, fmt.Errorf(
			"failed to read Springer DOI response: %w",
			err,
		)
	}

	finalURL := resolverResp.Request.URL.String()
	finalHost := normalizeHost(
		resolverResp.Request.URL.Hostname(),
	)

	if !isSpringerHost(finalHost) {
		return paper.DocumentInfo{}, fmt.Errorf(
			"DOI did not resolve to Springer Nature Link: %s",
			finalURL,
		)
	}

	documentType, err := classifyDocumentURL(
		resolverResp.Request.URL,
	)
	if err != nil {
		return paper.DocumentInfo{}, err
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
		appclient.ExtractMetaContent(
			landingHTML,
			"DC.title",
		),
		fallbackTitle(doi),
	)

	metadataPDF := firstNonEmpty(
		appclient.ExtractMetaContent(
			landingHTML,
			"citation_pdf_url",
		),
		buildDirectPDFURL(doi),
	)

	if metadataPDF != "" {
		metadataPDF, err = appclient.AbsoluteURL(
			finalURL,
			metadataPDF,
		)
		if err != nil {
			metadataPDF = buildDirectPDFURL(doi)
		}
	}

	return paper.DocumentInfo{
		DOI:          doi,
		Identifier:   doi,
		LandingURL:   finalURL,
		Title:        title,
		MetadataPDF:  metadataPDF,
		DocumentType: documentType,
	}, nil
}

func downloadSpringerPDF(
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
		buildDirectPDFURL(info.DOI),
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
			"Trying Springer PDF endpoint %d of %d...\n",
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
			"no usable Springer PDF endpoint was found",
		)
	}

	return "", fmt.Errorf(
		"all Springer PDF download endpoints failed: %w",
		lastErr,
	)
}

func classifyDocumentURL(
	landingURL *url.URL,
) (string, error) {
	path := strings.ToLower(
		strings.TrimSpace(landingURL.Path),
	)

	switch {
	case strings.HasPrefix(path, "/article/"):
		return "article", nil

	case strings.HasPrefix(path, "/chapter/"):
		return "chapter", nil

	case strings.HasPrefix(path, "/book/"):
		return "", ErrBookLevelDOI

	case strings.HasPrefix(path, "/referenceworkentry/"):
		return "reference-work-entry", nil

	default:
		return "", fmt.Errorf(
			"%w: %s",
			ErrUnsupportedDocumentType,
			landingURL.String(),
		)
	}
}

func buildDirectPDFURL(doi string) string {
	return fmt.Sprintf(
		"%s/content/pdf/%s.pdf",
		springerBaseURL,
		paper.EscapeDOIPath(doi),
	)
}

func isSpringerHost(host string) bool {
	return host == "link.springer.com"
}

func normalizeHost(host string) string {
	return strings.ToLower(
		strings.TrimSuffix(
			strings.TrimSpace(host),
			".",
		),
	)
}

func fallbackTitle(doi string) string {
	for _, prefix := range springerDOIPrefixies {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(doi)), prefix) {
			suffix := strings.TrimPrefix(
				strings.ToLower(
					strings.TrimSpace(doi),
				),
				prefix,
			)

			if suffix == "" {
				suffix = doi
			}

			return "Springer-" + appclient.SafeFilename(suffix)
		}
	}

	return "Springer-" + appclient.SafeFilename(strings.ToLower(strings.TrimSpace(doi)))
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
