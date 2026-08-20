package volcengine

import (
	"bytes"
	"io"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/QuantumNous/new-api/common"
)

func HandleRESTAction(action string, idParam string, aliases map[string]string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		query := ctx.Request.URL.Query()
		query.Set("Action", action)
		ctx.Request.URL.RawQuery = query.Encode()

		if idParam != "" || len(aliases) > 0 {
			var raw []byte
			if ctx.Request.Body != nil {
				data, err := io.ReadAll(ctx.Request.Body)
				if err != nil {
					common.ApiErrorMsg(ctx, "invalid request payload")
					return
				}
				raw = data
			}
			idField := ""
			idValue := ""
			if idParam != "" {
				idField = "Id"
				idValue = ctx.Param(idParam)
			}
			body, err := patchRESTActionPayload(raw, idField, idValue, aliases)
			if err != nil {
				common.ApiErrorMsg(ctx, "invalid request payload")
				return
			}
			ctx.Request.Body = io.NopCloser(bytes.NewReader(body))
			ctx.Request.ContentLength = int64(len(body))
			ctx.Request.Header.Set("Content-Type", "application/json")
		}

		HandleAction(ctx)
	}
}

func patchRESTActionPayload(raw []byte, idField string, idValue string, aliases map[string]string) ([]byte, error) {
	payload := map[string]any{}
	if strings.TrimSpace(string(raw)) != "" {
		if err := common.Unmarshal(raw, &payload); err != nil {
			return nil, err
		}
		if payload == nil {
			payload = map[string]any{}
		}
	}

	if idField != "" {
		payload[idField] = idValue
	}
	for alias, canonical := range aliases {
		if _, exists := payload[canonical]; exists {
			continue
		}
		if value, exists := payload[alias]; exists {
			payload[canonical] = value
		}
	}
	return common.Marshal(payload)
}
