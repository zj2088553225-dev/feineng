package main

import (
	"log"
	"sync_data/core"
	"sync_data/flag"
	"sync_data/global"
	"sync_data/service/cron_ser"
	"time"
)

// set
func main() {
	// ✅ 设置全局时区为中国标准时间
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		log.Fatal("无法加载时区:", err)
	}
	time.Local = location // 关键：修改 time.Local
	// 初始化配置、日志、数据库等
	core.InitConf()
	global.Log = core.InitLogger()
	global.DB = core.Initgorm()
	global.Log.Infoln("初始化配置完成")

	//命令行参数绑定迁移表结构函数
	var option = flag.Parse()
	if flag.IsWebStop(option) {
		flag.SwitchOption(option)
		//控制迁移表结构后退出
		return
	}
	// ✅ 只启动一次定时任务
	//cron_ser.CronInit()
	//cron_ser.SyncOrders()
	//cron_ser.SyncJumiaApiToken()
	//cron_ser.SyncJumiaCenterToken()
	//cron_ser.SyncAllAccountsWuliuData()
	//cron_ser.SyncAllAccountsOrdersTrajectory()
	//err = cron_ser.SyncONEOrderItems(global.Config.Jumia.AccessToken, []string{"365876217", "345716217", "327414417"})
	cron_ser.SyncUserProductInventoryForone()
	//if err != nil {
	//	global.Log.Error(err.Error())
	//}

	// ✅ 阻塞，防止程序退出
	select {}
}
