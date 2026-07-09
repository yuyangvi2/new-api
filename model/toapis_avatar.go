package model

import (
	"errors"

	"gorm.io/gorm"
)

type ToAPIsAvatarGroup struct {
	Id          int    `json:"id"`
	UserID      int    `json:"user_id" gorm:"index;index:idx_toapis_avatar_group_user_group,unique"`
	ChannelID   int    `json:"channel_id"`
	GroupID     string `json:"group_id" gorm:"index;index:idx_toapis_avatar_group_user_group,unique"`
	Name        string `json:"name"`
	Description string `json:"description" gorm:"type:text"`
	CreatedTime int64  `json:"created_time" gorm:"bigint"`
	UpdatedTime int64  `json:"updated_time" gorm:"bigint"`
}

type ToAPIsAvatarAsset struct {
	Id          int    `json:"id"`
	UserID      int    `json:"user_id" gorm:"index;index:idx_toapis_avatar_asset_user_group"`
	ChannelID   int    `json:"channel_id"`
	GroupID     string `json:"group_id" gorm:"index:idx_toapis_avatar_asset_user_group"`
	AssetID     string `json:"asset_id" gorm:"index;uniqueIndex"`
	AssetType   string `json:"asset_type" gorm:"index"`
	Name        string `json:"name"`
	SourceURL   string `json:"source_url" gorm:"type:text"`
	Status      string `json:"status" gorm:"index"`
	CreatedTime int64  `json:"created_time" gorm:"bigint"`
	UpdatedTime int64  `json:"updated_time" gorm:"bigint"`
}

func ListToAPIsAvatarGroups(userID int) ([]ToAPIsAvatarGroup, error) {
	var groups []ToAPIsAvatarGroup
	err := DB.Where("user_id = ?", userID).Order("id desc").Find(&groups).Error
	return groups, err
}

func SaveToAPIsAvatarGroup(group *ToAPIsAvatarGroup) error {
	var existing ToAPIsAvatarGroup
	err := DB.Where("user_id = ? AND group_id = ?", group.UserID, group.GroupID).First(&existing).Error
	if err == nil {
		group.Id = existing.Id
		if group.CreatedTime == 0 {
			group.CreatedTime = existing.CreatedTime
		}
		return DB.Model(&existing).Updates(group).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return DB.Create(group).Error
}

func GetToAPIsAvatarGroup(userID int, groupID string) (*ToAPIsAvatarGroup, error) {
	var group ToAPIsAvatarGroup
	err := DB.Where("user_id = ? AND group_id = ?", userID, groupID).First(&group).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func ListToAPIsAvatarAssets(userID int, groupID string) ([]ToAPIsAvatarAsset, error) {
	query := DB.Where("user_id = ?", userID)
	if groupID != "" {
		query = query.Where("group_id = ?", groupID)
	}
	var assets []ToAPIsAvatarAsset
	err := query.Order("id desc").Find(&assets).Error
	return assets, err
}

func ListRefreshableToAPIsAvatarAssets(userID int, groupID string) ([]ToAPIsAvatarAsset, error) {
	query := DB.Where("user_id = ? AND status IN ?", userID, []string{"processing", "failed"})
	if groupID != "" {
		query = query.Where("group_id = ?", groupID)
	}
	var assets []ToAPIsAvatarAsset
	err := query.Order("id desc").Find(&assets).Error
	return assets, err
}

func SaveToAPIsAvatarAsset(asset *ToAPIsAvatarAsset) error {
	var existing ToAPIsAvatarAsset
	err := DB.Where("asset_id = ?", asset.AssetID).First(&existing).Error
	if err == nil {
		asset.Id = existing.Id
		if asset.CreatedTime == 0 {
			asset.CreatedTime = existing.CreatedTime
		}
		return DB.Model(&existing).Updates(asset).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return DB.Create(asset).Error
}
