package servers

import "testing"

func TestFaviconCandidatesPrefersStoredURLAndFallsBackToRoot(t *testing.T) {
	server := Server{
		URL:     "jellyfin.torden.tech",
		Favicon: "https://jellyfin.torden.tech/favicon.bc8d51405ec040305a87.ico",
	}

	got := faviconCandidates(server)

	if len(got) != 2 {
		t.Fatalf("expected 2 favicon candidates, got %d: %#v", len(got), got)
	}

	if got[0] != "https://jellyfin.torden.tech/favicon.bc8d51405ec040305a87.ico" {
		t.Fatalf("unexpected primary favicon URL: %q", got[0])
	}

	if got[1] != "https://jellyfin.torden.tech/favicon.ico" {
		t.Fatalf("unexpected fallback favicon URL: %q", got[1])
	}
}

func TestResolveServerURLSupportsRelativeFaviconPaths(t *testing.T) {
	server := Server{URL: "example.com"}

	got := resolveServerURL(server, "icons/site.ico")

	if got != "https://example.com/icons/site.ico" {
		t.Fatalf("unexpected resolved URL: %q", got)
	}
}

func TestIsAllowedFaviconContentType(t *testing.T) {
	tests := map[string]bool{
		"image/x-icon":                true,
		"image/png":                   true,
		"image/svg+xml; charset=utf8": true,
		"application/octet-stream":    true,
		"text/html":                   false,
	}

	for contentType, want := range tests {
		if got := isAllowedFaviconContentType(contentType); got != want {
			t.Fatalf("content type %q: got %v want %v", contentType, got, want)
		}
	}
}
