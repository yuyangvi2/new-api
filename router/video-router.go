package router

import (
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/volcengine"

	"github.com/gin-gonic/gin"
)

func SetVideoRouter(router *gin.Engine) {
	// Video proxy: accepts either session auth (dashboard) or token auth (API clients)
	videoProxyRouter := router.Group("/v1")
	videoProxyRouter.Use(middleware.RouteTag("relay"))
	videoProxyRouter.Use(middleware.TokenOrUserAuth())
	{
		videoProxyRouter.GET("/videos/:task_id/content", controller.VideoProxy)
	}

	videoV1Router := router.Group("/v1")
	videoV1Router.Use(middleware.RouteTag("relay"))
	videoV1Router.Use(middleware.TokenAuth(), middleware.Distribute())
	{
		videoV1Router.POST("/video/generations", controller.RelayTask)
		videoV1Router.GET("/video/generations/:task_id", controller.RelayTaskFetch)
		videoV1Router.POST("/videos/generations", controller.RelayTask)
		videoV1Router.POST("/videos/:video_id/remix", controller.RelayTask)
		videoV1Router.GET("/seedance-m/video-tasks", controller.ListSeedanceMVideoTasks)
		videoV1Router.GET("/seedance-m/video-tasks/:upstream_task_id", controller.GetSeedanceMVideoTask)
		videoV1Router.DELETE("/seedance-m/video-tasks/:upstream_task_id", controller.DeleteSeedanceMVideoTask)
		videoV1Router.POST("/seedance-m/asset-groups", controller.CreateSeedanceMAssetGroup)
		videoV1Router.POST("/seedance-m/asset-groups/query", controller.QuerySeedanceMAssetGroups)
		videoV1Router.GET("/seedance-m/asset-groups/:group_id", controller.GetSeedanceMAssetGroup)
		videoV1Router.PUT("/seedance-m/asset-groups/:group_id", controller.UpdateSeedanceMAssetGroup)
		videoV1Router.DELETE("/seedance-m/asset-groups/:group_id", controller.DeleteSeedanceMAssetGroup)
		videoV1Router.POST("/seedance-m/assets", controller.CreateSeedanceMAsset)
		videoV1Router.POST("/seedance-m/assets/query", controller.QuerySeedanceMAssets)
		videoV1Router.GET("/seedance-m/assets/:asset_id", controller.GetSeedanceMAsset)
		videoV1Router.PUT("/seedance-m/assets/:asset_id", controller.UpdateSeedanceMAsset)
		videoV1Router.DELETE("/seedance-m/assets/:asset_id", controller.DeleteSeedanceMAsset)
		videoV1Router.POST("/seedance-m/real-person-auth/sessions", controller.CreateSeedanceMRealPersonAuthSession)
		videoV1Router.POST("/seedance-m/real-person-auth/asset-group/by-byted-token", controller.CreateSeedanceMRealPersonAssetGroupByBytedToken)
	}
	// openai compatible API video routes
	// docs: https://platform.openai.com/docs/api-reference/videos/create
	{
		videoV1Router.POST("/videos", controller.RelayTask)
		videoV1Router.GET("/videos/:task_id", controller.RelayTaskFetch)
	}

	klingV1Router := router.Group("/kling/v1")
	klingV1Router.Use(middleware.RouteTag("relay"))
	klingV1Router.Use(middleware.KlingRequestConvert(), middleware.TokenAuth(), middleware.Distribute())
	{
		klingV1Router.POST("/videos/text2video", controller.RelayTask)
		klingV1Router.POST("/videos/image2video", controller.RelayTask)
		klingV1Router.GET("/videos/text2video/:task_id", controller.RelayTaskFetch)
		klingV1Router.GET("/videos/image2video/:task_id", controller.RelayTaskFetch)
	}

	// Jimeng official API routes - direct mapping to official API format
	jimengOfficialGroup := router.Group("jimeng")
	jimengOfficialGroup.Use(middleware.RouteTag("relay"))
	jimengOfficialGroup.Use(middleware.JimengRequestConvert(), middleware.TokenAuth(), middleware.Distribute())
	{
		// Maps to: /?Action=CVSync2AsyncSubmitTask&Version=2022-08-31 and /?Action=CVSync2AsyncGetResult&Version=2022-08-31
		jimengOfficialGroup.POST("/", controller.RelayTask)
	}

	seedanceGroup := router.Group("/volcengine")
	{
		seedanceGroup.POST("/", middleware.TokenAuth(), volcengine.HandleAction)
		seedanceGroup.GET("/visual_validate_callback", volcengine.HandleVisualValidateCallback)
	}
	registerSeedanceAPIV3TaskRoutes(seedanceGroup.Group("/api/v3"))
	registerSeedanceOfficialAssetRoutes(router.Group("/v1/seedance-official"))

	// Drop-in compatibility for clients whose Seedance Official adaptor appends
	// /api/v3/contents/generations/tasks directly to the configured base URL.
	registerSeedanceAPIV3TaskRoutes(router.Group("/api/v3"))
	registerSeedanceMOfficialTaskRoutes(router.Group(""))
	registerHailuoOfficialTaskRoutes(router.Group("/v1"))
	registerAliOfficialTaskRoutes(router.Group("/api/v1"))
}

