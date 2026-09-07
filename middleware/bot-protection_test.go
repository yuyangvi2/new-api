package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type botProtectionSettingsSnapshot struct {
	capRegisterCheckEnabled bool
	capLoginCheckEnabled    bool
	capVerifyEndpoint       string
	capRegisterSiteKey      string
	capRegisterSecretKey    string
	capLoginSiteKey         string
	capLoginSecretKey       string
	turnstileCheckEnabled   bool
	turnstileSiteKey        string
	turnstileSecretKey      string
	httpClient              *http.Client
}

func snapshotBotProtectionSettings() botProtectionSettingsSnapshot {
	return botProtectionSettingsSnapshot{
		capRegisterCheckEnabled: common.CapRegisterCheckEnabled,
		capLoginCheckEnabled:    common.CapLoginCheckEnabled,
		capVerifyEndpoint:       common.CapVerifyEndpoint,
		capRegisterSiteKey:      common.CapRegisterSiteKey,
		capRegisterSecretKey:    common.CapRegisterSecretKey,
		capLoginSiteKey:         common.CapLoginSiteKey,
		capLoginSecretKey:       common.CapLoginSecretKey,
		turnstileCheckEnabled:   common.TurnstileCheckEnabled,
		turnstileSiteKey:        common.TurnstileSiteKey,
		turnstileSecretKey:      common.TurnstileSecretKey,
		httpClient:              botProtectionHTTPClient,
	}
}

func (s botProtectionSettingsSnapshot) restore() {
	common.CapRegisterCheckEnabled = s.capRegisterCheckEnabled
	common.CapLoginCheckEnabled = s.capLoginCheckEnabled
	common.CapVerifyEndpoint = s.capVerifyEndpoint
	common.CapRegisterSiteKey = s.capRegisterSiteKey
	common.CapRegisterSecretKey = s.capRegisterSecretKey
	common.CapLoginSiteKey = s.capLoginSiteKey
	common.CapLoginSecretKey = s.capLoginSecretKey
	common.TurnstileCheckEnabled = s.turnstileCheckEnabled
	common.TurnstileSiteKey = s.turnstileSiteKey
	common.TurnstileSecretKey = s.turnstileSecretKey
	botProtectionHTTPClient = s.httpClient
}

func configureCapForTests(endpoint string) {
	common.TurnstileCheckEnabled = false
	common.CapRegisterCheckEnabled = true
	common.CapLoginCheckEnabled = true
	common.CapVerifyEndpoint = endpoint
	common.CapRegisterSiteKey = "register-site"
	common.CapRegisterSecretKey = "register-secret"
	common.CapLoginSiteKey = "login-site"
	common.CapLoginSecretKey = "login-secret"
}

func performBotProtectionRequest(scene BotProtectionScene, token string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/protected", BotProtectionCheck(scene), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/protected", nil)
	if token != "" {
		req.Header.Set(CaptchaTokenHeader, token)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func TestBotProtectionCheckDisabledAllowsRequest(t *testing.T) {
	snapshot := snapshotBotProtectionSettings()
	t.Cleanup(snapshot.restore)
	common.CapRegisterCheckEnabled = false
	common.TurnstileCheckEnabled = false

	recorder := performBotProtectionRequest(BotProtectionSceneRegister, "")

	assert.Equal(t, http.StatusNoContent, recorder.Code)
}

func TestBotProtectionCheckRequiresHeaderToken(t *testing.T) {
	snapshot := snapshotBotProtectionSettings()
	t.Cleanup(snapshot.restore)
	configureCapForTests("http://unused.invalid")

	recorder := performBotProtectionRequest(BotProtectionSceneRegister, "")

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "human verification")
}

