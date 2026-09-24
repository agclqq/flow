package router

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"flow/application/service"
	"flow/interfaces/http/handler"
	"flow/interfaces/http/middleware"
)

// Services 聚合所有应用服务，供路由层统一引用。
type Services struct {
	Template *service.FlowTemplateService
	Version  *service.VersionService
	Instance *service.FlowInstanceService
	Notify   *service.NotifyService
	Resource *service.ExecutorResourceService
}

// New 组装 gin 引擎与全部路由。
func New(svc *Services, logger *slog.Logger) *gin.Engine {
	r := gin.New()
	r.Use(middleware.CORS(), gin.Logger(), middleware.Recovery(logger))

	ftHandler := handler.NewFlowTemplateHandler(svc.Template)
	verHandler := handler.NewVersionHandler(svc.Version)
	instHandler := handler.NewInstanceHandler(svc.Instance)
	notifyHandler := handler.NewNotifyHandler(svc.Notify)
	resourceHandler := handler.NewExecutorResourceHandler(svc.Resource)

	api := r.Group("/api/v1")
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(200, handler.Response{Code: handler.CodeOK, Data: gin.H{"status": "ok"}})
		})

		api.GET("/notify-templates", notifyHandler.List)
		api.POST("/notify-templates", notifyHandler.Create)
		api.GET("/notify-templates/:id", notifyHandler.Get)
		api.PUT("/notify-templates/:id", notifyHandler.Update)
		api.DELETE("/notify-templates/:id", notifyHandler.Delete)

		// 执行资源（节点在哪里执行）。
		api.GET("/executor-resources", resourceHandler.List)
		api.POST("/executor-resources", resourceHandler.Create)
		api.GET("/executor-resources/:id", resourceHandler.Get)
		api.PUT("/executor-resources/:id", resourceHandler.Update)
		api.DELETE("/executor-resources/:id", resourceHandler.Delete)

		templates := api.Group("/flow-templates")
		{
			templates.POST("", ftHandler.Create)
			api.GET("/cron/next", verHandler.CronNext)

		templates.GET("", ftHandler.List)
			templates.GET("/:id", ftHandler.Get)
			templates.PUT("/:id", ftHandler.Update)
			templates.POST("/:id/publish", ftHandler.Publish)
			// Webhook 触发（密钥校验见 handler）。
			templates.POST("/:id/webhook", instHandler.Webhook)
			templates.POST("/:id/clone", ftHandler.Clone)
			templates.DELETE("/:id", ftHandler.Delete)

			templates.GET("/:id/notify-bindings", ftHandler.GetNotifyBindings)
			templates.PUT("/:id/notify-bindings", ftHandler.ReplaceNotifyBindings)

			// 版本管理。
			templates.GET("/:id/versions", verHandler.ListVersions)
			templates.GET("/:id/editing-version", verHandler.GetEditingVersion)
			templates.POST("/:id/editing-version/ensure-draft", verHandler.EnsureDraftVersion)
			templates.GET("/:id/versions/:version_id", verHandler.GetVersion)
			templates.POST("/:id/versions/:version_id/publish", verHandler.PublishVersion)

			// 节点管理。
			templates.GET("/:id/versions/:version_id/nodes", verHandler.ListNodes)
			templates.POST("/:id/versions/:version_id/nodes", verHandler.AddNode)
			templates.PUT("/:id/versions/:version_id/nodes/:node_id", verHandler.UpdateNode)
			templates.DELETE("/:id/versions/:version_id/nodes/:node_id", verHandler.DeleteNode)
			templates.PUT("/:id/versions/:version_id/graph", verHandler.SaveGraph)

			// 参数管理。
			templates.GET("/:id/versions/:version_id/params", verHandler.ListParams)
			templates.POST("/:id/versions/:version_id/params", verHandler.AddParam)
			templates.PUT("/:id/versions/:version_id/params/:param_id", verHandler.UpdateParam)
			templates.DELETE("/:id/versions/:version_id/params/:param_id", verHandler.DeleteParam)

			// 触发配置。
			templates.GET("/:id/trigger", verHandler.GetTrigger)
			templates.PUT("/:id/trigger", verHandler.UpsertTrigger)

			// 流水线实例。
			templates.POST("/:id/instances", instHandler.Start)
			templates.GET("/:id/instances", instHandler.List)
			templates.GET("/:id/instances/:instance_id", instHandler.Get)
			templates.POST("/:id/instances/:instance_id/stop", instHandler.Stop)
			templates.POST("/:id/instances/:instance_id/retry", instHandler.Retry)
			templates.POST("/:id/instances/:instance_id/nodes/:node_id/approve", instHandler.Approve)
			templates.POST("/:id/instances/:instance_id/nodes/:node_id/skip", instHandler.Skip)
			templates.POST("/:id/instances/:instance_id/nodes/:node_id/reject", instHandler.Reject)
			templates.GET("/:id/instances/:instance_id/nodes/:node_id/logs", instHandler.ListNodeOperationLogs)
		}
	}

	// 未匹配路由：统一响应格式。
	r.NoRoute(func(c *gin.Context) {
		c.JSON(200, handler.Response{Code: handler.CodeNotFound, Msg: "接口不存在"})
	})

	return r
}
