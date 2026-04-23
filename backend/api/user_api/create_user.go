package user_api

import (
	"backend/global"
	"backend/models"
	"backend/models/ctype"
	"backend/models/res"
	"backend/untils/jwts"
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserCreateRequest struct {
	UserName string     `json:"user_name" binding:"required" msg:"请输入用户名"` // 用户名
	Password string     `json:"password" binding:"required" msg:"请输入密码"`   // 密码
	Role     ctype.Role `json:"role" binding:"required" msg:"请选择权限"`       // 权限  1 管理员  2 普通用户  3 游客
}

// 管理员创建用户
func (UserApi) CreateUserView(c *gin.Context) {
	var cr UserCreateRequest
	if err := c.ShouldBindJSON(&cr); err != nil {
		res.FailWithError(err, &cr, c)
		return
	}
	var user models.UserModel

	// 查询用户名是否已存在
	err := global.DB.Where("user_name = ?", cr.UserName).First(&user).Error

	if err == nil {
		// 用户名已存在
		res.FailWithMessage("用户名已存在，请更换用户名", c)
		return
	}

	if err != gorm.ErrRecordNotFound {
		// 数据库层面出错（比如连接失败），不是“记录不存在”
		res.FailWithMessage("数据库查询失败: "+err.Error(), c)
		return
	}
	err = global.DB.Create(&models.UserModel{
		UserName: cr.UserName,
		Password: cr.Password,
		Role:     cr.Role,
	}).Error
	if err != nil {
		global.Log.Error(err)
		res.FailWithMessage("创建用户失败", c)
		return
	}
	res.OkWithMessage("创建用户成功", c)
}

type UserEditRequest struct {
	UserID   int        `json:"user_id" binding:"required" msg:"请输入用户id"`  // 用户名
	UserName string     `json:"user_name" binding:"required" msg:"请输入用户名"` // 用户名
	Password string     `json:"password" binding:"required" msg:"请输入密码"`   // 密码
	Role     ctype.Role `json:"role" binding:"required" msg:"请选择权限"`       // 权限  1 管理员  2 普通用户  3 游客
}

// 管理员创建用户
// 管理员创建/编辑用户
func (UserApi) EditUserView(c *gin.Context) {
	var req UserEditRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		res.FailWithError(err, &req, c)
		return
	}

	// 1. 检查用户名是否被其他用户占用
	var existingUser models.UserModel
	err := global.DB.Where("user_name = ? AND id != ?", req.UserName, req.UserID).First(&existingUser).Error
	if err == nil {
		global.Log.Error("用户名已被其他用户使用")
		res.FailWithMessage("用户名已存在，请更换用户名", c)
		return
	}
	if err != gorm.ErrRecordNotFound {
		global.Log.Error("查询用户名是否冲突时数据库错误", err)
		res.FailWithMessage("服务器内部错误：数据库查询失败", c)
		return
	}

	// 2. 构建更新字段
	updateData := make(map[string]interface{})
	updateData["user_name"] = req.UserName
	updateData["role"] = req.Role
	updateData["password"] = req.Password // 假设有加密函数

	// 3. 使用事务更新用户 + 同步更新 UserProduct 表
	err = global.DB.Transaction(func(tx *gorm.DB) error {
		var user models.UserModel

		// 3.1 查询用户是否存在
		if err := tx.First(&user, req.UserID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return gorm.ErrRecordNotFound
			}
			return err
		}
		// 3.3 ✅ 同步更新 UserProduct 表中所有该用户的 user_name
		// 只有当 user_name 发生变化时才更新
		global.Log.Info(fmt.Sprintf("用户旧用户名: %s, 新用户名: %s", user.UserName, req.UserName))
		if user.UserName != req.UserName {
			if err := tx.Model(&models.UserProduct{}).
				Where("user_id = ?", req.UserID).
				Update("user_name", req.UserName).Error; err != nil {
				return err
			}
			global.Log.Info("✅ 检测到用户名变化，准备更新 UserProduct 表")
			// 执行更新...
		} else {
			global.Log.Info("❌ 用户名未变化，跳过更新 UserProduct")
		}

		// 3.2 更新用户表
		if err := tx.Model(&user).Updates(updateData).Error; err != nil {
			return err
		}

		return nil
	})

	if err == gorm.ErrRecordNotFound {
		res.FailWithMessage("用户不存在，无法更新", c)
		return
	}
	if err != nil {
		global.Log.Error("更新用户失败", err)
		res.FailWithMessage("更新用户失败，请稍后重试", c)
		return
	}

	global.Log.Info("用户信息及关联数据更新成功")
	res.OkWithMessage("更新用户数据成功", c)
}

