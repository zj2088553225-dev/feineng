package cron_ser

import (
	"github.com/robfig/cron/v3"
	"sync_data/global"
	"time"
)

func CronInit() {
	timezone, _ := time.LoadLocation("Asia/Shanghai")
	Cron := cron.New(cron.WithSeconds(), cron.WithLocation(timezone))

	// 每30分钟第5秒
	//更新jumiaapitoken
	Cron.AddFunc("0 30 */6 * * *", SyncJumiaApiToken)

	// 每2小时第10秒
	//更新jumiacentertoken
	Cron.AddFunc("0 */45 * * * *", SyncJumiaCenterToken)

	// 每1小时第20秒
	//更新jumia产品
	Cron.AddFunc("20 0 * * * *", SyncJumiaProduct)

	// 每3小时第30秒
	//更新产品库存
	Cron.AddFunc("50 0 */3 * * *", SyncJumiaProductInventory)

	// 每4小时第0秒（原240分钟）
	//更新jumia订单
	Cron.AddFunc("0 0 */4 * * *", SyncOrders)

	// 每天3点
	//更新交易数据
	Cron.AddFunc("0 0 3 * * *", SyncJumiaTransactions)

	// 每2小时更新社媒订单（76分钟不规则，暂用2小时）
	// 填充社媒订单追踪链接，状态和合伙人
	Cron.AddFunc("40 0 */2 * * *", SyncCustomizeOrder)

	// 每天2点
	//更新物流订单数据
	Cron.AddFunc("0 0 2 * * *", SyncAllAccountsWuliuData)

	// 每天4点
	//更新物流订单轨迹
	Cron.AddFunc("0 0 4 * * *", SyncAllAccountsOrdersTrajectory)

	// 每周一8点（原5点，你写错了）
	//每周一计算结算数据
	Cron.AddFunc("0 0 8 * * 1", SyncSettlementData)

	Cron.AddFunc("0 */15 * * * *", SendServiceStatus)

	// 测试任务
	Cron.AddFunc("@every 1m", func() {
		global.Log.Info("⏱️  Cron 正常运行中...")
	})

	Cron.Start()
	global.Log.Info("✅ Cron 定时任务已启动")
}

// 每两个小时     0 0 */2 * * *
// 每10秒        */10 * * * * *
