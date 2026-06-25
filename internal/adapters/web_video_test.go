package adapters

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWebVideoResolverExtractsDouyinMetadata(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`
<!doctype html>
<html>
<head>
  <title>Fallback title - Douyin</title>
  <meta name="description" content="Why saving harder makes you poorer - Money Brothers posted on Douyin, already got 85k likes.">
  <meta name="lark:url:video_title" content="Why saving harder makes you poorer - Douyin">
  <meta name="lark:url:video_cover_image_url" content="https://example.com/cover.jpg">
</head>
<body></body>
</html>`))
	}))
	defer server.Close()

	resolver := WebVideoResolver{Client: server.Client()}
	video, err := resolver.Resolve(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	if video.Title != "Why saving harder makes you poorer" {
		t.Fatalf("title = %q", video.Title)
	}
	if video.Author != "Money Brothers" {
		t.Fatalf("author = %q", video.Author)
	}
	if video.SourceURL != server.URL {
		t.Fatalf("source url = %q", video.SourceURL)
	}
}

func TestWebVideoResolverFallsBackToHTMLTitle(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><head><title>Only HTML Title - Douyin</title></head></html>`))
	}))
	defer server.Close()

	resolver := WebVideoResolver{Client: server.Client()}
	video, err := resolver.Resolve(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	if video.Title != "Only HTML Title" {
		t.Fatalf("title = %q", video.Title)
	}
}

func TestWebVideoResolverStoresFinalURLAfterRedirect(t *testing.T) {
	t.Parallel()

	detail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><head><title>Resolved Video - Douyin</title></head></html>`))
	}))
	defer detail.Close()

	short := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, detail.URL, http.StatusFound)
	}))
	defer short.Close()

	resolver := WebVideoResolver{Client: short.Client()}
	video, err := resolver.Resolve(context.Background(), short.URL)
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	if video.SourceURL != short.URL {
		t.Fatalf("source url = %q", video.SourceURL)
	}
	if video.ResolvedURL != detail.URL {
		t.Fatalf("resolved url = %q, want %q", video.ResolvedURL, detail.URL)
	}
}

func TestWebVideoResolverExtractsDouyinScriptFallback(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body><script>{"desc":"Script video title #tag","author":{"nickname":"Script author"}}</script></body></html>`))
	}))
	defer server.Close()

	resolver := WebVideoResolver{Client: server.Client()}
	video, err := resolver.Resolve(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	if video.Title != "Script video title" {
		t.Fatalf("title = %q", video.Title)
	}
	if video.Author != "Script author" {
		t.Fatalf("author = %q", video.Author)
	}
}

func TestWebVideoResolverUsesDouyinLightFallback(t *testing.T) {
	t.Parallel()

	var detailHits int
	var lightHits int
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	mux.HandleFunc("/video/123", func(w http.ResponseWriter, r *http.Request) {
		detailHits++
		_, _ = w.Write([]byte(`<html><head><title>Shell Page</title></head><body>loading</body></html>`))
	})
	mux.HandleFunc("/light/123", func(w http.ResponseWriter, r *http.Request) {
		lightHits++
		_, _ = w.Write([]byte(`<html><head><meta name="lark:url:video_title" content="Light title - Douyin"></head></html>`))
	})

	resolver := WebVideoResolver{Client: server.Client(), DouyinBaseURL: server.URL}
	video, err := resolver.Resolve(context.Background(), server.URL+"/video/123")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	if detailHits != 1 || lightHits != 1 {
		t.Fatalf("detail hits = %d, light hits = %d", detailHits, lightHits)
	}
	if video.Title != "Light title" {
		t.Fatalf("title = %q", video.Title)
	}
}
