package router

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImageTaskRoutesAreRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	require.NotPanics(t, func() {
		SetRelayRouter(engine)
	})

	routes := map[string]bool{}
	for _, route := range engine.Routes() {
		routes[route.Method+" "+route.Path] = true
	}
	assert.True(t, routes[http.MethodPost+" /v1/images/tasks"])
	assert.True(t, routes[http.MethodGet+" /v1/images/tasks/:task_id"])
	assert.True(t, routes[http.MethodPost+" /pg/images/tasks"])
	assert.True(t, routes[http.MethodGet+" /pg/images/tasks/:task_id"])
}
