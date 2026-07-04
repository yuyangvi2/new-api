package vipeak

import (
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

func TestBuildRequestUsesCanonicalSeedanceModel(t *testing.T) {
	req := &relaycommon.TaskSubmitReq{
		Model:    "seedance",
		Prompt:   "make it move",
		Image:    "https://example.com/input.png",
		Duration: 5,
	}

	body := buildRequest(req, "seedance")

	if got := body["provider"]; got != "seedance" {
		t.Fatalf("provider = %v, want seedance", got)
	}
	if got := body["model"]; got != "dreamina-seedance-2-0-260128" {
		t.Fatalf("model = %v, want dreamina-seedance-2-0-260128", got)
	}
	if got := body["seedanceMode"]; got != "reference_images" {
		t.Fatalf("seedanceMode = %v, want reference_images", got)
	}
	refs, ok := body["referenceImages"].([]map[string]any)
	if !ok || len(refs) != 1 {
		t.Fatalf("referenceImages = %#v, want one reference image", body["referenceImages"])
	}
	if got := refs[0]["url"]; got != "https://example.com/input.png" {
		t.Fatalf("reference image url = %v, want input URL", got)
	}
}

func TestBuildRequestUsesCanonicalSeedanceFastModel(t *testing.T) {
	req := &relaycommon.TaskSubmitReq{Model: "seedance-fast"}

	body := buildRequest(req, "seedance-fast")

	if got := body["model"]; got != "dreamina-seedance-2-0-fast-260128" {
		t.Fatalf("model = %v, want dreamina-seedance-2-0-fast-260128", got)
	}
}

func TestBuildRequestWithImageResolverConvertsAndDeduplicatesSeedanceImages(t *testing.T) {
	req := &relaycommon.TaskSubmitReq{
		Model:  "seedance",
		Image:  "raw-base64",
		Images: []string{"raw-base64", "https://example.com/other.png"},
	}

	body, err := buildRequestWithImageResolver(req, "seedance", func(img string) (string, error) {
		if img == "raw-base64" {
			return "https://cos.example.com/input.png", nil
		}
		return img, nil
	})
	if err != nil {
		t.Fatalf("buildRequestWithImageResolver returned error: %v", err)
	}

	refs, ok := body["referenceImages"].([]map[string]any)
	if !ok || len(refs) != 2 {
		t.Fatalf("referenceImages = %#v, want two deduplicated reference images", body["referenceImages"])
	}
	if got := refs[0]["url"]; got != "https://cos.example.com/input.png" {
		t.Fatalf("first reference image url = %v, want converted COS URL", got)
	}
	if got := refs[1]["url"]; got != "https://example.com/other.png" {
		t.Fatalf("second reference image url = %v, want original URL", got)
	}
}

func TestHasNonURLImage(t *testing.T) {
	if hasNonURLImage(&relaycommon.TaskSubmitReq{Images: []string{"https://example.com/input.png"}}) {
		t.Fatal("hasNonURLImage returned true for http URL")
	}
	if !hasNonURLImage(&relaycommon.TaskSubmitReq{Images: []string{"raw-base64"}}) {
		t.Fatal("hasNonURLImage returned false for base64 image")
	}
}

func TestNormalizeBaseURLStripsKnownAPIPaths(t *testing.T) {
	tests := map[string]string{
		"https://www.123vips.com":                       "https://www.123vips.com",
		"https://www.123vips.com/":                      "https://www.123vips.com",
		"https://www.123vips.com/api":                   "https://www.123vips.com",
		"https://www.123vips.com/api/advanced/generate": "https://www.123vips.com",
		"https://www.123vips.com/api/wan27/image-edit":  "https://www.123vips.com",
	}

	for input, want := range tests {
		if got := normalizeBaseURL(input); got != want {
			t.Fatalf("normalizeBaseURL(%q) = %q, want %q", input, got, want)
		}
	}
}
