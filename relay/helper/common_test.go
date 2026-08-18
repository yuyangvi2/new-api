package helper

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/dto"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type flushingRecorder struct {
	*httptest.ResponseRecorder
	flushes int
}

func (r *flushingRecorder) Flush() {
	r.flushes++
	r.ResponseRecorder.Flush()
}

func (r *flushingRecorder) CloseNotify() <-chan bool {
	return make(chan bool)
}

func (r *flushingRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, http.ErrNotSupported
}

func (r *flushingRecorder) Pusher() http.Pusher {
	return nil
}

func newStreamStateTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	return c, recorder
}

func TestNextPingDelayUsesTriggerThenActiveInterval(t *testing.T) {
	c, _ := newStreamStateTestContext()
	triggerDelay := 100 * time.Millisecond
	pingInterval := 10 * time.Millisecond

	initialDelay := NextPingDelay(c, triggerDelay, pingInterval)
	assert.Greater(t, initialDelay, time.Duration(0))
	assert.LessOrEqual(t, initialDelay, triggerDelay)

	state := getDownstreamStreamState(c)
	require.NotNil(t, state)
	state.mu.Lock()
	state.startedAt = time.Now().Add(-triggerDelay)
	state.mu.Unlock()
	assert.Equal(t, time.Nanosecond, NextPingDelay(c, triggerDelay, pingInterval))

	markKeepaliveWritten(c)
	assert.Equal(t, pingInterval, NextPingDelay(c, triggerDelay, pingInterval))
}

func TestDownstreamStreamStateMarkers(t *testing.T) {
	c, _ := newStreamStateTestContext()
	require.NoError(t, PingData(c))
	assert.True(t, HasKeepaliveOnly(c))
	assert.False(t, HasSemanticOutput(c))
	assert.False(t, HasTerminalOutput(c))

	require.NoError(t, StringData(c, `{"ok":true}`))
	assert.False(t, HasKeepaliveOnly(c))
	assert.True(t, HasSemanticOutput(c))
	assert.False(t, HasTerminalOutput(c))

	Done(c)
	assert.True(t, HasTerminalOutput(c))
}

func TestDownstreamStreamStateMarksProtocolTerminal(t *testing.T) {
	claudeCtx, _ := newStreamStateTestContext()
	require.NoError(t, ClaudeData(claudeCtx, dto.ClaudeResponse{Type: "message_stop"}))
	assert.True(t, HasTerminalOutput(claudeCtx))

	responsesCtx, _ := newStreamStateTestContext()
	require.NoError(t, ResponseChunkData(responsesCtx, dto.ResponsesStreamResponse{Type: "response.failed"}, `{"type":"response.failed"}`))
	assert.True(t, HasTerminalOutput(responsesCtx))
}

func TestSuppressAttemptSemanticOutputResetsOnNextAttempt(t *testing.T) {
	c, recorder := newStreamStateTestContext()
	SuppressAttemptSemanticOutput(c)
	require.NoError(t, StringData(c, `{"suppressed":true}`))
	assert.Empty(t, recorder.Body.String())
	assert.False(t, HasSemanticOutput(c))

	BeginStreamAttempt(c)
	require.NoError(t, StringData(c, `{"visible":true}`))
	assert.Contains(t, recorder.Body.String(), "visible")
	assert.True(t, HasSemanticOutput(c))
}

func TestClaudeTerminalFramesForceFlush(t *testing.T) {
	recorder := &flushingRecorder{ResponseRecorder: httptest.NewRecorder()}
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	state := getStreamFlushState(c)
	state.events = streamImmediateEventCount + 1
	state.pendingBytes = 1
	state.lastFlush = time.Now()

	require.NoError(t, ClaudeData(c, dto.ClaudeResponse{Type: "message_stop"}))
	assert.Greater(t, recorder.flushes, 0)
}
