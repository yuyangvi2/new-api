package router

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeedanceAPIV3RoutesSupportVolcengineAndRootPrefixes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()

	require.NotPanics(t, func() {
		SetVideoRouter(engine)
	})

	routes := map[string]bool{}
	for _, route := range engine.Routes() {
		routes[route.Method+" "+route.Path] = true
	}

	assert.True(t, routes[http.MethodPost+" /volcengine/api/v3/contents/generations/tasks"])
	assert.True(t, routes[http.MethodGet+" /volcengine/api/v3/contents/generations/tasks/:task_id"])
	assert.True(t, routes[http.MethodPost+" /api/v3/contents/generations/tasks"])
	assert.True(t, routes[http.MethodGet+" /api/v3/contents/generations/tasks/:task_id"])
}

func TestSeedanceOfficialAssetRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()

	require.NotPanics(t, func() {
		SetVideoRouter(engine)
	})

	routes := map[string]bool{}
	for _, route := range engine.Routes() {
		routes[route.Method+" "+route.Path] = true
	}

	assert.True(t, routes[http.MethodPost+" /v1/seedance-official/asset-groups"])
	assert.True(t, routes[http.MethodPost+" /v1/seedance-official/asset-groups/query"])
	assert.True(t, routes[http.MethodGet+" /v1/seedance-official/asset-groups/:group_id"])
	assert.True(t, routes[http.MethodPut+" /v1/seedance-official/asset-groups/:group_id"])
	assert.True(t, routes[http.MethodDelete+" /v1/seedance-official/asset-groups/:group_id"])
	assert.True(t, routes[http.MethodPost+" /v1/seedance-official/assets"])
	assert.True(t, routes[http.MethodPost+" /v1/seedance-official/assets/query"])
	assert.True(t, routes[http.MethodGet+" /v1/seedance-official/assets/:asset_id"])
	assert.True(t, routes[http.MethodPut+" /v1/seedance-official/assets/:asset_id"])
	assert.True(t, routes[http.MethodDelete+" /v1/seedance-official/assets/:asset_id"])
	assert.True(t, routes[http.MethodPost+" /v1/seedance-official/real-person-auth/sessions"])
	assert.True(t, routes[http.MethodPost+" /v1/seedance-official/real-person-auth/asset-group/by-byted-token"])
}

func TestXaiVideoGenerationRouteSupportsOfficialPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()

	require.NotPanics(t, func() {
		SetVideoRouter(engine)
	})

	routes := map[string]bool{}
	for _, route := range engine.Routes() {
		routes[route.Method+" "+route.Path] = true
	}

	assert.True(t, routes[http.MethodPost+" /v1/videos/generations"])
	assert.True(t, routes[http.MethodPost+" /v1/video/generations"])
	assert.True(t, routes[http.MethodPost+" /v1/videos"])
	assert.True(t, routes[http.MethodPost+" /v1/videos/:video_id/remix"])
}
