package ieee

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/elecbug/crawlp/internal/app/client"
	"github.com/elecbug/crawlp/internal/paper"
)

const ieeeBaseURL = "https://ieeexplore.ieee.org"

func resolveIEEEDocument(
	cli *http.Client,
	doi string,
) (paper.DocumentInfo, error) {
	resolverURL := paper.DoiResolver + paper.EscapeDOIPath(doi)

	resolverResp, err := client.DoGET(
		cli,
		resolverURL,
		"text/html,application/xhtml+xml",
		"",
	)
	if err != nil {
		return paper.DocumentInfo{}, fmt.Errorf("failed to resolve DOI: %w", err)
	}

	resolverBody, err := client.ReadAndClose(resolverResp, 32<<20)
	if err != nil {
		return paper.DocumentInfo{}, fmt.Errorf("failed to read DOI response: %w", err)
	}

	finalURL := resolverResp.Request.URL.String()
	finalHost := strings.ToLower(resolverResp.Request.URL.Hostname())

	if !strings.Contains(finalHost, "ieeexplore.ieee.org") {
		return paper.DocumentInfo{}, fmt.Errorf(
			"DOI did not resolve to IEEE Xplore: %s",
			finalURL,
		)
	}

	articleNo, err := paper.ExtractArticleNumber(
		finalURL,
		string(resolverBody),
	)
	if err != nil {
		return paper.DocumentInfo{}, err
	}

	canonicalURL := fmt.Sprintf(
		"%s/document/%s",
		ieeeBaseURL,
		articleNo,
	)

	landingResp, err := client.DoGET(
		cli,
		canonicalURL,
		"text/html,application/xhtml+xml",
		finalURL,
	)
	if err != nil {
		return paper.DocumentInfo{}, fmt.Errorf(
			"failed to request IEEE Xplore landing page: %w",
			err,
		)
	}

	landingBody, err := client.ReadAndClose(landingResp, 32<<20)
	if err != nil {
		return paper.DocumentInfo{}, fmt.Errorf(
			"failed to read IEEE Xplore landing page: %w",
			err,
		)
	}

	landingHTML := string(landingBody)
	resolverHTML := string(resolverBody)

	title := firstNonEmpty(
		client.ExtractMetaContent(landingHTML, "citation_title"),
		client.ExtractMetaContent(resolverHTML, "citation_title"),
		"IEEE-"+articleNo,
	)

	metadataPDF := firstNonEmpty(
		client.ExtractMetaContent(landingHTML, "citation_pdf_url"),
		client.ExtractMetaContent(resolverHTML, "citation_pdf_url"),
	)

	if metadataPDF != "" {
		metadataPDF, err = client.AbsoluteURL(ieeeBaseURL, metadataPDF)
		if err != nil {
			metadataPDF = ""
		}
	}

	return paper.DocumentInfo{
		LandingURL:  landingResp.Request.URL.String(),
		ArticleNo:   articleNo,
		Title:       title,
		MetadataPDF: metadataPDF,
	}, nil
}

func downloadIEEEPDF(
	cli *http.Client,
	info paper.DocumentInfo,
	outputDir string,
) (string, error) {
	candidates := make([]string, 0, 3)

	if info.MetadataPDF != "" {
		candidates = append(candidates, info.MetadataPDF)
	}

	candidates = append(
		candidates,
		fmt.Sprintf(
			"%s/stampPDF/getPDF.jsp?tp=&arnumber=%s&ref=",
			ieeeBaseURL,
			info.ArticleNo,
		),
		fmt.Sprintf(
			"%s/stampPDF/getPDF.jsp?arnumber=%s",
			ieeeBaseURL,
			info.ArticleNo,
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
			"Trying PDF endpoint %d of %d...\n",
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
			fmt.Fprintf(os.Stderr, "Attempt failed: %v\n", err)
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
			info.ArticleNo,
		)

		err = paper.SaveVerifiedPDF(resp, outputPath)
		if err == nil {
			return outputPath, nil
		}

		lastErr = err
		fmt.Fprintf(os.Stderr, "Attempt failed: %v\n", err)
	}

	if lastErr == nil {
		lastErr = errors.New("unknown download error")
	}

	return "", fmt.Errorf(
		"all PDF download endpoints failed: %w",
		lastErr,
	)
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
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))

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
