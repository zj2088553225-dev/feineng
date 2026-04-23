package user_api

import (
	"backend/global"
	"backend/models"
	"backend/models/res"
	"backend/untils/jwts"
	"github.com/gin-gonic/gin"
)

type UserLoginRequest struct {
	UserName string `json:"user_name" binding:"required" msg:"请输入用户名"` // 用户名
	Password string `json:"password" binding:"required" msg:"请输入密码"`   // 密码
}

// 用户登录
func (UserApi) UserLoginView(c *gin.Context) {
	var cr UserLoginRequest
	if err := c.ShouldBindJSON(&cr); err != nil {
		res.FailWithError(err, &cr, c)
		return
	}
	var userModel models.UserModel
	err := global.DB.Take(&userModel, "user_name = ? and  password = ?", cr.UserName, cr.Password).Error
	if err != nil {
		global.Log.Error(err)
		res.FailWithMessage("登录失败", c)
		return
	}
	// 登录成功，生成token
	token, err := jwts.GenToken(jwts.JwtPayLoad{
		UserName: userModel.UserName,
		Role:     int(userModel.Role),
		UserID:   userModel.ID,
	})
	res.OkWithData(token, c)
}