func registerSeedanceAPIV3TaskRoutes(seedanceAPIV3Group *gin.RouterGroup) {
	seedanceAPIV3Group.Use(middleware.RouteTag("relay"))
	seedanceAPIV3Group.Use(middleware.TokenAuth(), middleware.Distribute())
	{
		seedanceAPIV3Group.POST("/contents/generations/tasks", controller.RelayTask)
		seedanceAPIV3Group.GET("/contents/generations/tasks/:task_id", controller.RelayTaskFetch)
	}
}

func registerSeedanceMOfficialTaskRoutes(seedanceMGroup *gin.RouterGroup) {
	seedanceMGroup.Use(middleware.RouteTag("relay"))
	seedanceMGroup.Use(middleware.TokenAuth(), middleware.RequireChannelType(constant.ChannelTypeSeedanceM), middleware.Distribute())
	{
		seedanceMGroup.POST("/contents/generations/tasks", controller.RelayTask)
		seedanceMGroup.GET("/contents/generations/tasks/:task_id", controller.RelayTaskFetch)
	}
}

func registerHailuoOfficialTaskRoutes(hailuoGroup *gin.RouterGroup) {
	hailuoGroup.Use(middleware.RouteTag("relay"))
	hailuoGroup.Use(middleware.TokenAuth(), middleware.RequireChannelType(constant.ChannelTypeMiniMax), middleware.Distribute())
	{
		hailuoGroup.POST("/video_generation", controller.RelayTask)
		hailuoGroup.GET("/query/video_generation", controller.RelayTaskFetch)
	}
}

func registerAliOfficialTaskRoutes(aliGroup *gin.RouterGroup) {
	aliGroup.Use(middleware.RouteTag("relay"))
	aliGroup.Use(middleware.TokenAuth(), middleware.RequireChannelType(constant.ChannelTypeAli), middleware.Distribute())
	{
		aliGroup.POST("/services/aigc/video-generation/video-synthesis", controller.RelayTask)
		aliGroup.GET("/tasks/:task_id", controller.RelayTaskFetch)
	}
}

func registerSeedanceOfficialAssetRoutes(seedanceOfficialGroup *gin.RouterGroup) {
	seedanceOfficialGroup.Use(middleware.RouteTag("relay"))
	seedanceOfficialGroup.Use(middleware.TokenAuth())
	{
		seedanceOfficialGroup.POST("/asset-groups", volcengine.HandleRESTAction(volcengine.ActionCreateAssetGroup, "", nil))
		seedanceOfficialGroup.POST("/asset-groups/query", volcengine.HandleRESTAction(volcengine.ActionListAssetGroups, "", nil))
		seedanceOfficialGroup.GET("/asset-groups/:group_id", volcengine.HandleRESTAction(volcengine.ActionGetAssetGroup, "group_id", nil))
		seedanceOfficialGroup.PUT("/asset-groups/:group_id", volcengine.HandleRESTAction(volcengine.ActionUpdateAssetGroup, "group_id", nil))
		seedanceOfficialGroup.DELETE("/asset-groups/:group_id", volcengine.HandleRESTAction(volcengine.ActionDeleteAssetGroup, "group_id", nil))
		seedanceOfficialGroup.POST("/assets", volcengine.HandleRESTAction(volcengine.ActionCreateAsset, "", nil))
		seedanceOfficialGroup.POST("/assets/query", volcengine.HandleRESTAction(volcengine.ActionListAssets, "", nil))
		seedanceOfficialGroup.GET("/assets/:asset_id", volcengine.HandleRESTAction(volcengine.ActionGetAsset, "asset_id", nil))
		seedanceOfficialGroup.PUT("/assets/:asset_id", volcengine.HandleRESTAction(volcengine.ActionUpdateAsset, "asset_id", nil))
		seedanceOfficialGroup.DELETE("/assets/:asset_id", volcengine.HandleRESTAction(volcengine.ActionDeleteAsset, "asset_id", nil))
		seedanceOfficialGroup.POST("/real-person-auth/sessions", volcengine.HandleRESTAction(volcengine.ActionCreateVisualValidateSession, "", nil))
		seedanceOfficialGroup.POST("/real-person-auth/asset-group/by-byted-token", volcengine.HandleRESTAction(
			volcengine.ActionGetVisualValidateResult,
			"",
			map[string]string{"byted_token": "BytedToken", "bytedToken": "BytedToken"},
		))
	}
}