func TestBotProtectionCheckUsesSceneSpecificCapCredentials(t *testing.T) {
	snapshot := snapshotBotProtectionSettings()
	t.Cleanup(snapshot.restore)

	type verificationRequest struct {
		Secret   string `json:"secret"`
		Response string `json:"response"`
	}
	var mu sync.Mutex
	requests := make(map[string]verificationRequest)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload verificationRequest
		require.NoError(t, common.DecodeJson(r.Body, &payload))
		mu.Lock()
		requests[r.URL.Path] = payload
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	t.Cleanup(server.Close)
	configureCapForTests(server.URL)

	registerRecorder := performBotProtectionRequest(BotProtectionSceneRegister, "register-token")
	loginRecorder := performBotProtectionRequest(BotProtectionSceneLogin, "login-token")

	assert.Equal(t, http.StatusNoContent, registerRecorder.Code)
	assert.Equal(t, http.StatusNoContent, loginRecorder.Code)
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, verificationRequest{Secret: "register-secret", Response: "register-token"}, requests["/register-site/siteverify"])
	assert.Equal(t, verificationRequest{Secret: "login-secret", Response: "login-token"}, requests["/login-site/siteverify"])
}

func TestBotProtectionCheckRejectsInvalidAndReplayedCapToken(t *testing.T) {
	snapshot := snapshotBotProtectionSettings()
	t.Cleanup(snapshot.restore)

	var consumed atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if consumed.CompareAndSwap(false, true) {
			_, _ = w.Write([]byte(`{"success":true}`))
			return
		}
		_, _ = w.Write([]byte(`{"success":false}`))
	}))
	t.Cleanup(server.Close)
	configureCapForTests(server.URL)

	first := performBotProtectionRequest(BotProtectionSceneLogin, "one-time-token")
	second := performBotProtectionRequest(BotProtectionSceneLogin, "one-time-token")

	assert.Equal(t, http.StatusNoContent, first.Code)
	assert.Equal(t, http.StatusBadRequest, second.Code)
	assert.Contains(t, second.Body.String(), "verification failed")
}

func TestBotProtectionCheckTreatsCapNotFoundAsInvalidToken(t *testing.T) {
	snapshot := snapshotBotProtectionSettings()
	t.Cleanup(snapshot.restore)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"success":false,"error":"Invalid token"}`))
	}))
	t.Cleanup(server.Close)
	configureCapForTests(server.URL)

	recorder := performBotProtectionRequest(BotProtectionSceneLogin, "replayed-token")

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "verification failed")
}

func TestBotProtectionCheckFailsClosedWhenCapUnavailable(t *testing.T) {
	snapshot := snapshotBotProtectionSettings()
	t.Cleanup(snapshot.restore)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	t.Cleanup(server.Close)
	configureCapForTests(server.URL)
	botProtectionHTTPClient = &http.Client{Timeout: 10 * time.Millisecond}

	recorder := performBotProtectionRequest(BotProtectionSceneRegister, "token")

	assert.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "temporarily unavailable")
}

func TestBotProtectionCheckRejectsOversizedResponse(t *testing.T) {
	snapshot := snapshotBotProtectionSettings()
	t.Cleanup(snapshot.restore)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"padding":"` + strings.Repeat("x", 70*1024) + `"}`))
	}))
	t.Cleanup(server.Close)
	configureCapForTests(server.URL)

	recorder := performBotProtectionRequest(BotProtectionSceneRegister, "token")

	assert.Equal(t, http.StatusServiceUnavailable, recorder.Code)
}

func TestBotProtectionCheckDoesNotForwardSecretAcrossRedirects(t *testing.T) {
	snapshot := snapshotBotProtectionSettings()
	t.Cleanup(snapshot.restore)

	var redirected atomic.Bool
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		redirected.Store(true)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	t.Cleanup(target.Close)
	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	t.Cleanup(redirector.Close)
	configureCapForTests(redirector.URL)

	recorder := performBotProtectionRequest(BotProtectionSceneRegister, "token")

	assert.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	assert.False(t, redirected.Load())
}
