package service_api

import (
	"backend/global"
	"backend/models"
	"backend/models/res"
	"github.com/gin-gonic/gin"
)

func (ServiceApi) GetServiceStatusView(c *gin.Context) {
	var services []models.ServiceStatus
	result := global.DB.Find(&services)
	if result.Error != nil {
		global.Log.Errorf("查询服务状态失败: %v", result.Error)
		res.FailWithMessage("查询失败", c)
		return
	}
	res.OkWithData(services, c)
}
