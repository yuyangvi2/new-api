package controller

import (
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

// UpdateTaskBulk 薄入口，实际轮询逻辑在 service 层
func UpdateTaskBulk() {
	service.TaskPollingLoop()
}

func GetAllTask(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)

	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	// 解析其他查询参数
	queryParams := model.SyncTaskQueryParams{
		Platform:       constant.TaskPlatform(c.Query("platform")),
		TaskID:         c.Query("task_id"),
		Status:         c.Query("status"),
		Action:         c.Query("action"),
		StartTimestamp: startTimestamp,
		EndTimestamp:   endTimestamp,
		ChannelID:      c.Query("channel_id"),
	}

	items := model.TaskGetAllTasks(pageInfo.GetStartIdx(), pageInfo.GetPageSize(), queryParams)
	total := model.TaskCountAllTasks(queryParams)
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(adminTasksToDto(items))
	common.ApiSuccess(c, pageInfo)
}

func GetUserTask(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)

	userId := c.GetInt("id")

	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)

	queryParams := model.SyncTaskQueryParams{
		Platform:       constant.TaskPlatform(c.Query("platform")),
		TaskID:         c.Query("task_id"),
		Status:         c.Query("status"),
		Action:         c.Query("action"),
		StartTimestamp: startTimestamp,
		EndTimestamp:   endTimestamp,
	}

	items := model.TaskGetAllUserTask(userId, pageInfo.GetStartIdx(), pageInfo.GetPageSize(), queryParams)
	total := model.TaskCountAllUserTask(userId, queryParams)
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(userTasksToDto(items))
	common.ApiSuccess(c, pageInfo)
}

// adminTasksToDto 管理员接口：包含完整字段（含上游原始数据、渠道 ID、用户名等）。
func adminTasksToDto(tasks []*model.Task) []*dto.TaskDto {
	userIdMap := make(map[int]*model.UserBase)
	userIds := types.NewSet[int]()
	for _, task := range tasks {
		userIds.Add(task.UserId)
	}
	for _, userId := range userIds.Items() {
		cacheUser, err := model.GetUserCache(userId)
		if err == nil {
			userIdMap[userId] = cacheUser
		}
	}
	result := make([]*dto.TaskDto, len(tasks))
	for i, task := range tasks {
		if user, ok := userIdMap[task.UserId]; ok {
			task.Username = user.Username
		}
		result[i] = relay.TaskModel2Dto(task)
	}
	return result
}

// userTasksToDto 用户接口：不包含上游原始数据(Data)和渠道 ID(ChannelId)。
func userTasksToDto(tasks []*model.Task) []*dto.TaskDto {
	result := make([]*dto.TaskDto, len(tasks))
	for i, task := range tasks {
		result[i] = relay.TaskModel2UserDto(task)
	}
	return result
}
