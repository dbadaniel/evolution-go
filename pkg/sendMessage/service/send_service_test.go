package send_service

import (
	"bytes"
	"image"
	"image/png"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func httpResponse(status int, body, contentType string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{"Content-Type": []string{contentType}},
	}
}

func TestResolveLinkTextAndMatchedURL(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		explicitURL string
		wantText    string
		wantMatched string
	}{
		{
			name:        "explicit URL already in text",
			text:        "Assista https://www.youtube.com/watch?v=abc agora",
			explicitURL: "https://www.youtube.com/watch?v=abc",
			wantText:    "Assista https://www.youtube.com/watch?v=abc agora",
			wantMatched: "https://www.youtube.com/watch?v=abc",
		},
		{
			name:        "explicit URL is appended exactly once",
			text:        "Assista agora",
			explicitURL: "https://youtu.be/abc",
			wantText:    "Assista agora\nhttps://youtu.be/abc",
			wantMatched: "https://youtu.be/abc",
		},
		{
			name:        "substring is not treated as the exact URL",
			text:        "Assista https://youtu.be/abcdef",
			explicitURL: "https://youtu.be/abc",
			wantText:    "Assista https://youtu.be/abcdef\nhttps://youtu.be/abc",
			wantMatched: "https://youtu.be/abc",
		},
		{
			name:        "uppercase scheme is accepted",
			text:        "Assista HTTP://youtube.com/watch?v=abc",
			explicitURL: "HTTP://youtube.com/watch?v=abc",
			wantText:    "Assista HTTP://youtube.com/watch?v=abc",
			wantMatched: "HTTP://youtube.com/watch?v=abc",
		},
		{
			name:        "uppercase scheme is discovered in text",
			text:        "Assista HTTPS://youtube.com/watch?v=abc",
			wantText:    "Assista HTTPS://youtube.com/watch?v=abc",
			wantMatched: "HTTPS://youtube.com/watch?v=abc",
		},
		{
			name:        "legitimate URL ending in punctuation is preserved",
			text:        "Veja https://example.com/wiki/Foo_(bar)",
			explicitURL: "https://example.com/wiki/Foo_(bar)",
			wantText:    "Veja https://example.com/wiki/Foo_(bar)",
			wantMatched: "https://example.com/wiki/Foo_(bar)",
		},
		{
			name:        "explicit URL legitimately ending in period is not duplicated",
			text:        "Veja https://example.com/path.",
			explicitURL: "https://example.com/path.",
			wantText:    "Veja https://example.com/path.",
			wantMatched: "https://example.com/path.",
		},
		{
			name:        "URL enclosed in parentheses is not duplicated",
			text:        "Veja (https://example.com)",
			explicitURL: "https://example.com",
			wantText:    "Veja (https://example.com)",
			wantMatched: "https://example.com",
		},
		{
			name:        "URL in markdown is not duplicated",
			text:        "[site](https://example.com)",
			explicitURL: "https://example.com",
			wantText:    "[site](https://example.com)",
			wantMatched: "https://example.com",
		},
		{
			name:        "URL in markdown followed by punctuation is not duplicated",
			text:        "[site](https://example.com).",
			explicitURL: "https://example.com",
			wantText:    "[site](https://example.com).",
			wantMatched: "https://example.com",
		},
		{
			name:        "URL after punctuation is not duplicated",
			text:        "link=https://example.com",
			explicitURL: "https://example.com",
			wantText:    "link=https://example.com",
			wantMatched: "https://example.com",
		},
		{
			name:        "longer query is not mistaken for explicit URL",
			text:        "Veja https://example.com/path?a=1&b=2",
			explicitURL: "https://example.com/path?a=1",
			wantText:    "Veja https://example.com/path?a=1&b=2\nhttps://example.com/path?a=1",
			wantMatched: "https://example.com/path?a=1",
		},
		{
			name:        "URL adjacent to emoji is not duplicated",
			text:        "Veja👉https://example.com✅",
			explicitURL: "https://example.com",
			wantText:    "Veja👉https://example.com✅",
			wantMatched: "https://example.com",
		},
		{
			name:        "whitespace-only text becomes only the URL",
			text:        " \n\t ",
			explicitURL: "https://youtu.be/abc",
			wantText:    "https://youtu.be/abc",
			wantMatched: "https://youtu.be/abc",
		},
		{
			name:        "invalid explicit URL falls back to URL in text",
			text:        "Leia https://example.com/noticia",
			explicitURL: "javascript:alert(1)",
			wantText:    "Leia https://example.com/noticia",
			wantMatched: "https://example.com/noticia",
		},
		{
			name:        "multiline explicit URL is ignored",
			text:        "Texto seguro",
			explicitURL: "https://example.com\r\nconteudo injetado",
			wantText:    "Texto seguro",
			wantMatched: "",
		},
		{
			name:        "explicit URL with internal whitespace is ignored",
			text:        "Texto seguro",
			explicitURL: "https://example.com/parte\tindevida",
			wantText:    "Texto seguro",
			wantMatched: "",
		},
		{
			name:        "explicit URL with internal control is ignored",
			text:        "Texto seguro",
			explicitURL: "https://example.com/parte\x00indevida",
			wantText:    "Texto seguro",
			wantMatched: "",
		},
		{
			name:        "explicit URL with credentials is ignored",
			text:        "Texto seguro",
			explicitURL: "https://usuario:senha@example.com/privado",
			wantText:    "Texto seguro",
			wantMatched: "",
		},
		{
			name:        "consumer line breaks are preserved",
			text:        "Linha um\nLinha dois\n",
			explicitURL: "https://example.com",
			wantText:    "Linha um\nLinha dois\n\nhttps://example.com",
			wantMatched: "https://example.com",
		},
		{
			name:        "URL is discovered in text",
			text:        "Leia https://example.com/noticia hoje",
			wantText:    "Leia https://example.com/noticia hoje",
			wantMatched: "https://example.com/noticia",
		},
		{
			name:        "URL before unicode whitespace is discovered",
			text:        "Leia https://example.com/noticia\u00a0agora",
			wantText:    "Leia https://example.com/noticia\u00a0agora",
			wantMatched: "https://example.com/noticia",
		},
		{
			name:        "URL before curly quote is discovered",
			text:        "Leia “https://example.com/noticia” agora",
			wantText:    "Leia “https://example.com/noticia” agora",
			wantMatched: "https://example.com/noticia",
		},
		{
			name:        "URL before guillemet is discovered",
			text:        "Leia «https://example.com/noticia» agora",
			wantText:    "Leia «https://example.com/noticia» agora",
			wantMatched: "https://example.com/noticia",
		},
		{
			name:        "URL before unicode ellipsis is discovered",
			text:        "Leia https://example.com/noticia… depois",
			wantText:    "Leia https://example.com/noticia… depois",
			wantMatched: "https://example.com/noticia",
		},
		{
			name:        "URL with apostrophe in path is preserved",
			text:        "Leia https://example.com/o'brien agora",
			wantText:    "Leia https://example.com/o'brien agora",
			wantMatched: "https://example.com/o'brien",
		},
		{
			name:        "URL before zero width separator is discovered",
			text:        "Leia https://example.com/noticia\u200bagora",
			wantText:    "Leia https://example.com/noticia\u200bagora",
			wantMatched: "https://example.com/noticia",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotText, gotMatched := resolveLinkTextAndMatchedURL(tt.text, tt.explicitURL)
			if gotText != tt.wantText || gotMatched != tt.wantMatched {
				t.Fatalf("resolve link = (%q, %q), want (%q, %q)", gotText, gotMatched, tt.wantText, tt.wantMatched)
			}
			if gotMatched != "" && !strings.Contains(gotText, gotMatched) {
				t.Fatalf("resolved text %q does not contain matched URL %q", gotText, gotMatched)
			}
			secondText, secondMatched := resolveLinkTextAndMatchedURL(gotText, tt.explicitURL)
			if secondText != gotText || secondMatched != gotMatched {
				t.Fatalf("resolution must be idempotent, got (%q, %q) after (%q, %q)", secondText, secondMatched, gotText, gotMatched)
			}
		})
	}
}

