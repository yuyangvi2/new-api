package doubao

import (
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

func TestConvertToRequestPayloadUsesCanonicalModelName(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := relaycommon.TaskSubmitReq{
		Model:  "doubao-seedance-2-0-fast",
		Prompt: "make it cinematic",
	}

	body, err := adaptor.convertToRequestPayload(&req)
	if err != nil {
		t.Fatalf("convertToRequestPayload error = %v", err)
	}

	if body.Model != "doubao-seedance-2-0-fast-260128" {
		t.Fatalf("model = %s, want doubao-seedance-2-0-fast-260128", body.Model)
	}
}

func TestConvertToRequestPayloadResolvesUploadedImages(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := relaycommon.TaskSubmitReq{
		Model:  "doubao-seedance-2-0",
		Prompt: "make it cinematic",
		Images: []string{"raw-base64"},
	}

	body, err := adaptor.convertToRequestPayloadWithImageResolver(&req, func(img string) (string, error) {
		if img == "raw-base64" {
			return "https://cos.example.com/input.png", nil
		}
		return img, nil
	})
	if err != nil {
		t.Fatalf("convertToRequestPayloadWithImageResolver error = %v", err)
	}

	if len(body.Content) == 0 || body.Content[0].ImageURL == nil {
		t.Fatalf("content = %#v, want image content first", body.Content)
	}
	if got := body.Content[0].ImageURL.URL; got != "https://cos.example.com/input.png" {
		t.Fatalf("image url = %s, want converted COS URL", got)
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
