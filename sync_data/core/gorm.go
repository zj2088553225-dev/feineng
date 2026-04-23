package core

import (
	"fmt"
	"log"
	"sync_data/global"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Initgorm() *gorm.DB {

	if global.Config.MySQL.Host == "" {
		global.Log.Warn("未配置mysql，取消gorm连接数据库")
	}

	var mysqlLogger logger.Interface
	if global.Config.System.Env == "debug" {
		//开发环境显示所有的sql日志
		mysqlLogger = logger.Default.LogMode(logger.Info)
	} else {
		//只打印错误的sql日志
		mysqlLogger = logger.Default.LogMode(logger.Error)
	}
	//自定义数据库日志，便于查看某一个服务的日志
	global.MysqlLog = logger.Default.LogMode(logger.Error)

	dsn := global.Config.MySQL.Dsn()
	log.Printf("Generated DSN: %s", dsn)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: mysqlLogger,
	})
	if err != nil {
		global.Log.Error(err.Error())
		global.Log.Fatalf(fmt.Sprintf("[%s] 连接数据库失败", dsn))
	}

	sqlDB, _ := db.DB()

	// ✅ 推荐配置（根据 wait_timeout 调整）
	sqlDB.SetMaxOpenConns(50)                  // 不宜过大，50~100 足够（除非高并发）
	sqlDB.SetMaxIdleConns(25)                  // MaxOpen 的 1/2 左右，利于复用
	sqlDB.SetConnMaxLifetime(30 * time.Minute) // ✅ 严格小于 wait_timeout（建议 ≤ 30min）
	sqlDB.SetConnMaxIdleTime(15 * time.Minute) // 防止空闲太久被 DB 关闭
	global.Log.Info("连接数据库成功")
	return db
}