func TestIsYouTubeURLOnlyAcceptsTrustedHosts(t *testing.T) {
	tests := map[string]bool{
		"https://youtube.com/watch?v=abc":              true,
		"https://www.youtube.com/live/abc":             true,
		"https://music.youtube.com/watch?v=abc":        true,
		"https://youtu.be/abc":                         true,
		"HTTP://YOUTUBE.COM/watch?v=abc":               true,
		"https://youtube.com.evil.example/watch?v=abc": false,
		"https://notyoutube.com/watch?v=abc":           false,
		"https://youtu.be.evil.example/abc":            false,
		"ftp://youtube.com/watch?v=abc":                false,
		"javascript://youtube.com/watch?v=abc":         false,
		"not a URL":                                    false,
	}

	for rawURL, want := range tests {
		if got := isYouTubeURL(rawURL); got != want {
			t.Errorf("isYouTubeURL(%q) = %v, want %v", rawURL, got, want)
		}
	}
}

func TestFetchLinkMetadataUsesYouTubeOEmbed(t *testing.T) {
	targetURL := "https://www.youtube.com/live/live-id"
	requests := 0
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.URL.Host != "www.youtube.com" || req.URL.Path != "/oembed" {
			t.Fatalf("unexpected request URL: %s", req.URL)
		}
		if req.URL.Query().Get("url") != targetURL || req.URL.Query().Get("format") != "json" {
			t.Fatalf("unexpected oEmbed query: %s", req.URL.RawQuery)
		}
		return httpResponse(http.StatusOK, `{"title":"Live correta","author_name":"Canal correto","thumbnail_url":"https://i.ytimg.com/vi/live-id/hqdefault.jpg"}`, "application/json"), nil
	})}

	title, description, imgURL, err := fetchLinkMetadataWithClient(targetURL, client)
	if err != nil {
		t.Fatalf("fetch YouTube metadata: %v", err)
	}
	if title != "Live correta" || description != "Canal correto" || imgURL != "https://i.ytimg.com/vi/live-id/hqdefault.jpg" {
		t.Fatalf("unexpected metadata: title=%q description=%q image=%q", title, description, imgURL)
	}
	if requests != 1 {
		t.Fatalf("successful oEmbed must avoid HTML fallback, got %d requests", requests)
	}
}

