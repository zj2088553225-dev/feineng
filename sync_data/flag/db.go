package flag

import (
	"github.com/sirupsen/logrus"
	"sync_data/global"
	"sync_data/models"
)

func Makemigrations() {
	var err error
	//生成四张表的表结构
	err = global.DB.Set("gorm:table_options", "ENGINE=InnoDB").
		AutoMigrate(
			//&models.UserModel{},
			//&models.UserSellerSkuModel{},
			//&models.UserProduct{},
			//&models.Order{},
			//&models.OrderItem{},
			//&models.Transaction{},
			//&models.ServiceStatus{},
			&models.CesFbjOrder{},
			//&models.CesFbjPackage{},
			//&models.CesFbjCargoInfo{},
			//&models.CesFbjCommodityInfo{},
			//&models.CesFbjOrderTrajectory{},
			//&models.UserSettlementSummary{},
			//&models.UserSettlementDetail{},
			//&models.UserSettlementConfig{},
		)
	if err != nil {
		logrus.Error("初始化数据库失败", err)
		return
	}
	logrus.Info("初始化数据库成功")
}
