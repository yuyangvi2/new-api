package service

import (
	"strings"

	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

func GetUserUsableGroups(userGroup string) map[string]string {
	groupsCopy := setting.GetUserUsableGroupsCopy()
	if userGroup != "" {
		// userGroup 可能是逗号分隔的多个分组，如 "default,vip,svip"
		subGroups := strings.Split(userGroup, ",")
		for _, sg := range subGroups {
			sg = strings.TrimSpace(sg)
			if sg == "" {
				continue
			}
			specialSettings, b := ratio_setting.GetGroupRatioSetting().GroupSpecialUsableGroup.Get(sg)
			if b {
				// 处理特殊可用分组
				for specialGroup, desc := range specialSettings {
					if strings.HasPrefix(specialGroup, "-:") {
						// 移除分组
						groupToRemove := strings.TrimPrefix(specialGroup, "-:")
						delete(groupsCopy, groupToRemove)
					} else if strings.HasPrefix(specialGroup, "+:") {
						// 添加分组
						groupToAdd := strings.TrimPrefix(specialGroup, "+:")
						groupsCopy[groupToAdd] = desc
					} else {
						// 直接添加分组
						groupsCopy[specialGroup] = desc
					}
				}
			}
			// 如果该子分组不在 UserUsableGroups 中，也加入结果
			if _, ok := groupsCopy[sg]; !ok {
				groupsCopy[sg] = "用户分组"
			}
		}
	}
	return groupsCopy
}

func GroupInUserUsableGroups(userGroup, groupName string) bool {
	_, ok := GetUserUsableGroups(userGroup)[groupName]
	return ok
}

// GetUserAutoGroup 根据用户分组获取自动分组设置
func GetUserAutoGroup(userGroup string) []string {
	groups := GetUserUsableGroups(userGroup)
	autoGroups := make([]string, 0)
	for _, group := range setting.GetAutoGroups() {
		if _, ok := groups[group]; ok {
			autoGroups = append(autoGroups, group)
		}
	}
	return autoGroups
}

// GetUserGroupRatio 获取用户使用某个分组的倍率
// userGroup 用户分组（可能逗号分隔多个）
// group 需要获取倍率的分组
func GetUserGroupRatio(userGroup, group string) float64 {
	// userGroup 可能包含多个分组，逐个查找分组间倍率，取第一个匹配的
	for _, sg := range strings.Split(userGroup, ",") {
		sg = strings.TrimSpace(sg)
		if sg == "" {
			continue
		}
		if ratio, ok := ratio_setting.GetGroupGroupRatio(sg, group); ok {
			return ratio
		}
	}
	return ratio_setting.GetGroupRatio(group)
}
