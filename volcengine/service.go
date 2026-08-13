package volcengine

import (
	"context"
	stderr "errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"gorm.io/gorm"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/service"
)

const (
	ActionCreateAssetGroup = "CreateAssetGroup"
	ActionListAssetGroups  = "ListAssetGroups"
	ActionGetAssetGroup    = "GetAssetGroup"
	ActionUpdateAssetGroup = "UpdateAssetGroup"
	ActionDeleteAssetGroup = "DeleteAssetGroup"
	ActionCreateAsset      = "CreateAsset"
	ActionListAssets       = "ListAssets"
	ActionGetAsset         = "GetAsset"
	ActionUpdateAsset      = "UpdateAsset"
	ActionDeleteAsset      = "DeleteAsset"

	ActionCreateVisualValidateSession = "CreateVisualValidateSession"
	ActionGetVisualValidateResult     = "GetVisualValidateResult"
)

type (
	ResponseError struct {
		Code    string `json:"Code"`
		Message string `json:"Message"`
	}
	ResponseMetadata struct {
		RequestID string         `json:"RequestId"`
		Action    string         `json:"Action"`
		Version   string         `json:"Version"`
		Service   string         `json:"Service"`
		Region    string         `json:"Region"`
		Error     *ResponseError `json:"Error,omitempty"`
	}
	Response[Result any] struct {
		ResponseMetadata ResponseMetadata `json:"ResponseMetadata"`
		Result           Result           `json:"Result,omitempty"`
	}
)

func HandleAction(ctx *gin.Context) {
	action := ctx.Query("Action")
	userID := ctx.GetInt(string(constant.ContextKeyUserId))
	resp := Response[any]{
		ResponseMetadata: ResponseMetadata{
			RequestID: ctx.GetString(common.RequestIdKey),
			Action:    action,
			Version:   version,
			Service:   serviceName,
			Region:    region,
		},
	}
	if !arkSetting.Enabled {
		resp.ResponseMetadata.Error = &ResponseError{Code: "Forbidden", Message: "Resource is disabled"}
		ctx.JSON(http.StatusOK, resp)
		return
	}
	if userID == 0 {
		resp.ResponseMetadata.Error = &ResponseError{Code: "Unauthorized", Message: "Unauthorized"}
		ctx.JSON(http.StatusOK, resp)
		return
	}

	var rst any
	var err error
	switch action {
	case ActionCreateAssetGroup:
		rst, err = handleCreateAssetGroup(ctx, userID)
	case ActionCreateAsset:
		rst, err = handleCreateAsset(ctx, userID)
	case ActionListAssetGroups:
		rst, err = handleListAction[AssetGroup](ctx, action, userID)
	case ActionListAssets:
		rst, err = handleListAction[Asset](ctx, action, userID)
	case ActionGetAssetGroup:
		rst, err = handleOwnedResourceAction[GetAssetGroupRequest, GetAssetGroupResponse](ctx, action, userID, isAssetGroupOwnedByUser, nil)
	case ActionGetAsset:
		rst, err = handleOwnedResourceAction[GetAssetRequest, GetAssetResponse](ctx, action, userID, isAssetOwnedByUser, nil)
	case ActionUpdateAssetGroup:
		rst, err = handleOwnedResourceAction[UpdateAssetGroupRequest, UpdateAssetGroupResponse](ctx, action, userID, isAssetGroupOwnedByUser, nil)
	case ActionUpdateAsset:
		rst, err = handleOwnedResourceAction[UpdateAssetRequest, UpdateAssetResponse](ctx, action, userID, isAssetOwnedByUser, nil)
	case ActionDeleteAssetGroup:
		rst, err = handleOwnedResourceAction[DeleteAssetGroupRequest, DeleteAssetGroupResponse](ctx, action, userID, isAssetGroupOwnedByUser, deleteAssetGroupOwnership)
	case ActionDeleteAsset:
		rst, err = handleOwnedResourceAction[DeleteAssetRequest, DeleteAssetResponse](ctx, action, userID, isAssetOwnedByUser, deleteAssetOwnership)
	case ActionCreateVisualValidateSession:
		rst, err = handleCreateVisualValidateSession(ctx, action, userID)
	case ActionGetVisualValidateResult:
		rst, err = handleGetVisualValidateResult(ctx, userID)
	default:
		err = newError("InvalidAction", fmt.Sprintf("Unsupported action: %s", action))
	}
	if err != nil {
		var arkErr Error
		if stderr.As(err, &arkErr) {
			resp.ResponseMetadata.Error = &ResponseError{Code: arkErr.code, Message: arkErr.msg}
		} else {
			logger.LogError(ctx, fmt.Sprintf("ark action: %s, error: %v", action, err))
			resp.ResponseMetadata.Error = &ResponseError{Code: "InternalError", Message: "Internal error"}
		}
		ctx.JSON(http.StatusOK, resp)
		return
	}

	resp.Result = rst
	ctx.JSON(http.StatusOK, resp)
}

