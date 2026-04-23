package routers

import (
	"backend/api"
	"backend/middleware"

	"github.com/gin-gonic/gin"
)

func UserRouter(router *gin.RouterGroup) {
	UserApi := api.ApiGroupApp.UserApi
	//用户登录接口
	router.POST("/user/login", UserApi.UserLoginView)
	//管理员创建用户接口
	router.POST("/user/create_user", middleware.JwtAdmin(), UserApi.CreateUserView)
	//管理员编辑用户数据接口
	router.PUT("/user/create_user", middleware.JwtAdmin(), UserApi.EditUserView)
	//管理员删除用户数据接口
	router.POST("/user/delete_user", middleware.JwtAdmin(), UserApi.DeleteUserView)

	//管理员查询用户列表以及他们绑定的skus
	router.GET("/user/user_list", middleware.JwtAdmin(), UserApi.GetUserListView)
	//管理员查询用户名列表
	router.GET("/user/user_name_list", middleware.JwtAdmin(), UserApi.GetUserNameListView)

	//管理员查看所有的用户的产品数据
	router.GET("/user/user_product", middleware.JwtAdmin(), UserApi.GetAdminProductView)
	//管理员修改用户的的产品数据
	router.PUT("/user/user_product", middleware.JwtAdmin(), UserApi.AdminEditUserProductView)
	//管理员修改产品的合伙人
	router.PUT("/user/product_user", middleware.JwtAdmin(), UserApi.AdminUpdateProductUserView)
	//管理员批量修改产品的合伙人
	router.PUT("/user/product_users", middleware.JwtAdmin(), UserApi.AdminBatchUpdateProductUserView)
	//管理员查看所有订单
	router.GET("/user/orders", middleware.JwtAdmin(), UserApi.GetAllOrderListView)
	//管理员查看所有交易记录
	router.GET("/user/transactions", middleware.JwtAdmin(), UserApi.AdminTransactionView)
	//管理员查看所有社媒订单记录
	router.GET("/user/customize_order", middleware.JwtAdmin(), UserApi.AdminCustomizeOrderView)
	//管理员查看所有物流订单记录
	router.GET("/user/wuliu", middleware.JwtAdmin(), UserApi.AdminWuliuView)

	//管理员查看所有结算数据
	router.GET("/user/settlement", middleware.JwtAdmin(), UserApi.AdminSettlementView)
	//管理员更新结算状态
	router.PUT("/user/settlement", middleware.JwtAdmin(), UserApi.AdminEditSettlementView)
	//管理员查看结算配置
	router.GET("/user/settlement_config", middleware.JwtAdmin(), UserApi.AdminSettlementConfig)
	//管理员增加配置
	router.POST("/user/settlement_config", middleware.JwtAdmin(), UserApi.AdminAddSettlementConfig)
	//管理员更新配置
	router.PUT("/user/settlement_config", middleware.JwtAdmin(), UserApi.AdminUpdateSettlementConfig)
	//管理员删除配置
	router.DELETE("/user/settlement_config/:id", middleware.JwtAdmin(), UserApi.AdminDeleteSettlementConfig)

	//管理员手动根据配置重新计算结算数据
	router.GET("/user/settlement_config/:id", middleware.JwtAdmin(), UserApi.TriggerSettlementCalculation)
	//管理员获取周期内国家数据总和
	router.GET("/user/settlement_total", middleware.JwtAdmin(), UserApi.GetUserSettlementTotal)

	//管理员获取合营合伙人信息
	router.GET("/user/cooperation_partner", middleware.JwtAdmin(), UserApi.GetCooperationPartners)
	router.POST("/user/cooperation_partner", middleware.JwtAdmin(), UserApi.AddCooperationPartner)
	router.PUT("/user/cooperation_partner", middleware.JwtAdmin(), UserApi.EditCooperationPartner)
	router.DELETE("/user/cooperation_partner", middleware.JwtAdmin(), UserApi.DeleteCooperationPartner)

	//用户获取他们自己的sku列表
	router.GET("/user/my_sku", middleware.JwtAuth(), UserApi.GetMyskuView)
	//用户获取他们自己的产品列表
	router.GET("/user/my_product", middleware.JwtAuth(), UserApi.GetUserProductView)
	//用户修改他们自己的产品数据
	router.PUT("/user/my_product", middleware.JwtAuth(), UserApi.EditUserProductView)
	//用户查看他们的订单
	router.GET("/user/my_order", middleware.JwtAuth(), UserApi.GetOrderListView)
	//用户查看自己的交易记录
	router.GET("/user/my_transactions", middleware.JwtAuth(), UserApi.UserTransactionView)
	//用户查看自己的交易记录
	router.POST("/user/userinfo", middleware.JwtAuth(), UserApi.UserUpdatePasswordView)
	//用户查看自己的社媒订单记录
	router.GET("/user/my_customize_order", middleware.JwtAuth(), UserApi.UserCustomizeOrderView)
	//用户查看自己的物流订单记录
	router.GET("/user/my_wuliu", middleware.JwtAuth(), UserApi.UserWuliuView)
	//用户查看自己的结算数据
	router.GET("/user/my_settlement", middleware.JwtAuth(), UserApi.GetUserSettlementView)

	//用户下载高清无码tiktok视频
	//router.POST("/user/tiktok", UserApi.TiktokDownloadView)

}
