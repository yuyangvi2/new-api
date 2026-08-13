package volcengine

import (
	"errors"
	"fmt"
	"time"
	"unicode/utf8"
)

type Error struct {
	code string
	msg  string
}

func (e Error) Error() string {
	return fmt.Sprintf("code: %s, msg: %s", e.code, e.msg)
}

func newError(code string, msg string) Error {
	return Error{code: code, msg: msg}
}

type (
	CreateAssetGroupRequest struct {
		Name        string    `json:"Name"`
		Description string    `json:"Description,omitempty"`
		GroupType   GroupType `json:"GroupType,omitempty"`
		ProjectName string    `json:"ProjectName,omitempty"`
	}
	CreateAssetGroupResponse struct {
		ID string `json:"Id"`
	}

	CreateAssetRequest struct {
		GroupID     string `json:"GroupId,omitempty"`
		URL         string `json:"URL"`
		Name        string `json:"Name,omitempty"`
		AssetType   string `json:"AssetType,omitempty"`
		ProjectName string `json:"ProjectName,omitempty"`
	}
	CreateAssetResponse struct {
		ID      string `json:"Id"`
		AssetID string `json:"AssetId,omitempty"`
	}
)

type (
	ListResponse[Item any] struct {
		TotalCount int64  `json:"TotalCount"`
		PageSize   int    `json:"PageSize"`
		PageNumber int    `json:"PageNumber"`
		Items      []Item `json:"Items,omitempty"`
	}
	Filter struct {
		GroupIDs  []string  `json:"GroupIds,omitempty"`
		GroupType GroupType `json:"GroupType,omitempty"`
		Name      string    `json:"Name,omitempty"`
		Statuses  []string  `json:"Statuses,omitempty"`
	}
	ListRequest struct {
		Filter      Filter `json:"Filter,omitempty"`
		PageNumber  int    `json:"PageNumber,omitempty"`
		PageSize    int    `json:"PageSize,omitempty"`
		SortBy      string `json:"SortBy,omitempty"`
		SortOrder   string `json:"SortOrder,omitempty"`
		ProjectName string `json:"ProjectName,omitempty"`
	}

	AssetGroup struct {
		ID          string    `json:"Id"`
		Name        string    `json:"Name"`
		Description string    `json:"Description"`
		GroupType   GroupType `json:"GroupType"`
		ProjectName string    `json:"ProjectName"`
		CreateTime  time.Time `json:"CreateTime"`
		UpdateTime  time.Time `json:"UpdateTime"`
	}
	Asset struct {
		ID         string `json:"Id"`
		Name       string `json:"Name"`
		URL        string `json:"URL"`
		GroupID    string `json:"GroupId"`
		AssetType  string `json:"AssetType"`
		Status     string `json:"Status"`
		Moderation struct {
			Strategy string `json:"Strategy"`
		} `json:"Moderation,omitempty"`
		Error struct {
			Code    string `json:"Code"`
			Message string `json:"Message"`
		} `json:"Error,omitempty"`
		ProjectName       string    `json:"ProjectName,omitempty"`
		CreateTime        time.Time `json:"CreateTime,omitempty"`
		UpdateTime        time.Time `json:"UpdateTime,omitempty"`
		LastInferenceTime time.Time `json:"LastInferenceTime,omitempty"`
	}

	ListAssetGroupsRequest  = ListRequest
	ListAssetGroupsResponse = ListResponse[AssetGroup]

	ListAssetsRequest  = ListRequest
	ListAssetsResponse = ListResponse[Asset]
)

type (
	RequestWithID interface {
		Validate() error
		ResourceID() string
	}

	GetAssetGroupRequest struct {
		ID          string `json:"Id"`
		ProjectName string `json:"ProjectName,omitempty"`
	}
	GetAssetGroupResponse = AssetGroup

	UpdateAssetGroupRequest struct {
		ID          string `json:"Id"`
		Name        string `json:"Name,omitempty"`
		Description string `json:"Description,omitempty"`
		ProjectName string `json:"ProjectName,omitempty"`
	}
	UpdateAssetGroupResponse = CreateAssetGroupResponse

	DeleteAssetGroupRequest  = GetAssetGroupRequest
	DeleteAssetGroupResponse = CreateAssetGroupResponse

	GetAssetRequest  = GetAssetGroupRequest
	GetAssetResponse = Asset

	UpdateAssetRequest struct {
		ID          string `json:"Id"`
		Name        string `json:"Name,omitempty"`
		ProjectName string `json:"ProjectName,omitempty"`
	}
	UpdateAssetResponse = CreateAssetResponse

	DeleteAssetRequest  = GetAssetRequest
	DeleteAssetResponse = CreateAssetGroupResponse
)

func (req GetAssetGroupRequest) Validate() error {
	if req.ID == "" {
		return errors.New("The required parameter Id is missing.")
	}
	return nil
}

func (req GetAssetGroupRequest) ResourceID() string { return req.ID }

func (req UpdateAssetGroupRequest) Validate() error {
	if req.ID == "" {
		return errors.New("The required parameter Id is missing.")
	}
	if req.Name != "" && utf8.RuneCountInString(req.Name) > 64 {
		return errors.New("Name must not exceed 64 characters")
	}
	if req.Description != "" && utf8.RuneCountInString(req.Description) > 300 {
		return errors.New("Description must not exceed 300 characters")
	}
	return nil
}

func (req UpdateAssetGroupRequest) ResourceID() string { return req.ID }

func (req UpdateAssetRequest) Validate() error {
	if req.ID == "" {
		return errors.New("The required parameter Id is missing.")
	}
	if utf8.RuneCountInString(req.Name) > 64 {
		return errors.New("Name must not exceed 64 characters")
	}
	return nil
}

func (req UpdateAssetRequest) ResourceID() string { return req.ID }

type (
	CreateVisualValidateSessionRequest struct {
		CallbackURL string `json:"CallbackURL,omitempty"`
		ProjectName string `json:"ProjectName,omitempty"`
	}
	CreateVisualValidateSessionResponse struct {
		BytedToken  string `json:"BytedToken"`
		H5Link      string `json:"H5Link"`
		CallbackURL string `json:"CallbackURL"`
	}

	GetVisualValidateResultRequest struct {
		BytedToken  string `json:"BytedToken"`
		ProjectName string `json:"ProjectName,omitempty"`
	}
	GetVisualValidateResultResponse struct {
		GroupID string `json:"GroupId"`
	}
)