func decodeRequestBody(ctx *gin.Context, req any) error {
	if err := common.DecodeJson(ctx.Request.Body, req); err != nil {
		return newError("InvalidParameter", "Invalid json body")
	}
	return nil
}

func handleCreateAssetGroup(ctx *gin.Context, userID int) (*CreateAssetGroupResponse, error) {
	var req CreateAssetGroupRequest
	if err := decodeRequestBody(ctx, &req); err != nil {
		return nil, err
	}
	if req.Name == "" || utf8.RuneCountInString(req.Name) > 64 {
		return nil, newError("InvalidParameter", "Name is required and must not exceed 64 characters")
	}
	if req.Description != "" && utf8.RuneCountInString(req.Description) > 300 {
		return nil, newError("InvalidParameter", "Description must not exceed 300 characters")
	}
	return createAIGCAssetGroup(ctx, &req, userID)
}

func handleCreateAsset(ctx *gin.Context, userID int) (*CreateAssetResponse, error) {
	var req CreateAssetRequest
	if err := decodeRequestBody(ctx, &req); err != nil {
		return nil, err
	}
	if req.Name != "" && utf8.RuneCountInString(req.Name) > 64 {
		return nil, newError("InvalidParameter", "Name must not exceed 64 characters")
	}
	if req.URL == "" {
		return nil, newError("InvalidParameter", "The required parameter URL is missing.")
	}
	parsedURL, err := url.ParseRequestURI(req.URL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Host == "" {
		return nil, newError("InvalidParameter", "The specified parameter URL is invalid.")
	}
	if req.AssetType != "" && req.AssetType != "Image" && req.AssetType != "Video" && req.AssetType != "Audio" {
		return nil, newError("InvalidParameter", "The specified parameter AssetType is invalid.")
	}

	if req.GroupID != "" {
		exist, err := isAssetGroupOwnedByUser(userID, req.GroupID)
		if err != nil {
			return nil, err
		}
		if !exist {
			return nil, newError("NotFound", "The specified asset group is not found.")
		}
	} else {
		group, err := findFirstAIGCAssetGroup(userID)
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, err
			}
			resp, err := createAIGCAssetGroup(ctx, &CreateAssetGroupRequest{Name: "default"}, userID)
			if err != nil {
				return nil, err
			}
			req.GroupID = resp.ID
		} else {
			req.GroupID = group.ID
		}
	}
	return createAsset(ctx, &req, userID)
}

func createAIGCAssetGroup(ctx *gin.Context, req *CreateAssetGroupRequest, userID int) (*CreateAssetGroupResponse, error) {
	req.GroupType = GroupTypeAIGC
	resp, err := callArk[CreateAssetGroupResponse](ctx, ActionCreateAssetGroup, req)
	if err != nil {
		return nil, err
	}
	group := ArkAssetGroup{
		ID:        resp.ID,
		UserID:    userID,
		GroupType: GroupTypeAIGC,
	}
	if err = createAssetGroupOwnership(&group); err != nil {
		_, deleteErr := callArk[DeleteAssetGroupResponse](ctx, ActionDeleteAssetGroup, DeleteAssetGroupRequest{ID: resp.ID})
		if deleteErr != nil {
			logger.LogError(ctx, fmt.Sprintf("compensate ark group creation failed, id: %s, err: %v", resp.ID, deleteErr))
		}
		return nil, err
	}
	return resp, nil
}

