package flag

import (
	sys_flag "flag"
	"github.com/fatih/structs"
	"sync_data/service/cron_ser"
)

type Option struct {
	DB           bool
	KilimallCost bool
	User         string // 创建用户 -u user -u admin
	ES           string // 创建索引-es create 删除索引-es delete
}

// Parse 解析命令行参数
func Parse() Option {
	//为db设置默认值，默认不进行表结构迁移
	db := sys_flag.Bool("db", false, "初始化数据库")
	kilimallCost := sys_flag.Bool("kilimall-cost", false, "同步 Kilimall 物流计费单并核算利润")
	//解析命令行参数写入注册的db中
	sys_flag.Parse()
	return Option{
		DB:           *db,
		KilimallCost: *kilimallCost,
	}
}

// 是否停止web项目
// IsWebStop 是否停止web项目
func IsWebStop(option Option) (f bool) {
	maps := structs.Map(&option)
	for _, v := range maps {
		switch val := v.(type) {
		case string:
			if val != "" {
				f = true
			}
		case bool:
			if val == true {
				f = true
			}
		}
	}
	return f
}

// 根据命令执行不同的函数
func SwitchOption(option Option) {
	if option.DB {
		Makemigrations()
		return
	}
	if option.KilimallCost {
		cron_ser.SyncKilimallCostSheets()
		return
	}

}
