package routers

import (
	"backend/api"
	"backend/middleware"
	"github.com/gin-gonic/gin"
)

func ServiceRouter(router *gin.RouterGroup) {
	ServiceApi := api.ApiGroupApp.ServiceApi
	//管理员查询用户列表以及他们绑定的skus
	router.GET("/service", middleware.JwtAdmin(), ServiceApi.GetServiceStatusView)
	router.POST("/service/upload", middleware.JwtAdmin(), ServiceApi.UploadCSV)
	router.GET("/service/status/:taskID", middleware.JwtAdmin(), ServiceApi.GetSyncCSVStatus)
	router.GET("/service/dashboard", middleware.JwtAdmin(), ServiceApi.AdminDashBoardView)
	router.GET("/service/my_dashboard", middleware.JwtAuth(), ServiceApi.UserDashBoardView)
	router.GET("/logistics/list", middleware.JwtAuth(), ServiceApi.GetLogisticsListView)
	router.POST("/system/kilimall-cookie", middleware.LocalOnly(), ServiceApi.UpdateKilimallCookieView)
}