func createAsset(ctx *gin.Context, req *CreateAssetRequest, userID int) (*CreateAssetResponse, error) {
	resp, err := callArk[CreateAssetResponse](ctx, ActionCreateAsset, req)
	if err != nil {
		return nil, err
	}
	assetID := lo.Ternary(resp.ID != "", resp.ID, resp.AssetID)
	asset := ArkAsset{
		ID:      assetID,
		UserID:  userID,
		GroupID: req.GroupID,
	}
	if err = createAssetOwnership(&asset); err != nil {
		_, deleteErr := callArk[DeleteAssetResponse](ctx, ActionDeleteAsset, DeleteAssetRequest{ID: assetID})
		if deleteErr != nil {
			logger.LogError(ctx, fmt.Sprintf("compensate ark asset creation failed, id: %s, err: %v", assetID, deleteErr))
		}
		return nil, err
	}
	return &CreateAssetResponse{ID: assetID}, nil
}

func handleListAction[Item any](ctx *gin.Context, action string, userID int) (*ListResponse[Item], error) {
	var req ListRequest
	if err := decodeRequestBody(ctx, &req); err != nil {
		return nil, err
	}
	if req.PageNumber < 1 {
		req.PageNumber = 1
	}
	if req.PageSize < 1 || req.PageSize > 20 {
		req.PageSize = 20
	}
	if order := strings.ToLower(req.SortOrder); order != "desc" && order != "asc" {
		req.SortOrder = "Desc"
	}

	groups, err := listOwnedAssetGroups(userID, req.Filter.GroupIDs, req.Filter.GroupType)
	if err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return &ListResponse[Item]{TotalCount: 0, PageSize: req.PageSize, PageNumber: req.PageNumber}, nil
	}
	req.Filter.GroupIDs = lo.Map(groups, func(resource *ArkAssetGroup, _ int) string { return resource.ID })

	return callArk[ListResponse[Item]](ctx, action, req)
}

func handleOwnedResourceAction[Req RequestWithID, Resp any](
	ctx *gin.Context,
	action string,
	userID int,
	check func(userID int, reqID string) (bool, error),
	remove func(userID int, reqID string) error,
) (*Resp, error) {
	var req Req
	if err := decodeRequestBody(ctx, &req); err != nil {
		return nil, err
	}
	if err := req.Validate(); err != nil {
		return nil, newError("InvalidParameter", err.Error())
	}

	id := req.ResourceID()
	ok, err := check(userID, id)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, newError("NotFound", "The specified resource is not found")
	}

	resp, err := callArk[Resp](ctx, action, req)
	if err != nil {
		return nil, err
	}

	if remove != nil {
		if err = remove(userID, id); err != nil {
			return nil, err
		}
	}

	return resp, nil
}

func handleCreateVisualValidateSession(ctx *gin.Context, action string, userID int) (*CreateVisualValidateSessionResponse, error) {
	var req CreateVisualValidateSessionRequest
	if err := decodeRequestBody(ctx, &req); err != nil {
		return nil, err
	}
	callbackURL := req.CallbackURL
	if callbackURL != "" {
		if err := service.ValidateSSRFProtectedFetchURL(callbackURL); err != nil {
			return nil, newError("InvalidParameter", "The specified parameter CallbackURL is invalid.")
		}
	}
	callbackBaseURL := strings.TrimRight(strings.TrimSpace(arkSetting.CallbackURL), "/")
	if callbackBaseURL == "" {
		return nil, newError("InvalidConfiguration", "ark_setting.callback_url is required.")
	}
	parsedCallbackURL, err := url.ParseRequestURI(callbackBaseURL)
	if err != nil ||
		(parsedCallbackURL.Scheme != "http" && parsedCallbackURL.Scheme != "https") ||
		parsedCallbackURL.Host == "" {
		return nil, newError("InvalidConfiguration", "ark_setting.callback_url must be an absolute http(s) URL.")
	}
	req.CallbackURL = callbackBaseURL + "/volcengine/visual_validate_callback"
	resp, err := callArk[CreateVisualValidateSessionResponse](ctx, action, req)
	if err != nil {
		return nil, err
	}
	if err = createVisualValidateSession(userID, resp.BytedToken, callbackURL); err != nil {
		return nil, err
	}
	resp.CallbackURL = callbackURL
	return resp, nil
}

