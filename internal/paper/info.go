package paper

import (
	"errors"
	"regexp"
)

type DocumentInfo struct {
	DOI         string
	Identifier  string
	LandingURL  string
	Title       string
	MetadataPDF string
}

var articlePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)/(?:abstract/)?document/(\d+)`),
	regexp.MustCompile(`(?i)[?&]arnumber=(\d+)`),
	regexp.MustCompile(`(?i)"articleNumber"\s*:\s*"?(\d+)"?`),
	regexp.MustCompile(`(?i)"arnumber"\s*:\s*"?(\d+)"?`),
	regexp.MustCompile(`(?i)"article_number"\s*:\s*"?(\d+)"?`),
}

func ExtractArticleNumber(
	targetURL string,
	htmlText string,
) (string, error) {
	for _, target := range []string{targetURL, htmlText} {
		for _, pattern := range articlePatterns {
			match := pattern.FindStringSubmatch(target)
			if len(match) >= 2 {
				return match[1], nil
			}
		}
	}

	return "", errors.New(
		"failed to find an IEEE article number; confirm that the DOI belongs to IEEE Xplore",
	)
}