func TestFetchLinkMetadataFallsBackToHTMLWhenYouTubeOEmbedIsIncomplete(t *testing.T) {
	targetURL := "https://www.youtube.com/live/live-id"
	requests := 0
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.URL.Path == "/oembed" {
			return httpResponse(http.StatusOK, `{"title":"Live sem thumbnail","author_name":"Canal"}`, "application/json"), nil
		}
		if req.URL.String() != targetURL {
			t.Fatalf("unexpected fallback URL: %s", req.URL)
		}
		return httpResponse(http.StatusOK, `<html><head><meta property="og:title" content="Live via HTML"><meta property="og:description" content="Descricao HTML"><meta property="og:image" content="/thumb.jpg"></head></html>`, "text/html"), nil
	})}

	title, description, imgURL, err := fetchLinkMetadataWithClient(targetURL, client)
	if err != nil {
		t.Fatalf("fetch fallback metadata: %v", err)
	}
	if title != "Live via HTML" || description != "Descricao HTML" || imgURL != "https://www.youtube.com/thumb.jpg" {
		t.Fatalf("unexpected fallback metadata: title=%q description=%q image=%q", title, description, imgURL)
	}
	if requests != 2 {
		t.Fatalf("incomplete oEmbed must use HTML fallback, got %d requests", requests)
	}
}

