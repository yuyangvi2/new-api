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
