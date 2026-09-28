package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInsertInitialToken(t *testing.T) {
	truncateTables(t)

	originalGenerateDefaultToken := constant.GenerateDefaultToken
	originalDefaultUseAutoGroup := setting.DefaultUseAutoGroup
	t.Cleanup(func() {
		constant.GenerateDefaultToken = originalGenerateDefaultToken
		setting.DefaultUseAutoGroup = originalDefaultUseAutoGroup
	})

	constant.GenerateDefaultToken = true
	setting.DefaultUseAutoGroup = true
	require.NoError(t, InsertInitialToken(DB, 101, "new-user"))

	var token Token
	require.NoError(t, DB.Where("user_id = ?", 101).First(&token).Error)
	assert.Equal(t, "new-user的初始令牌", token.Name)
	assert.Equal(t, common.TokenStatusEnabled, token.Status)
	assert.Equal(t, "auto", token.Group)
	assert.True(t, token.CrossGroupRetry)
	assert.True(t, token.UnlimitedQuota)
	assert.False(t, token.ModelLimitsEnabled)
	assert.Equal(t, int64(-1), token.ExpiredTime)
	assert.NotEmpty(t, token.Key)
}

func TestInsertInitialTokenDisabled(t *testing.T) {
	truncateTables(t)

	originalGenerateDefaultToken := constant.GenerateDefaultToken
	t.Cleanup(func() {
		constant.GenerateDefaultToken = originalGenerateDefaultToken
	})
	constant.GenerateDefaultToken = false

	require.NoError(t, InsertInitialToken(DB, 102, "new-user"))

	var count int64
	require.NoError(t, DB.Model(&Token{}).Where("user_id = ?", 102).Count(&count).Error)
	assert.Zero(t, count)
}