func TestFetchLinkMetadataFallsBackToHTMLWhenYouTubeOEmbedFails(t *testing.T) {
	targetURL := "https://youtu.be/live-id"
	requests := 0
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.URL.Path == "/oembed" {
			return httpResponse(http.StatusNotFound, `not found`, "text/plain"), nil
		}
		if req.URL.String() != targetURL {
			t.Fatalf("unexpected fallback URL: %s", req.URL)
		}
		return httpResponse(http.StatusOK, `<html><head><meta property="og:title" content="Fallback"><meta property="og:description" content="Pagina publica"><meta property="og:image" content="https://img.example/thumb.jpg"></head></html>`, "text/html"), nil
	})}

	title, description, imgURL, err := fetchLinkMetadataWithClient(targetURL, client)
	if err != nil {
		t.Fatalf("fetch fallback metadata after oEmbed failure: %v", err)
	}
	if title != "Fallback" || description != "Pagina publica" || imgURL != "https://img.example/thumb.jpg" {
		t.Fatalf("unexpected fallback metadata: title=%q description=%q image=%q", title, description, imgURL)
	}
	if requests != 2 {
		t.Fatalf("failed oEmbed must use HTML fallback, got %d requests", requests)
	}
}

func TestFetchLinkMetadataFallsBackWhenYouTubeOEmbedJSONIsMalformed(t *testing.T) {
	targetURL := "https://www.youtube.com/watch?v=live-id"
	requests := 0
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.URL.Path == "/oembed" {
			return httpResponse(http.StatusOK, `{not-json`, "application/json"), nil
		}
		return httpResponse(http.StatusOK, `<html><head><title>Fallback seguro</title></head></html>`, "text/html"), nil
	})}

	title, _, _, err := fetchLinkMetadataWithClient(targetURL, client)
	if err != nil {
		t.Fatalf("fetch fallback after malformed JSON: %v", err)
	}
	if title != "Fallback seguro" || requests != 2 {
		t.Fatalf("malformed JSON did not use fallback: title=%q requests=%d", title, requests)
	}
}

func TestFetchLinkMetadataFallsBackWhenYouTubeThumbnailURLIsInvalid(t *testing.T) {
	targetURL := "https://www.youtube.com/watch?v=live-id"
	requests := 0
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.URL.Path == "/oembed" {
			return httpResponse(http.StatusOK, `{"title":"Live","author_name":"Canal","thumbnail_url":"javascript:alert(1)"}`, "application/json"), nil
		}
		return httpResponse(http.StatusOK, `<html><head><title>Fallback thumbnail segura</title></head></html>`, "text/html"), nil
	})}

	title, _, _, err := fetchLinkMetadataWithClient(targetURL, client)
	if err != nil {
		t.Fatalf("fetch fallback after invalid thumbnail URL: %v", err)
	}
	if title != "Fallback thumbnail segura" || requests != 2 {
		t.Fatalf("invalid oEmbed thumbnail did not use fallback: title=%q requests=%d", title, requests)
	}
}

func TestFetchLinkMetadataReservesTimeForHTMLFallback(t *testing.T) {
	targetURL := "https://www.youtube.com/watch?v=live-id"
	var oEmbedRemaining, fallbackRemaining time.Duration
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		deadline, ok := req.Context().Deadline()
		if !ok {
			t.Fatal("metadata request must have a deadline")
		}
		if req.URL.Path == "/oembed" {
			oEmbedRemaining = time.Until(deadline)
			return httpResponse(http.StatusNotFound, "missing", "text/plain"), nil
		}
		fallbackRemaining = time.Until(deadline)
		return httpResponse(http.StatusOK, `<html><head><title>Fallback com tempo</title></head></html>`, "text/html"), nil
	})}

	if _, _, _, err := fetchLinkMetadataWithClient(targetURL, client); err != nil {
		t.Fatalf("fetch metadata with reserved fallback time: %v", err)
	}
	if oEmbedRemaining <= 0 || oEmbedRemaining > 3100*time.Millisecond {
		t.Fatalf("unexpected oEmbed deadline: %v", oEmbedRemaining)
	}
	if fallbackRemaining < 9*time.Second || fallbackRemaining <= oEmbedRemaining {
		t.Fatalf("fallback did not retain the total budget: oEmbed=%v fallback=%v", oEmbedRemaining, fallbackRemaining)
	}
}

