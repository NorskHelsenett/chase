package servers

import (
	"net/url"
	"strings"
	"testing"
)

func TestExtractSiteMetadataCapturesManifest(t *testing.T) {
	html := `<head><link rel="manifest" href="manifest.json"><link rel="shortcut icon" href="favicon.bc8d51405ec040305a87.ico"><title>Jellyfin</title></head>`

	meta := extractSiteMetadata(strings.NewReader(html))

	if meta.ManifestURL != "manifest.json" {
		t.Fatalf("unexpected manifest URL: %q", meta.ManifestURL)
	}
	if meta.Favicon != "favicon.bc8d51405ec040305a87.ico" {
		t.Fatalf("unexpected favicon: %q", meta.Favicon)
	}
}

func TestResolveSiteMetadataURLs(t *testing.T) {
	baseURL, err := url.Parse("https://jellyfin.torden.tech/web/index.html")
	if err != nil {
		t.Fatal(err)
	}

	meta := SiteMetadata{
		Favicon:     "favicon.bc8d51405ec040305a87.ico",
		ManifestURL: "manifest.json",
		OGImage:     "/images/cover.png",
	}

	resolveSiteMetadataURLs(&meta, baseURL)

	if meta.Favicon != "https://jellyfin.torden.tech/web/favicon.bc8d51405ec040305a87.ico" {
		t.Fatalf("unexpected resolved favicon: %q", meta.Favicon)
	}
	if meta.ManifestURL != "https://jellyfin.torden.tech/web/manifest.json" {
		t.Fatalf("unexpected resolved manifest URL: %q", meta.ManifestURL)
	}
	if meta.OGImage != "https://jellyfin.torden.tech/images/cover.png" {
		t.Fatalf("unexpected resolved og image: %q", meta.OGImage)
	}
}

func TestShouldPreferManifestIconForFingerprintFavicon(t *testing.T) {
	meta := SiteMetadata{
		Favicon:     "https://jellyfin.torden.tech/web/favicon.bc8d51405ec040305a87.ico",
		ManifestURL: "https://jellyfin.torden.tech/web/manifest.json",
	}

	if !shouldPreferManifestIcon(meta) {
		t.Fatal("expected manifest icon to be preferred for fingerprinted favicon")
	}
}

func TestShouldNotPreferManifestIconForStableFavicon(t *testing.T) {
	meta := SiteMetadata{
		Favicon:     "https://example.com/favicon.ico",
		ManifestURL: "https://example.com/manifest.json",
	}

	if shouldPreferManifestIcon(meta) {
		t.Fatal("did not expect manifest icon to be preferred for stable favicon")
	}
}

func TestParseManifestIconSize(t *testing.T) {
	if got := parseManifestIconSize("72x72 512x512"); got != 512 {
		t.Fatalf("unexpected size: %d", got)
	}
	if got := parseManifestIconSize("any"); got != 1024 {
		t.Fatalf("unexpected any size: %d", got)
	}
}
