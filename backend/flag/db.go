package flag

import (
	"backend/global"
	"backend/models"
	"backend/models/ctype"
	"github.com/sirupsen/logrus"
)

func Makemigrations() {
	err := global.DB.Set("gorm:table_options", "ENGINE=InnoDB").
		AutoMigrate(
			&models.UserModel{},
			&models.UserSellerSkuModel{},
			&models.UserProduct{},
			&models.Order{},
			&models.OrderItem{},
			&models.Transaction{},
			&models.CostSheet{},
			&models.ServiceStatus{},
			&models.UserSettlementSummary{},
			&models.UserSettlementDetail{},
			&models.UserSettlementConfig{},
			&models.CustomizeOrderGH{},
			&models.CustomizeOrderKE{},
			&models.CustomizeOrderNG{},
			&models.CesFbjOrder{},
			&models.CesFbjCargoInfo{},
			&models.CesFbjPackage{},
			&models.CesFbjCommodityInfo{},
			&models.CesFbjOrderTrajectory{},
			&models.CooperationPartner{},
		)
	if err != nil {
		logrus.Error("初始化数据库失败", err)
		return
	}
	if err := seedAdminUser(); err != nil {
		logrus.Error("初始化管理员失败", err)
		return
	}
	if err := seedServiceStatuses(); err != nil {
		logrus.Error("初始化服务状态失败", err)
		return
	}
	logrus.Info("初始化数据库成功")
}

func seedAdminUser() error {
	var count int64
	if err := global.DB.Model(&models.UserModel{}).Where("id = ? OR user_name = ?", 1, "admin").Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return global.DB.Create(&models.UserModel{
		MODEL:    models.MODEL{ID: 1},
		UserName: "admin",
		Password: "123456",
		Role:     ctype.PermissionAdmin,
	}).Error
}

func seedServiceStatuses() error {
	statuses := []models.ServiceStatus{
		{ID: 1, Service: "jumia_api_token_1", Status: "未知", Message: "初始化"},
		{ID: 2, Service: "jumia_center_token", Status: "未知", Message: "初始化"},
		{ID: 3, Service: "jumia_product_sync", Status: "未知", Message: "初始化"},
		{ID: 4, Service: "jumia_inventory_sync", Status: "未知", Message: "初始化"},
		{ID: 5, Service: "jumia_order_sync", Status: "未知", Message: "初始化"},
		{ID: 6, Service: "jumia_transaction_sync", Status: "未知", Message: "初始化"},
		{ID: 7, Service: "customize_order_sync", Status: "未知", Message: "初始化"},
		{ID: 8, Service: "wuliu_token_sync", Status: "未知", Message: "初始化"},
		{ID: 9, Service: "wuliu_order_sync", Status: "未知", Message: "初始化"},
		{ID: 10, Service: "wuliu_trajectory_sync", Status: "未知", Message: "初始化"},
		{ID: 11, Service: "settlement_sync", Status: "未知", Message: "初始化"},
		{ID: 12, Service: "jumia_api_token_2", Status: "未知", Message: "初始化"},
		{ID: 13, Service: "kilimall_logistics_sync", Status: "未知", Message: "初始化"},
	}
	for _, status := range statuses {
		var count int64
		if err := global.DB.Model(&models.ServiceStatus{}).Where("id = ? OR service = ?", status.ID, status.Service).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		if err := global.DB.Create(&status).Error; err != nil {
			return err
		}
	}
	return nil
}
