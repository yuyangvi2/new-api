package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupDistributorTestDB(t *testing.T) {
	t.Helper()

	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	originalSQLitePath := common.SQLitePath
	originalMainDatabaseType := common.MainDatabaseType()
	originalLogDatabaseType := common.LogDatabaseType()
	originalSQLDSN, hadSQLDSN := os.LookupEnv("SQL_DSN")
	originalDB := model.DB
	originalLogDB := model.LOG_DB

	t.Cleanup(func() {
		common.MemoryCacheEnabled = originalMemoryCacheEnabled
		common.SQLitePath = originalSQLitePath
		common.SetDatabaseTypes(originalMainDatabaseType, originalLogDatabaseType)
		model.DB = originalDB
		model.LOG_DB = originalLogDB
		if hadSQLDSN {
			require.NoError(t, os.Setenv("SQL_DSN", originalSQLDSN))
		} else {
			require.NoError(t, os.Unsetenv("SQL_DSN"))
		}
	})

	gin.SetMode(gin.TestMode)
	common.MemoryCacheEnabled = false
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	common.SQLitePath = "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	require.NoError(t, os.Setenv("SQL_DSN", "local"))

	require.NoError(t, model.InitDB())
	if model.DB != nil {
		sqlDB, err := model.DB.DB()
		if err == nil {
			require.NoError(t, sqlDB.Close())
		}
	}

	db, err := gorm.Open(sqlite.Open(common.SQLitePath), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.Ability{}))
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
}

func TestDistributeRequiredChannelTypeFiltersCandidates(t *testing.T) {
	setupDistributorTestDB(t)

	highPriority := int64(10)
	lowPriority := int64(1)
	weight := uint(100)
	channels := []model.Channel{
		{
			Id:       1,
			Type:     constant.ChannelTypeKling,
			Key:      "wrong-key",
			Name:     "wrong type",
			Status:   common.ChannelStatusEnabled,
			Models:   "shared-video-model",
			Group:    "default",
			Priority: &highPriority,
			Weight:   &weight,
		},
		{
			Id:       2,
			Type:     constant.ChannelTypeMiniMax,
			Key:      "right-key",
			Name:     "right type",
			Status:   common.ChannelStatusEnabled,
			Models:   "shared-video-model",
			Group:    "default",
			Priority: &lowPriority,
			Weight:   &weight,
		},
	}
	require.NoError(t, model.DB.Create(&channels).Error)
	require.NoError(t, model.DB.Create(&[]model.Ability{
		{Group: "default", Model: "shared-video-model", ChannelId: 1, Enabled: true, Priority: &highPriority, Weight: weight},
		{Group: "default", Model: "shared-video-model", ChannelId: 2, Enabled: true, Priority: &lowPriority, Weight: weight},
	}).Error)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("id", 7)
		common.SetContextKey(c, constant.ContextKeyUsingGroup, "default")
	})
	router.POST("/v1/video_generation", RequireChannelType(constant.ChannelTypeMiniMax), Distribute(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"channel_id":   common.GetContextKeyInt(c, constant.ContextKeyChannelId),
			"channel_type": common.GetContextKeyInt(c, constant.ContextKeyChannelType),
		})
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/video_generation",
		strings.NewReader(`{"model":"shared-video-model","prompt":"test"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)

	var body struct {
		ChannelID   int `json:"channel_id"`
		ChannelType int `json:"channel_type"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &body))
	assert.Equal(t, 2, body.ChannelID)
	assert.Equal(t, constant.ChannelTypeMiniMax, body.ChannelType)
}

func TestDistributeNoAvailableChannelReturnsSanitizedModelNotFound(t *testing.T) {
	setupDistributorTestDB(t)
	require.NoError(t, i18n.Init())

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("id", 7)
		common.SetContextKey(c, constant.ContextKeyUsingGroup, "openai")
	})
	router.POST("/v1/chat/completions", Distribute(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/chat/completions",
		strings.NewReader(`{"model":"qa-nonexistent-model","messages":[{"role":"user","content":"hi"}]}`),
	)
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusNotFound, recorder.Code)

	var body struct {
		Error types.OpenAIError `json:"error"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &body))
	assert.Equal(t, string(types.ErrorCodeModelNotFound), body.Error.Code)
	assert.Contains(t, body.Error.Message, "requested model is not available")
	assert.NotContains(t, body.Error.Message, "openai")
	assert.NotContains(t, body.Error.Message, "qa-nonexistent-model")
	assert.NotContains(t, body.Error.Message, "group")
	assert.NotContains(t, body.Error.Message, "channel")
	assert.NotContains(t, body.Error.Message, "distributor")
}

func TestDistributeInvalidJSONMessageIsNotDoubleWrapped(t *testing.T) {
	setupDistributorTestDB(t)
	require.NoError(t, i18n.Init())

	router := gin.New()
	router.POST("/v1/chat/completions", Distribute(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/chat/completions",
		strings.NewReader(`{"model":"gpt-5.4-mini","messages":[`),
	)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept-Language", "zh-CN")
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)

	var body struct {
		Error types.OpenAIError `json:"error"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &body))
	assert.Contains(t, body.Error.Message, "无效的请求：invalid JSON request body")
	assert.NotContains(t, body.Error.Message, "无效的请求：无效的请求")
	assert.NotContains(t, body.Error.Message, "无效的请求，无效的请求")
}
