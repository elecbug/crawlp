package paper

import (
	"fmt"
	"net/url"
	"strings"
)

const DoiResolver = "https://doi.org/"

func NormalizeDOI(value string) (string, error) {
	value = strings.TrimSpace(value)
	lower := strings.ToLower(value)

	prefixes := []string{
		"https://doi.org/",
		"http://doi.org/",
		"https://dx.doi.org/",
		"http://dx.doi.org/",
		"doi:",
	}

	for _, prefix := range prefixes {
		if strings.HasPrefix(lower, prefix) {
			value = strings.TrimSpace(value[len(prefix):])
			break
		}
	}

	if !strings.HasPrefix(value, "10.") || !strings.Contains(value, "/") {
		return "", fmt.Errorf("invalid DOI format: %s", value)
	}

	return value, nil
}

func EscapeDOIPath(doi string) string {
	parts := strings.Split(doi, "/")

	for index, part := range parts {
		parts[index] = url.PathEscape(part)
	}

	return strings.Join(parts, "/")
}