func TestFetchLinkMetadataLimitsOnlyYouTubeOEmbedResponse(t *testing.T) {
	targetURL := "https://www.youtube.com/watch?v=live-id"
	requests := 0
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.URL.Path == "/oembed" {
			return httpResponse(http.StatusOK, strings.Repeat("x", maxLinkMetadataResponseBytes+1), "application/json"), nil
		}
		largePadding := strings.Repeat(" ", maxLinkMetadataResponseBytes+1)
		return httpResponse(http.StatusOK, `<html><head><title>HTML grande aceito</title></head><body>`+largePadding+`</body></html>`, "text/html"), nil
	})}

	title, _, _, err := fetchLinkMetadataWithClient(targetURL, client)
	if err != nil {
		t.Fatalf("fetch HTML fallback after oversized oEmbed: %v", err)
	}
	if title != "HTML grande aceito" || requests != 2 {
		t.Fatalf("oversized oEmbed did not use unrestricted HTML fallback: title=%q requests=%d", title, requests)
	}
}

func TestFetchLinkMetadataPreservesBothFailureCauses(t *testing.T) {
	targetURL := "https://youtu.be/live-id"
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == "/oembed" {
			return httpResponse(http.StatusNotFound, "missing", "text/plain"), nil
		}
		return httpResponse(http.StatusBadGateway, "unavailable", "text/plain"), nil
	})}

	_, _, _, err := fetchLinkMetadataWithClient(targetURL, client)
	if err == nil || !strings.Contains(err.Error(), "oEmbed") || !strings.Contains(err.Error(), "404") || !strings.Contains(err.Error(), "502") {
		t.Fatalf("combined error must preserve both causes, got %v", err)
	}
}

func TestFetchLinkMetadataCommonSiteDoesNotCallYouTubeOEmbed(t *testing.T) {
	targetURL := "https://example.com/article"
	requests := 0
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.URL.String() != targetURL {
			t.Fatalf("common site unexpectedly called another endpoint: %s", req.URL)
		}
		return httpResponse(http.StatusOK, `<html><head><title>Example</title><meta name="description" content="Article"><meta property="og:image" content="https://cdn.example.com/thumb.jpg"></head></html>`, "text/html"), nil
	})}

	title, description, imgURL, err := fetchLinkMetadataWithClient(targetURL, client)
	if err != nil {
		t.Fatalf("fetch common metadata: %v", err)
	}
	if title != "Example" || description != "Article" || imgURL != "https://cdn.example.com/thumb.jpg" {
		t.Fatalf("unexpected common metadata: title=%q description=%q image=%q", title, description, imgURL)
	}
	if requests != 1 {
		t.Fatalf("common site must make one HTML request, got %d", requests)
	}
}

func TestFetchHTMLLinkMetadataResolvesImageAgainstFinalRedirectURL(t *testing.T) {
	targetURL := "https://short.example/article"
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		resp := httpResponse(http.StatusOK, `<html><head><title>Redirected</title><meta property="og:image" content="../thumb.jpg"></head></html>`, "text/html")
		finalRequest, err := http.NewRequest(http.MethodGet, "https://final.example/posts/article", nil)
		if err != nil {
			t.Fatalf("build final request: %v", err)
		}
		resp.Request = finalRequest
		return resp, nil
	})}

	_, _, imgURL, err := fetchLinkMetadataWithClient(targetURL, client)
	if err != nil {
		t.Fatalf("fetch redirected metadata: %v", err)
	}
	if imgURL != "https://final.example/thumb.jpg" {
		t.Fatalf("relative image resolved against wrong base: %q", imgURL)
	}
}

