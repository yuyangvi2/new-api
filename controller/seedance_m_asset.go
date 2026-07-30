package controller

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/seedancemasset"
)

const (
	seedanceMAssetGroupPath           = "/api/openapi-maas/exp/aicc/v2/asset-group"
	seedanceMAssetPath                = "/api/openapi-maas/exp/aicc/v2/asset"
	seedanceMRealPersonSessionPath    = "/api/openapi-maas/exp/aicc/v2/real-person-auth/sessions"
	seedanceMRealPersonBytedTokenPath = "/api/openapi-maas/exp/aicc/v2/real-person-auth/asset-group/by-byted-token"
	maxSeedanceMProxyResponseBytes    = 8 << 20
)

func CreateSeedanceMAssetGroup(c *gin.Context) {
	proxySeedanceMAssetRequest(c, http.MethodPost, seedanceMAssetGroupPath, true)
}

func QuerySeedanceMAssetGroups(c *gin.Context) {
	proxySeedanceMAssetRequest(c, http.MethodPost, seedanceMAssetGroupPath+"/query", true)
}

func GetSeedanceMAssetGroup(c *gin.Context) {
	proxySeedanceMAssetRequest(c, http.MethodGet, seedanceMAssetGroupPath+"/"+url.PathEscape(c.Param("group_id")), false)
}

func UpdateSeedanceMAssetGroup(c *gin.Context) {
	proxySeedanceMAssetRequest(c, http.MethodPut, seedanceMAssetGroupPath+"/"+url.PathEscape(c.Param("group_id")), true)
}

func DeleteSeedanceMAssetGroup(c *gin.Context) {
	proxySeedanceMAssetRequest(c, http.MethodDelete, seedanceMAssetGroupPath+"/"+url.PathEscape(c.Param("group_id")), false)
}

func CreateSeedanceMAsset(c *gin.Context) {
	proxySeedanceMAssetRequest(c, http.MethodPost, seedanceMAssetPath, true)
}

func QuerySeedanceMAssets(c *gin.Context) {
	proxySeedanceMAssetRequest(c, http.MethodPost, seedanceMAssetPath+"/query", true)
}

func GetSeedanceMAsset(c *gin.Context) {
	proxySeedanceMAssetRequest(c, http.MethodGet, seedanceMAssetPath+"/"+url.PathEscape(c.Param("asset_id")), false)
}

func UpdateSeedanceMAsset(c *gin.Context) {
	proxySeedanceMAssetRequest(c, http.MethodPut, seedanceMAssetPath+"/"+url.PathEscape(c.Param("asset_id")), true)
}

func DeleteSeedanceMAsset(c *gin.Context) {
	proxySeedanceMAssetRequest(c, http.MethodDelete, seedanceMAssetPath+"/"+url.PathEscape(c.Param("asset_id")), false)
}

func CreateSeedanceMRealPersonAuthSession(c *gin.Context) {
	proxySeedanceMAssetRequest(c, http.MethodPost, seedanceMRealPersonSessionPath, true)
}

func CreateSeedanceMRealPersonAssetGroupByBytedToken(c *gin.Context) {
	proxySeedanceMAssetRequest(c, http.MethodPost, seedanceMRealPersonBytedTokenPath, true)
}

func proxySeedanceMAssetRequest(c *gin.Context, method string, path string, hasBody bool) {
	settings, proxy, ok := getSelectedSeedanceMAssetSettings(c)
	if !ok {
		return
	}
	var payload map[string]any
	if hasBody {
		if err := common.DecodeJson(c.Request.Body, &payload); err != nil {
			common.ApiErrorMsg(c, "invalid request payload")
			return
		}
		delete(payload, "model")
		delete(payload, "group")
	}
	client, err := seedancemasset.New(settings, proxy)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	query := c.Request.URL.Query()
	query.Del("model")
	query.Del("group")
	status, header, body, err := client.Call(method, path, query, payload)
	if err != nil {
		common.ApiErrorMsg(c, "request failed")
		return
	}
	if len(body) > maxSeedanceMProxyResponseBytes {
		common.ApiErrorMsg(c, "response is too large")
		return
	}
	if status < http.StatusOK || status >= http.StatusMultipleChoices {
		c.JSON(status, gin.H{
			"error": gin.H{
				"message": "request failed",
			},
		})
		return
	}
	c.Data(status, firstNonEmpty(header.Get("Content-Type"), "application/json"), body)
}

func getSelectedSeedanceMAssetSettings(c *gin.Context) (*dto.SeedanceMSettings, string, bool) {
	channelType := common.GetContextKeyInt(c, constant.ContextKeyChannelType)
	if channelType != constant.ChannelTypeSeedanceM {
		common.ApiErrorMsg(c, "selected channel is not Seedance-M")
		return nil, "", false
	}
	channelID := common.GetContextKeyInt(c, constant.ContextKeyChannelId)
	channel, err := model.CacheGetChannel(channelID)
	if err != nil {
		common.ApiErrorMsg(c, "failed to load channel")
		return nil, "", false
	}
	settings := channel.GetOtherSettings().SeedanceM
	if settings == nil {
		common.ApiErrorMsg(c, "seedance_m settings are required")
		return nil, "", false
	}
	return settings, channel.GetSetting().Proxy, true
}
