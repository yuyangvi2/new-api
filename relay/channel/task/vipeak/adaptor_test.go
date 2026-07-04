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