func TestNormalizeLinkPreviewThumbnailLimitsDimensionsAndProducesJPEG(t *testing.T) {
	source := image.NewRGBA(image.Rect(0, 0, 1000, 1))
	var input bytes.Buffer
	if err := png.Encode(&input, source); err != nil {
		t.Fatalf("encode source PNG: %v", err)
	}

	thumbnail, width, height, sourceFormat, err := normalizeLinkPreviewThumbnail(input.Bytes(), 200)
	if err != nil {
		t.Fatalf("normalize thumbnail: %v", err)
	}
	if sourceFormat != "png" {
		t.Fatalf("unexpected source format: %q", sourceFormat)
	}
	if width != 200 || height != 1 {
		t.Fatalf("unexpected normalized dimensions: %dx%d", width, height)
	}

	decoded, format, err := image.Decode(bytes.NewReader(thumbnail))
	if err != nil {
		t.Fatalf("decode normalized thumbnail: %v", err)
	}
	if format != "jpeg" {
		t.Fatalf("normalized thumbnail must be JPEG, got %q", format)
	}
	if decoded.Bounds().Dx() != int(width) || decoded.Bounds().Dy() != int(height) {
		t.Fatalf("encoded dimensions differ from metadata: %dx%d", decoded.Bounds().Dx(), decoded.Bounds().Dy())
	}
}

func TestNormalizeLinkPreviewThumbnailInvalidImageFallsBack(t *testing.T) {
	thumbnail, width, height, _, err := normalizeLinkPreviewThumbnail([]byte("not-an-image"), 200)
	if err == nil {
		t.Fatal("expected invalid image error")
	}
	if thumbnail != nil || width != 0 || height != 0 {
		t.Fatalf("invalid image must return empty fallback, got bytes=%d dimensions=%dx%d", len(thumbnail), width, height)
	}
}

func TestBuildLinkExtendedTextMessageWithThumbnail(t *testing.T) {
	data := &LinkStruct{
		Text:        "Confira https://example.com/article",
		Title:       "Example",
		Description: "An example article",
	}
	matchedText := "https://example.com/article"
	thumbnail := []byte{0xff, 0xd8, 0xff, 0xd9}

	message := buildLinkExtendedTextMessage(data, matchedText, thumbnail, 200, 120)

	if message.GetText() != data.Text {
		t.Fatalf("unexpected text: %q", message.GetText())
	}
	if message.GetMatchedText() != matchedText {
		t.Fatalf("unexpected matched URL: %q", message.GetMatchedText())
	}
	if message.GetTitle() != data.Title || message.GetDescription() != data.Description {
		t.Fatalf("metadata was not preserved: title=%q description=%q", message.GetTitle(), message.GetDescription())
	}
	if message.GetPreviewType() != waE2E.ExtendedTextMessage_IMAGE {
		t.Fatalf("unexpected preview type: %s", message.GetPreviewType())
	}
	if !bytes.Equal(message.GetJPEGThumbnail(), thumbnail) {
		t.Fatal("thumbnail was not preserved in JPEGThumbnail")
	}
	if message.GetThumbnailWidth() != 200 || message.GetThumbnailHeight() != 120 {
		t.Fatalf("unexpected thumbnail dimensions: %dx%d", message.GetThumbnailWidth(), message.GetThumbnailHeight())
	}
	if message.GetContextInfo() != nil {
		t.Fatal("common link preview must not include context info or external ad reply")
	}

	encoded, err := proto.Marshal(message)
	if err != nil {
		t.Fatalf("marshal message: %v", err)
	}
	if bytes.Count(encoded, thumbnail) != 1 {
		t.Fatalf("thumbnail must occur exactly once in protobuf, got %d", bytes.Count(encoded, thumbnail))
	}
}