func handleGetVisualValidateResult(ctx *gin.Context, userID int) (*GetVisualValidateResultResponse, error) {
	var req GetVisualValidateResultRequest
	if err := decodeRequestBody(ctx, &req); err != nil {
		return nil, err
	}
	if req.BytedToken == "" {
		return nil, newError("InvalidParameter", "The required parameter BytedToken is missing.")
	}

	session, err := findVisualValidateSession(req.BytedToken)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, newError("NotFound", "The specified token Visual Face token is not found.")
		}
		return nil, err
	}
	if session.UserID != userID {
		return nil, newError("NotFound", "The specified token Visual Face token is not found.")
	}
	if session.GroupID == "" {
		groupID, err := resolveLivenessFaceGroupForSession(ctx, session)
		if err != nil {
			return nil, err
		}
		session.GroupID = groupID
	}
	return &GetVisualValidateResultResponse{GroupID: session.GroupID}, nil
}

func HandleVisualValidateCallback(ctx *gin.Context) {
	bytedToken := ctx.Query("bytedToken")
	resultCode := ctx.Query("resultCode")
	if bytedToken == "" || resultCode == "" {
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}

	session, err := findVisualValidateSession(bytedToken)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.AbortWithStatus(http.StatusBadRequest)
			return
		}
		logger.LogError(ctx, fmt.Sprintf("find visual validate failed, err: %v", err))
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	query := ctx.Request.URL.Query()
	const resultCodeSuccess = "10000"
	if resultCode == resultCodeSuccess {
		groupID, err := resolveLivenessFaceGroupForSession(ctx, session)
		if err != nil {
			logger.LogError(ctx, fmt.Sprintf("create liveness face group failed, err: %v", err))
			query.Set("resultCode", "50500")
		} else {
			query.Set("groupId", groupID)
		}
	}

	if session.CallbackURL == "" {
		ctx.Status(http.StatusOK)
		return
	}

	resp, err := forwardVisualValidateCallback(ctx.Request.Context(), session.CallbackURL, query)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("do notify failed, err: %v", err))
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	defer func() { _ = resp.Body.Close() }()
	ctx.Status(resp.StatusCode)
	if _, err = io.Copy(ctx.Writer, resp.Body); err != nil {
		logger.LogError(ctx, fmt.Sprintf("callback copy body failed, err: %v", err))
	}
}

func resolveLivenessFaceGroupForSession(ctx context.Context, session *ArkVisualValidateSession) (string, error) {
	if session.GroupID != "" {
		return session.GroupID, nil
	}
	req := &GetVisualValidateResultRequest{BytedToken: session.ID}
	resp, err := callArk[GetVisualValidateResultResponse](ctx, ActionGetVisualValidateResult, req)
	if err != nil {
		return "", err
	}
	if resp.GroupID == "" {
		return "", newError("NotFound", "The visual validation result is not ready.")
	}
	if err = completeVisualValidateSession(session.UserID, session.ID, resp.GroupID); err != nil {
		return "", err
	}
	return resp.GroupID, nil
}

func forwardVisualValidateCallback(ctx context.Context, callbackURL string, query url.Values) (*http.Response, error) {
	if err := service.ValidateSSRFProtectedFetchURL(callbackURL); err != nil {
		return nil, errors.Wrap(err, "validate visual validate callback URL")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, callbackURL, nil)
	if err != nil {
		return nil, err
	}
	req.URL.RawQuery = query.Encode()
	return service.GetSSRFProtectedHTTPClient().Do(req)
}
