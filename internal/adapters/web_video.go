package adapters

import (
	"context"
	"errors"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"video-summary-mvp/internal/domain"
)

type WebVideoResolver struct {
	Client        *http.Client
	DouyinBaseURL string
}

func (r WebVideoResolver) Resolve(ctx context.Context, rawURL string) (domain.Video, error) {
	client := r.Client
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return domain.Video{}, err
	}
	setBrowserHeaders(req)

	res, err := client.Do(req)
	if err != nil {
		return domain.Video{}, err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return domain.Video{}, errors.New("video page returned non-success status")
	}

	body, err := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if err != nil {
		return domain.Video{}, err
	}
	page := string(body)
	resolvedURL := res.Request.URL.String()
	meta := extractMeta(page)
	if shouldUseDouyinLightFallback(resolvedURL, meta, page) {
		if lightPage, lightURL, err := r.fetchDouyinLight(ctx, client, resolvedURL); err == nil && lightPage != "" {
			page = lightPage
			resolvedURL = lightURL
			meta = extractMeta(page)
		}
	}

	title := cleanDouyinTitle(firstNonEmptyString(
		meta["lark:url:video_title"],
		meta["og:title"],
		extractHTMLTitle(page),
		extractJSONStringField(page, "desc"),
	))
	if title == "" {
		title = "Untitled short video"
	}

	description := firstNonEmptyString(meta["description"], meta["og:description"])
	author := firstNonEmptyString(extractDouyinAuthor(description), extractJSONStringField(page, "nickname"))
	if author == "" {
		author = "unknown"
	}

	return domain.Video{
		SourceURL:  rawURL,
		ResolvedURL: resolvedURL,
		Title:      title,
		Author:     author,
	}, nil
}

func (r WebVideoResolver) fetchDouyinLight(ctx context.Context, client *http.Client, resolvedURL string) (string, string, error) {
	videoID := extractDouyinVideoID(resolvedURL)
	if videoID == "" {
		return "", "", errors.New("douyin video id not found")
	}
	baseURL := strings.TrimRight(firstNonEmptyString(r.DouyinBaseURL, "https://www.douyin.com"), "/")
	lightURL := baseURL + "/light/" + videoID
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, lightURL, nil)
	if err != nil {
		return "", "", err
	}
	setBrowserHeaders(req)
	res, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", "", errors.New("douyin light page returned non-success status")
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if err != nil {
		return "", "", err
	}
	return string(body), res.Request.URL.String(), nil
}

func extractMeta(page string) map[string]string {
	meta := map[string]string{}
	metaTagPattern := regexp.MustCompile(`(?is)<meta\s+[^>]*>`)

	for _, tag := range metaTagPattern.FindAllString(page, -1) {
		key := firstNonEmptyString(extractAttr(tag, "name"), extractAttr(tag, "property"))
		content := extractAttr(tag, "content")
		if key != "" && content != "" {
			meta[strings.ToLower(key)] = strings.TrimSpace(content)
		}
	}

	return meta
}

func setBrowserHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Cache-Control", "no-cache")
}

func shouldUseDouyinLightFallback(resolvedURL string, meta map[string]string, page string) bool {
	if !strings.Contains(resolvedURL, "douyin.com/video/") && !strings.Contains(resolvedURL, "/video/") {
		return false
	}
	if firstNonEmptyString(meta["lark:url:video_title"], meta["description"], extractJSONStringField(page, "desc")) != "" {
		return false
	}
	return extractDouyinVideoID(resolvedURL) != ""
}

func extractDouyinVideoID(value string) string {
	pattern := regexp.MustCompile(`/video/([0-9]+)`)
	match := pattern.FindStringSubmatch(value)
	if len(match) < 2 {
		return ""
	}
	return match[1]
}

func extractAttr(tag string, name string) string {
	doubleQuoted := regexp.MustCompile(`(?is)` + regexp.QuoteMeta(name) + `\s*=\s*"([^"]*)"`)
	if match := doubleQuoted.FindStringSubmatch(tag); len(match) >= 2 {
		return html.UnescapeString(match[1])
	}
	singleQuoted := regexp.MustCompile(`(?is)` + regexp.QuoteMeta(name) + `\s*=\s*'([^']*)'`)
	if match := singleQuoted.FindStringSubmatch(tag); len(match) >= 2 {
		return html.UnescapeString(match[1])
	}
	return ""
}

func extractHTMLTitle(page string) string {
	titlePattern := regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	match := titlePattern.FindStringSubmatch(page)
	if len(match) < 2 {
		return ""
	}
	return strings.TrimSpace(html.UnescapeString(match[1]))
}

func cleanDouyinTitle(value string) string {
	value = strings.TrimSpace(value)
	suffixes := []string{" - Douyin", " - douyin", " - 抖音"}
	for _, suffix := range suffixes {
		value = strings.TrimSuffix(value, suffix)
	}
	if idx := strings.Index(value, " #"); idx > 0 {
		value = value[:idx]
	}
	return strings.TrimSpace(value)
}

func extractDouyinAuthor(description string) string {
	description = strings.TrimSpace(description)
	if description == "" {
		return ""
	}

	patterns := []*regexp.Regexp{
		regexp.MustCompile(`-\s*(.*?)\s+posted on Douyin`),
		regexp.MustCompile(`-\s*(.*?)于\d{8}发布在抖音`),
	}
	for _, pattern := range patterns {
		match := pattern.FindStringSubmatch(description)
		if len(match) >= 2 && strings.TrimSpace(match[1]) != "" {
			return strings.TrimSpace(match[1])
		}
	}
	return ""
}

func extractJSONStringField(page string, field string) string {
	pattern := regexp.MustCompile(`(?is)"` + regexp.QuoteMeta(field) + `"\s*:\s*"((?:\\.|[^"\\])*)"`)
	match := pattern.FindStringSubmatch(page)
	if len(match) < 2 {
		return ""
	}
	value := strings.ReplaceAll(match[1], `\"`, `"`)
	value = strings.ReplaceAll(value, `\\`, `\`)
	value = strings.ReplaceAll(value, `\n`, "\n")
	return html.UnescapeString(strings.TrimSpace(value))
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
