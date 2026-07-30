package router

import (
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"

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
}
