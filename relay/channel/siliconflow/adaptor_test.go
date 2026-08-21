package siliconflow

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

func TestConvertImageRequestUsesAndBoundsBatchSize(t *testing.T) {
	batchSize := uint(3)
	request := dto.ImageRequest{
		Model:  "black-forest-labs/FLUX.1-schnell",
		Prompt: "cat",
		N:      common.GetPointer(uint(2)),
		Extra: map[string]json.RawMessage{
			"batch_size": json.RawMessage(`3`),
		},
	}

	converted, err := (&Adaptor{}).ConvertImageRequest(nil, nil, request)
	require.NoError(t, err)
	actual := converted.(*SFImageRequest)
	require.NotNil(t, actual.BatchSize)
	assert.Equal(t, batchSize, *actual.BatchSize)

	request.Extra["batch_size"] = json.RawMessage(`129`)
	_, err = (&Adaptor{}).ConvertImageRequest(nil, nil, request)
	require.Error(t, err)
}
