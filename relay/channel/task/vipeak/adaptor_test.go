package vipeak

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestBuildRequestRoutesOfficialSeedanceModelByImageInput(t *testing.T) {
	textReq := &relaycommon.TaskSubmitReq{
		Model:  "doubao-seedance-2-0-260128",
		Prompt: "make it move",
	}

	textBody := buildRequest(textReq, "doubao-seedance-2-0-260128")

	if got := textBody["model"]; got != "seedance2.0_direct" {
		t.Fatalf("text-only model = %v, want seedance2.0_direct", got)
	}
	if _, ok := textBody["seedanceMode"]; ok {
		t.Fatalf("text-only request unexpectedly set seedanceMode: %#v", textBody["seedanceMode"])
	}

	imageReq := &relaycommon.TaskSubmitReq{
		Model:  "doubao-seedance-2-0-260128",
		Prompt: "make it move",
		Image:  "https://example.com/input.png",
	}

	imageBody := buildRequest(imageReq, "doubao-seedance-2-0-260128")

	if got := imageBody["model"]; got != "seedance2.0_vision" {
		t.Fatalf("image model = %v, want seedance2.0_vision", got)
	}
	if got := imageBody["seedanceMode"]; got != "reference_images" {
		t.Fatalf("seedanceMode = %v, want reference_images", got)
	}
}

func TestBuildRequestRoutesOfficialSeedanceFastModelByImageInput(t *testing.T) {
	textBody := buildRequest(
		&relaycommon.TaskSubmitReq{Model: "doubao-seedance-2-0-fast-260128"},
		"doubao-seedance-2-0-fast-260128",
	)

	if got := textBody["model"]; got != "seedance2.0_fast_direct" {
		t.Fatalf("text-only fast model = %v, want seedance2.0_fast_direct", got)
	}

	imageBody := buildRequest(
		&relaycommon.TaskSubmitReq{
			Model: "doubao-seedance-2-0-fast-260128",
			Image: "https://example.com/input.png",
		},
		"doubao-seedance-2-0-fast-260128",
	)

	if got := imageBody["model"]; got != "seedance2.0_fast_vision" {
		t.Fatalf("image fast model = %v, want seedance2.0_fast_vision", got)
	}
}

func TestEstimateBillingUsesAPIZSeedancePricingForDreaminaStandard(t *testing.T) {
	originalQuotaPerUnit := common.QuotaPerUnit
	originalExchangeRate := operation_setting.USDExchangeRate
	t.Cleanup(func() {
		common.QuotaPerUnit = originalQuotaPerUnit
		operation_setting.USDExchangeRate = originalExchangeRate
	})
	common.QuotaPerUnit = 1_000_000
	operation_setting.USDExchangeRate = 1

	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set("task_request", relaycommon.TaskSubmitReq{
		Model:    "dreamina-seedance-2-0-260128",
		Duration: 5,
		Size:     "16:9",
		Metadata: map[string]interface{}{
			"resolution":           "720p",
			"video_files":          []interface{}{"https://example.com/reference.mp4"},
			"input_video_duration": 5,
		},
	})

	adaptor := &TaskAdaptor{}
	ratios := adaptor.EstimateBilling(ctx, &relaycommon.RelayInfo{
		OriginModelName: "dreamina-seedance-2-0-260128",
		PriceData: types.PriceData{
			Quota:          4_968_000,
			GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 1},
		},
	})

	got, ok := ratios["seedance_estimated_price"]
	require.True(t, ok)
	assert.InDelta(t, 6_048_000.0/4_968_000.0, got, 0.000001)
}

func TestEstimateBillingUsesAPIZSeedancePricingForDreaminaFast(t *testing.T) {
	originalQuotaPerUnit := common.QuotaPerUnit
	originalExchangeRate := operation_setting.USDExchangeRate
	t.Cleanup(func() {
		common.QuotaPerUnit = originalQuotaPerUnit
		operation_setting.USDExchangeRate = originalExchangeRate
	})
	common.QuotaPerUnit = 1_000_000
	operation_setting.USDExchangeRate = 1

	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set("task_request", relaycommon.TaskSubmitReq{
		Model:    "dreamina-seedance-2-0-fast-260128",
		Duration: 5,
		Size:     "16:9",
		Metadata: map[string]interface{}{
			"resolution":           "720p",
			"referenceVideoUrls":   []interface{}{"https://example.com/reference.mp4"},
			"input_video_duration": 2.5,
		},
	})

	adaptor := &TaskAdaptor{}
	ratios := adaptor.EstimateBilling(ctx, &relaycommon.RelayInfo{
		OriginModelName: "dreamina-seedance-2-0-fast-260128",
		PriceData: types.PriceData{
			Quota:          3_996_000,
			GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 1},
		},
	})

	got, ok := ratios["seedance_estimated_price"]
	require.True(t, ok)
	assert.InDelta(t, 2_268_000.0/3_996_000.0, got, 0.000001)
}

func TestEnforceCoreRequestFieldsKeepsOfficialSeedanceCanonicalModel(t *testing.T) {
	req := &relaycommon.TaskSubmitReq{
		Model: "seedance2.0_vision",
		Image: "https://example.com/input.png",
	}
	body := map[string]any{
		"provider": "seedance",
		"model":    "doubao-seedance-2-0-260128",
	}

	enforceCoreRequestFields(body, req, "doubao-seedance-2-0-260128")

	if got := body["model"]; got != "seedance2.0_vision" {
		t.Fatalf("model = %v, want seedance2.0_vision", got)
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
