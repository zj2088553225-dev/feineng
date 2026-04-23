package main

import (
	"backend/core"
	"backend/flag"
	"backend/global"
	"backend/routers"
	"fmt"
	"log"
	"time"
)

// $env:GOOS="linux"; $env:GOARCH="amd64"; go build -o main main.go

// chmod +x main
// ./main
func init() {
	// ✅ 设置全局时区为中国标准时间
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		log.Fatal("无法加载时区:", err)
	}
	time.Local = location // 关键：修改 time.Local
	// 配置信息读取
	core.InitConf()
	//日志初始化
	global.Log = core.InitLogger()
	//数据库连接
	global.DB = core.Initgorm()
	global.Log.Infoln("初始化配置完成")

}
func main() {
	//命令行参数绑定迁移表结构函数
	var option = flag.Parse()
	if flag.IsWebStop(option) {
		flag.SwitchOption(option)
		//控制迁移表结构后退出
		return
	}
	//路由初始化
	router := routers.InitRouter()
	//启动服务
	addr := global.Config.System.Addr()

	global.Log.Infoln(fmt.Sprintf("项目运行在: %s", global.Config.System.Addr()))
	router.Run(addr)
}
