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
