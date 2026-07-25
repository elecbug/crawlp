package client

import (
	"errors"
	"fmt"
	"html"
	"io"
	"mime"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) " +
	"AppleWebKit/537.36 (KHTML, like Gecko) " +
	"Chrome/150.0.0.0 Safari/537.36"

var (
	invalidFilenameChars = regexp.MustCompile(`[\\/:*?"<>|]`)
	whitespace           = regexp.MustCompile(`\s+`)
)

func NewHTTPClient(timeout time.Duration) (*http.Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	return &http.Client{
		Jar:     jar,
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.New("too many HTTP redirects")
			}

			setBrowserHeaders(req)
			return nil
		},
	}, nil
}

func DoGET(
	client *http.Client,
	targetURL string,
	accept string,
	referer string,
) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, err
	}

	setBrowserHeaders(req)
	req.Header.Set("Accept", accept)

	if referer != "" {
		req.Header.Set("Referer", referer)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("HTTP request failed: %+v", resp)
	}

	return resp, nil
}

func ReadAndClose(
	resp *http.Response,
	limit int64,
) ([]byte, error) {
	defer resp.Body.Close()

	reader := io.LimitReader(resp.Body, limit+1)
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	if int64(len(data)) > limit {
		return nil, fmt.Errorf(
			"HTTP response exceeded the %d-byte limit",
			limit,
		)
	}

	return data, nil
}

func ExtractMetaContent(htmlText string, name string) string {
	quotedName := regexp.QuoteMeta(name)

	patterns := []*regexp.Regexp{
		regexp.MustCompile(
			`(?is)<meta[^>]+name\s*=\s*["']` +
				quotedName +
				`["'][^>]+content\s*=\s*["']([^"']+)["']`,
		),
		regexp.MustCompile(
			`(?is)<meta[^>]+content\s*=\s*["']([^"']+)["'][^>]+name\s*=\s*["']` +
				quotedName +
				`["']`,
		),
	}

	for _, pattern := range patterns {
		match := pattern.FindStringSubmatch(htmlText)
		if len(match) >= 2 {
			return strings.TrimSpace(
				html.UnescapeString(match[1]),
			)
		}
	}

	return ""
}

func AbsoluteURL(base string, reference string) (string, error) {
	baseURL, err := url.Parse(base)
	if err != nil {
		return "", err
	}

	referenceURL, err := url.Parse(reference)
	if err != nil {
		return "", err
	}

	return baseURL.ResolveReference(referenceURL).String(), nil
}

func ContentDispositionFilename(
	resp *http.Response,
) string {
	header := resp.Header.Get("Content-Disposition")
	if header == "" {
		return ""
	}

	_, params, err := mime.ParseMediaType(header)
	if err != nil {
		return ""
	}

	filename := strings.TrimSpace(params["filename"])
	if filename == "" {
		return ""
	}

	return SafeFilename(filename)
}

func SafeFilename(value string) string {
	value = html.UnescapeString(value)
	value = invalidFilenameChars.ReplaceAllString(value, "_")
	value = whitespace.ReplaceAllString(value, " ")
	value = strings.TrimSpace(strings.TrimRight(value, ". "))

	if value == "" {
		return "paper" + time.Now().Format("20060102-150405")
	}

	runes := []rune(value)
	if len(runes) > 180 {
		value = string(runes[:180])
	}

	return value
}

func setBrowserHeaders(req *http.Request) {
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set(
		"Accept-Language",
		"en-US,en;q=0.9",
	)
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")
}