func TestResolvedMergedLinkPayloadPreservesConsumerMetadataAndSimpleShape(t *testing.T) {
	original := LinkStruct{
		Text:  "Assista a live",
		Url:   "https://youtu.be/live-id",
		Title: "Titulo fornecido",
	}
	resolvedText, matchedText := resolveLinkTextAndMatchedURL(original.Text, original.Url)
	working := original
	working.Text = resolvedText
	working = mergeLinkMetadata(working, "Titulo remoto", "Canal remoto", "https://img.example/thumb.jpg")
	thumbnail := []byte{0xff, 0xd8, 0xff, 0xd9}
	message := buildLinkExtendedTextMessage(&working, matchedText, thumbnail, 200, 120)

	if original.Text != "Assista a live" || original.Description != "" || original.ImgUrl != "" {
		t.Fatalf("pure resolution/merge flow mutated caller input: %+v", original)
	}
	if message.GetTitle() != "Titulo fornecido" || message.GetDescription() != "Canal remoto" {
		t.Fatalf("consumer metadata precedence was not preserved: title=%q description=%q", message.GetTitle(), message.GetDescription())
	}
	if !strings.Contains(message.GetText(), message.GetMatchedText()) {
		t.Fatalf("payload text %q does not contain MatchedText %q", message.GetText(), message.GetMatchedText())
	}
	if message.GetContextInfo() != nil {
		t.Fatal("simple link payload must not contain ContextInfo")
	}
	encoded, err := proto.Marshal(message)
	if err != nil {
		t.Fatalf("marshal integrated payload: %v", err)
	}
	if bytes.Count(encoded, thumbnail) != 1 {
		t.Fatalf("thumbnail must occur exactly once, got %d", bytes.Count(encoded, thumbnail))
	}
}

func TestMergeLinkMetadataPreservesAllConsumerFields(t *testing.T) {
	original := LinkStruct{
		Title:       "Titulo fornecido",
		Description: "Descricao fornecida",
		ImgUrl:      "https://consumer.example/thumb.jpg",
	}
	merged := mergeLinkMetadata(original, "Titulo remoto", "Descricao remota", "https://remote.example/thumb.jpg")

	if merged.Title != original.Title || merged.Description != original.Description || merged.ImgUrl != original.ImgUrl {
		t.Fatalf("consumer metadata was overwritten: %+v", merged)
	}
}

func TestBuildLinkExtendedTextMessageWithoutThumbnail(t *testing.T) {
	data := &LinkStruct{
		Text:        "Confira https://example.com/article",
		Title:       "Example",
		Description: "An example article",
	}
	matchedText := "https://example.com/article"

	message := buildLinkExtendedTextMessage(data, matchedText, nil, 0, 0)

	if message.GetMatchedText() != matchedText {
		t.Fatalf("unexpected matched URL: %q", message.GetMatchedText())
	}
	if message.GetPreviewType() != waE2E.ExtendedTextMessage_PLACEHOLDER {
		t.Fatalf("unexpected preview type: %s", message.GetPreviewType())
	}
	if len(message.GetJPEGThumbnail()) != 0 {
		t.Fatal("message without an image must not include a thumbnail")
	}
	if message.ThumbnailWidth != nil || message.ThumbnailHeight != nil {
		t.Fatal("message without an image must not include thumbnail dimensions")
	}
	if message.GetContextInfo() != nil {
		t.Fatal("common link preview must not include context info or external ad reply")
	}
}

func TestBuildLinkExtendedTextMessageRejectsThumbnailWithoutDimensions(t *testing.T) {
	data := &LinkStruct{Text: "Confira https://example.com/article"}

	message := buildLinkExtendedTextMessage(data, "https://example.com/article", []byte{0xff, 0xd8}, 0, 120)

	if message.GetPreviewType() != waE2E.ExtendedTextMessage_PLACEHOLDER {
		t.Fatalf("unexpected preview type: %s", message.GetPreviewType())
	}
	if len(message.GetJPEGThumbnail()) != 0 || message.ThumbnailWidth != nil || message.ThumbnailHeight != nil {
		t.Fatal("thumbnail with invalid dimensions must be discarded")
	}
}
