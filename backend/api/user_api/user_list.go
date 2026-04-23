package user_api

import (
	"backend/global"
	"backend/models"
	"backend/models/res"
	"backend/untils/jwts"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 管理员查看用户以及他们绑定的sku
func (UserApi) GetUserNameListView(c *gin.Context) {

	// 使用 Preload 加载关联的 SellerSkus
	var users []map[string]interface{}
	err := global.DB.
		Select([]string{"id", "user_name"}).
		Model(&models.UserModel{}).
		Find(&users).Error
	if err != nil {
		res.FailWithMessage("查询失败: "+err.Error(), c)
		return
	}

	// 返回成功响应
	res.OkWithData(users, c)
}

// 管理员查看用户以及他们绑定的sku
func (UserApi) GetUserListView(c *gin.Context) {
	var users []models.UserModel

	// 使用 Preload 加载关联的 SellerSkus
	err := global.DB.Preload("SellerSkus").Find(&users).Error
	if err != nil {
		res.FailWithMessage("查询失败: "+err.Error(), c)
		return
	}

	// 返回成功响应
	res.OkWithData(users, c)
}

// 用户查看他们自己绑定的sku
func (UserApi) GetMyskuView(c *gin.Context) {
	// 获取用户 ID
	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.CustomClaims)

	var user models.UserModel

	err := global.DB.Where("id = ?", claims.UserID).Preload("SellerSkus").First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			res.FailWithMessage("用户不存在", c)
		} else {
			// 处理其他数据库错误
			res.FailWithMessage("查询用户信息失败: "+err.Error(), c)
		}
		return
	}

	res.OkWithData(gin.H{
		"user_name": user.UserName,
		"pass_word": user.Password,
		"skus":      user.SellerSkus, // 假设关联的字段是 SellerSkus
	}, c)

	// 或者，如果只需要返回 SKU 列表：
	// res.OkWithData(user.SellerSkus, c)
}
