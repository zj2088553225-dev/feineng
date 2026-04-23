package routers

import (
	"backend/global"
	"github.com/gin-gonic/gin"
	"net/http"
)

func InitRouter() *gin.Engine {
	gin.SetMode(global.Config.System.Env)
	router := gin.Default()
	router.Use(CORSMiddleware())
	// swagger使用
	// 测试qq登录接口

	//如有需求在这里读取json错误码文件

	routerGroup := router.Group("/api")

	UserRouter(routerGroup)
	ServiceRouter(routerGroup)
	return router
}
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "http://localhost:5173") // ✅ 建议不要用 *，指定前端地址
		c.Header("Access-Control-Allow-Methods", "POST, GET, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, token") // ✅ 添加 token
		c.Header("Access-Control-Expose-Headers", "Content-Length")
		c.Header("Access-Control-Allow-Credentials", "true") // 如果用 cookie 或 credentials

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent) // 204
			return
		}

		c.Next()
	}
}
