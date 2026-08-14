package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateChannelOpenAICompatibleUnlimitedBalanceDoesNotPersistSentinel(t *testing.T) {
	db := setupModelListControllerTestDB(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/dashboard/billing/subscription":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"object":"billing_subscription","has_payment_method":true,"hard_limit_usd":100000000,"system_hard_limit_usd":100000000}`))
		case "/v1/dashboard/billing/usage":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"object":"list","total_usage":250}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	initialBalance := 12.34
	channel := model.Channel{
		Type:    constant.ChannelTypeOpenAI,
		Name:    "compatible-unlimited",
		Key:     "sk-test",
		BaseURL: &server.URL,
		Balance: initialBalance,
	}
	require.NoError(t, db.Create(&channel).Error)

	_, err := updateChannelBalance(&channel)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unlimited quota placeholder")

	var stored model.Channel
	require.NoError(t, db.First(&stored, channel.Id).Error)
	assert.InDelta(t, initialBalance, stored.Balance, 0.0001)
	assert.Zero(t, stored.BalanceUpdatedTime)
}

func TestUpdateChannelOpenAICompatibleFiniteBalancePersists(t *testing.T) {
	db := setupModelListControllerTestDB(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/dashboard/billing/subscription":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"object":"billing_subscription","has_payment_method":true,"hard_limit_usd":25,"system_hard_limit_usd":25}`))
		case "/v1/dashboard/billing/usage":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"object":"list","total_usage":250}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	channel := model.Channel{
		Type:    constant.ChannelTypeOpenAI,
		Name:    "compatible-finite",
		Key:     "sk-test",
		BaseURL: &server.URL,
	}
	require.NoError(t, db.Create(&channel).Error)

	balance, err := updateChannelBalance(&channel)

	require.NoError(t, err)
	assert.InDelta(t, 22.5, balance, 0.0001)

	var stored model.Channel
	require.NoError(t, db.First(&stored, channel.Id).Error)
	assert.InDelta(t, 22.5, stored.Balance, 0.0001)
	assert.NotZero(t, stored.BalanceUpdatedTime)
}
