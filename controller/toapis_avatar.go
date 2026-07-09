package controller

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
)

const (
	toapisAvatarDefaultBaseURL = "https://toapis.com"
	toapisAvatarGroupsPath     = "/v1/videos/doubao-seedance-2-0/private-avatar/groups"
	toapisAvatarAssetsPath     = "/v1/videos/doubao-seedance-2-0/private-avatar/assets"
)

type toapisAvatarGroupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type toapisAvatarAssetRequest struct {
	GroupID   string `json:"group_id"`
	SourceURL string `json:"source_url"`
	Name      string `json:"name"`
}

type toapisAvatarRefreshRequest struct {
	GroupID string `json:"group_id"`
}

type toapisAvatarResponse[T any] struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type toapisAvatarGroupData struct {
	GroupID     string `json:"group_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type toapisAvatarAssetData struct {
	AssetID   string `json:"asset_id"`
	GroupID   string `json:"group_id"`
	AssetType string `json:"asset_type"`
	SourceURL string `json:"source_url"`
	Name      string `json:"name"`
	Status    string `json:"status"`
}

func ListToAPIsAvatarGroups(c *gin.Context) {
	groups, err := model.ListToAPIsAvatarGroups(c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, groups)
}

func CreateToAPIsAvatarGroup(c *gin.Context) {
	var req toapisAvatarGroupRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorMsg(c, "invalid request payload")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	if req.Name == "" {
		common.ApiErrorMsg(c, "name is required")
		return
	}

	channel, apiKey, err := getToAPIsAvatarChannel()
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	payload := map[string]any{"name": req.Name}
	if req.Description != "" {
		payload["description"] = req.Description
	}
	var upstream toapisAvatarResponse[toapisAvatarGroupData]
	err = callToAPIsAvatarAPI(channel, apiKey, http.MethodPost, toapisAvatarGroupsPath, payload, &upstream)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !upstream.Success {
		common.ApiErrorMsg(c, firstNonEmpty(upstream.Message, "failed to create avatar group"))
		return
	}
	if strings.TrimSpace(upstream.Data.GroupID) == "" {
		common.ApiErrorMsg(c, "empty group_id from ToAPIs")
		return
	}

	now := common.GetTimestamp()
	group := &model.ToAPIsAvatarGroup{
		UserID:      c.GetInt("id"),
		ChannelID:   channel.Id,
		GroupID:     strings.TrimSpace(upstream.Data.GroupID),
		Name:        firstNonEmpty(upstream.Data.Name, req.Name),
		Description: firstNonEmpty(upstream.Data.Description, req.Description),
		CreatedTime: now,
		UpdatedTime: now,
	}
	if err := model.SaveToAPIsAvatarGroup(group); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, group)
}

func ListToAPIsAvatarAssets(c *gin.Context) {
	assets, err := model.ListToAPIsAvatarAssets(c.GetInt("id"), strings.TrimSpace(c.Query("group_id")))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, assets)
}

func CreateToAPIsAvatarAsset(c *gin.Context) {
	var req toapisAvatarAssetRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorMsg(c, "invalid request payload")
		return
	}
	req.GroupID = strings.TrimSpace(req.GroupID)
	req.SourceURL = strings.TrimSpace(req.SourceURL)
	req.Name = strings.TrimSpace(req.Name)
	if req.GroupID == "" {
		common.ApiErrorMsg(c, "group_id is required")
		return
	}
	if !isHTTPURL(req.SourceURL) {
		common.ApiErrorMsg(c, "source_url must be an HTTP/HTTPS URL")
		return
	}

	if _, err := model.GetToAPIsAvatarGroup(c.GetInt("id"), req.GroupID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.ApiErrorMsg(c, "avatar group not found")
			return
		}
		common.ApiError(c, err)
		return
	}
	channel, apiKey, err := getToAPIsAvatarChannel()
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	payload := map[string]any{
		"group_id":   req.GroupID,
		"asset_type": "image",
		"source_url": req.SourceURL,
	}
	if req.Name != "" {
		payload["name"] = req.Name
	}
	var upstream toapisAvatarResponse[toapisAvatarAssetData]
	err = callToAPIsAvatarAPI(channel, apiKey, http.MethodPost, toapisAvatarAssetsPath, payload, &upstream)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !upstream.Success {
		common.ApiErrorMsg(c, firstNonEmpty(upstream.Message, "failed to create avatar asset"))
		return
	}
	if strings.TrimSpace(upstream.Data.AssetID) == "" {
		common.ApiErrorMsg(c, "empty asset_id from ToAPIs")
		return
	}
	asset := toLocalToAPIsAvatarAsset(c.GetInt("id"), channel.Id, req, upstream.Data)
	if err := model.SaveToAPIsAvatarAsset(asset); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, asset)
}

func RefreshToAPIsAvatarAssets(c *gin.Context) {
	var req toapisAvatarRefreshRequest
	_ = common.DecodeJson(c.Request.Body, &req)
	req.GroupID = strings.TrimSpace(req.GroupID)

	assets, err := model.ListRefreshableToAPIsAvatarAssets(c.GetInt("id"), req.GroupID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	channel, apiKey, err := getToAPIsAvatarChannel()
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	for _, asset := range assets {
		var upstream toapisAvatarResponse[toapisAvatarAssetData]
		path := toapisAvatarAssetsPath + "/" + url.PathEscape(asset.AssetID)
		if err := callToAPIsAvatarAPI(channel, apiKey, http.MethodGet, path, nil, &upstream); err != nil {
			continue
		}
		if !upstream.Success || strings.TrimSpace(upstream.Data.AssetID) == "" {
			continue
		}
		asset.ChannelID = channel.Id
		asset.GroupID = firstNonEmpty(upstream.Data.GroupID, asset.GroupID)
		asset.AssetType = firstNonEmpty(upstream.Data.AssetType, asset.AssetType)
		asset.SourceURL = firstNonEmpty(upstream.Data.SourceURL, asset.SourceURL)
		asset.Name = firstNonEmpty(upstream.Data.Name, asset.Name)
		asset.Status = firstNonEmpty(upstream.Data.Status, asset.Status)
		asset.UpdatedTime = common.GetTimestamp()
		_ = model.SaveToAPIsAvatarAsset(&asset)
	}
	nextAssets, err := model.ListToAPIsAvatarAssets(c.GetInt("id"), req.GroupID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nextAssets)
}

func toLocalToAPIsAvatarAsset(userID int, channelID int, req toapisAvatarAssetRequest, upstream toapisAvatarAssetData) *model.ToAPIsAvatarAsset {
	now := common.GetTimestamp()
	return &model.ToAPIsAvatarAsset{
		UserID:      userID,
		ChannelID:   channelID,
		GroupID:     firstNonEmpty(upstream.GroupID, req.GroupID),
		AssetID:     strings.TrimSpace(upstream.AssetID),
		AssetType:   firstNonEmpty(upstream.AssetType, "image"),
		Name:        firstNonEmpty(upstream.Name, req.Name),
		SourceURL:   firstNonEmpty(upstream.SourceURL, req.SourceURL),
		Status:      firstNonEmpty(upstream.Status, "processing"),
		CreatedTime: now,
		UpdatedTime: now,
	}
}

func getToAPIsAvatarChannel() (*model.Channel, string, error) {
	var channel model.Channel
	err := model.DB.Where("type = ? AND status = ?", constant.ChannelTypeToAPIs, common.ChannelStatusEnabled).
		Order("priority desc, id desc").
		First(&channel).Error
	if err != nil {
		return nil, "", fmt.Errorf("ToAPIs channel not configured")
	}
	apiKey, _, apiErr := channel.GetNextEnabledKey()
	if apiErr != nil || strings.TrimSpace(apiKey) == "" {
		return nil, "", fmt.Errorf("ToAPIs channel key not available")
	}
	return &channel, strings.TrimSpace(apiKey), nil
}

func callToAPIsAvatarAPI(channel *model.Channel, apiKey string, method string, path string, payload any, out any) error {
	var body io.Reader
	if payload != nil {
		data, err := common.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, toapisAvatarBaseURL(channel)+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("ToAPIs status %d: %s", resp.StatusCode, strings.TrimSpace(string(msg)))
	}
	return common.DecodeJson(resp.Body, out)
}

func toapisAvatarBaseURL(channel *model.Channel) string {
	base := toapisAvatarDefaultBaseURL
	if channel.BaseURL != nil && strings.TrimSpace(*channel.BaseURL) != "" {
		base = strings.TrimSpace(*channel.BaseURL)
	}
	base = strings.TrimRight(base, "/")
	for _, suffix := range []string{toapisAvatarGroupsPath, toapisAvatarAssetsPath, "/v1/videos", "/v1"} {
		base = strings.TrimSuffix(base, suffix)
	}
	return base
}

func isHTTPURL(value string) bool {
	parsed, err := url.ParseRequestURI(strings.TrimSpace(value))
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
