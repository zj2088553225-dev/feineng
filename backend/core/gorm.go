package core

import (
	"backend/global"
	"fmt"
	"log"
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
	global.MysqlLog = logger.Default.LogMode(logger.Info)

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

	sqlDB.SetMaxOpenConns(100)              //最多可容纳
	sqlDB.SetMaxIdleConns(20)               //最大空闲连接数
	sqlDB.SetConnMaxLifetime(time.Hour * 4) //连接最大复用时间，不能超过mysql的wait_timeout
	global.Log.Info("连接数据库成功")
	return db
}