// UserDeleteRequest 删除用户请求参数
type UserDeleteRequest struct {
	UserID uint `json:"user_id" binding:"required" msg:"请输入用户ID"`
}

// DeleteUserView 删除用户及其所有绑定数据（归还 SellerSku 给管理员）
// DeleteUserView 删除用户及其所有绑定数据（归还 SellerSku 给管理员）
func (UserApi) DeleteUserView(c *gin.Context) {
	var req UserDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		res.FailWithError(err, &req, c)
		return
	}

	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.CustomClaims)

	// ❗️防止删除自己
	if claims.UserID == req.UserID {
		res.FailWithMessage("无法删除当前登录账户", c)
		return
	}

	// ❗️禁止删除管理员（user_id = 1）
	if req.UserID == 1 {
		res.FailWithMessage("管理员账户不可删除", c)
		return
	}

	// 查询目标用户
	var user models.UserModel
	err := global.DB.First(&user, req.UserID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			res.FailWithMessage("用户不存在", c)
			return
		}
		global.Log.Error("查询用户失败: ", err)
		res.FailWithMessage("服务器内部错误", c)
		return
	}

	// 🔥 使用事务：归还 SellerSku + 更新产品归属 + 删除用户
	err = global.DB.Transaction(func(tx *gorm.DB) error {
		// 1. 查询该用户所有 SellerSku 绑定（可选，仅用于日志）
		var skuBindings []models.UserSellerSkuModel
		if err := tx.Where("user_id = ?", req.UserID).Find(&skuBindings).Error; err != nil {
			return fmt.Errorf("查询用户绑定的 SellerSku 失败: %w", err)
		}

		// 2. 归还所有 SellerSku 给管理员（user_id = 1）
		if len(skuBindings) > 0 {
			result := tx.Model(&models.UserSellerSkuModel{}).
				Where("user_id = ?", req.UserID).
				Update("user_id", 1)

			if result.Error != nil {
				return fmt.Errorf("归还 SellerSku 给管理员失败: %w", result.Error)
			}
			global.Log.Infof("用户 %d 的 %d 个 SellerSku 已归还给管理员", req.UserID, result.RowsAffected)
		}

		// 3. ✅ 更新 user_product 表：将该用户的产品归属改为管理员
		productUpdateResult := tx.Model(&models.UserProduct{}).
			Where("user_id = ?", req.UserID).
			Updates(map[string]interface{}{
				"user_id":   uint(1),
				"user_name": "admin", // 你可以改为动态获取管理员姓名
			})

		if productUpdateResult.Error != nil {
			return fmt.Errorf("更新 user_product 表失败: %w", productUpdateResult.Error)
		}
		global.Log.Infof("已将用户 %d 的 %d 个产品归属更新为管理员", req.UserID, productUpdateResult.RowsAffected)

		// 4. 删除用户本身（软删除）
		if err := tx.Delete(&user).Error; err != nil {
			return fmt.Errorf("删除用户记录失败: %w", err)
		}

		return nil
	})

	if err != nil {
		global.Log.WithField("user_id", req.UserID).WithError(err).Error("删除用户及归还数据失败")
		res.FailWithMessage("删除失败，请稍后重试", c)
		return
	}

	global.Log.Infof("用户 %d 删除成功，其绑定的 SKU 和产品数据已归还给管理员", req.UserID)
	res.OkWithMessage("用户删除成功，SKU 和产品归属已归还", c)
}

type UserExitPasswordRequest struct {
	Password string `json:"password" binding:"required" msg:"请输入密码"` // 密码
}

// 管理员创建用户

// UserUpdatePasswordView 处理更新当前用户密码
func (UserApi) UserUpdatePasswordView(c *gin.Context) {
	var req UserExitPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		res.FailWithError(err, &req, c)
		return
	}

	// 2. 获取当前用户信息（从 JWT）
	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.CustomClaims)

	var user models.UserModel
	if result := global.DB.Where("id = ?", claims.UserID).First(&user); result.Error != nil {
		res.FailWithMessage("用户不存在", c)
		return
	}
	if req.Password == "" {
		res.FailWithMessage("密码不能为空", c)
		return
	}
	user.Password = req.Password

	// 4. 保存到数据库
	if result := global.DB.Save(&user); result.Error != nil {
		res.FailWithMessage("更新失败："+result.Error.Error(), c)
		return
	}
	// 5. 返回成功
	res.OkWithMessage("密码更新成功", c)
}
