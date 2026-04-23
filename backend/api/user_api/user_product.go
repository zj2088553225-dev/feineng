package user_api

import (
	"backend/global"
	"backend/models"
	"backend/models/res"
	"backend/service/common"
	"backend/untils/jwts"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strconv"
)

// 用户查看他们自己绑定的sku的产品
// 在你的 API 处理函数中
func (UserApi) GetUserProductView(c *gin.Context) {
	var pageInfo models.PageInfo
	if err := c.ShouldBindQuery(&pageInfo); err != nil {
		res.FailWithMessage("请求参数无效: "+err.Error(), c)
		return
	}
	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.CustomClaims)

	// ✅ 指定支持模糊搜索的数据库字段名
	list, count, err := common.ComList(
		models.UserProduct{UserID: claims.UserID},
		common.Option{
			PageInfo: pageInfo,
			Likes: []string{
				"user_name",
				"name_en",
				"name_zh",
				"seller_sku",
				"jumia_sku",
				"country_name",
			},
			CustomCond: func(db *gorm.DB) *gorm.DB {
				// 精确匹配 country_name 和 status
				if pageInfo.CountryName != "" {
					db = db.Where("country_name = ?", pageInfo.CountryName)
				}
				return db
			},
		},
	)
	if err != nil {
		res.FailWithMessage(err.Error(), c)
		global.Log.Error(err.Error())
		return
	}
	res.OkWithList(list, count, c)
}

type EditUserProductRequest struct {
	JumiaSku  string  `json:"jumia_sku" binding:"required"`
	NameZh    *string `json:"name_zh,omitempty"`   // 指针
	Inventory *int    `json:"inventory,omitempty"` // 指针
	BuyUrl    *string `json:"buy_url,omitempty"`   // 指针
	SellUrl   *string `json:"sell_url,omitempty"`  // 指针
}

// 用户修改他们自己绑定的sku的产品
func (UserApi) EditUserProductView(c *gin.Context) {
	var req EditUserProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		res.FailWithError(err, &req, c)
		return
	}

	if req.JumiaSku == "" {
		res.FailWithMessage("无效的 jumia_sku", c)
		return
	}
	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.CustomClaims)
	// ✅ 2. 查询该产品是否属于当前用户
	var userProduct models.UserProduct
	err := global.DB.Where("jumia_sku = ? and user_id = ?", req.JumiaSku, claims.UserID).First(&userProduct).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			res.FailWithMessage("该产品不存在或您无权修改", c)
		} else {
			res.FailWithMessage("查询产品失败: "+err.Error(), c)
		}
		return
	}

	// ✅ 构建更新字段：只更新“传了”的字段
	updateData := make(map[string]interface{})

	if req.NameZh != nil {
		updateData["name_zh"] = *req.NameZh
	}
	if req.Inventory != nil {
		updateData["inventory"] = *req.Inventory
	}
	if req.BuyUrl != nil {
		updateData["buy_url"] = *req.BuyUrl
	}
	if req.SellUrl != nil {
		updateData["sell_url"] = *req.SellUrl
	}

	// 如果没有字段要更新，可直接返回
	if len(updateData) == 0 {
		res.OkWithMessage("无更新数据", c)
		return
	}

	// ✅ 4. 使用 Select 或 Updates 避免零值覆盖
	err = global.DB.Model(&userProduct).Updates(updateData).Error
	if err != nil {
		res.FailWithMessage("更新产品失败: "+err.Error(), c)
		return
	}

	res.OkWithMessage("修改产品数据成功", c)
}

// 查看所有用户绑定的SKU产品（管理员）
func (UserApi) GetAdminProductView(c *gin.Context) {
	var pageInfo models.PageInfo
	if err := c.ShouldBindQuery(&pageInfo); err != nil {
		res.FailWithMessage("请求参数无效: "+err.Error(), c)
		return
	}

	query := models.UserProduct{}
	var option common.Option
	if pageInfo.PartnerID > 0 {
		// 查询用户绑定的 seller_sku 列表
		var userSellerSkus []models.UserSellerSkuModel
		err := global.DB.Where("user_id = ?", pageInfo.PartnerID).Pluck("seller_sku", &userSellerSkus).Error
		if err != nil {
			global.Log.Errorf("查询用户 %d 的 seller_sku 失败: %v", pageInfo.PartnerID, err)
			res.FailWithMessage("查询失败，请稍后重试", c)
			return
		}

		if len(userSellerSkus) == 0 {
			res.OkWithList([]models.Order{}, 0, c)
			return
		}

		// 提取 seller_sku 字符串切片
		var sellerSkus []string
		for _, item := range userSellerSkus {
			sellerSkus = append(sellerSkus, item.SellerSku)
		}
		// 当前分页查询的是
		option = common.Option{
			PageInfo: pageInfo,
			Likes: []string{
				"user_name",
				"name_en",
				"name_zh",
				"seller_sku",
				"jumia_sku",
				"country_name",
			},
			CustomCond: func(db *gorm.DB) *gorm.DB {
				db = db.Where("seller_sku IN ?", sellerSkus)

				// 精确匹配 country_name 和 status
				if pageInfo.CountryName != "" {
					db = db.Where("country_name = ?", pageInfo.CountryName)
				}
				return db
			},
		}
	} else {
		// 当前分页查询的是
		option = common.Option{
			PageInfo: pageInfo,
			Likes: []string{
				"user_name",
				"name_en",
				"name_zh",
				"seller_sku",
				"jumia_sku",
				"country_name",
			},
			CustomCond: func(db *gorm.DB) *gorm.DB {
				// 精确匹配 country_name 和 status
				if pageInfo.CountryName != "" {
					db = db.Where("country_name = ?", pageInfo.CountryName)
				}
				return db
			},
		}
	}
	// ✅ 指定支持模糊搜索的数据库字段名
	// 调用通用分页查询 OrderItem
	list, count, err := common.ComList(query, option)
	if err != nil {
		res.FailWithMessage(err.Error(), c)
		global.Log.Error(err.Error())
		return
	}
	res.OkWithList(list, count, c)
}

type UserProductRequest struct {
	SellerSku string `json:"seller_sku" binding:"required"`
	UserID    string `json:"user_id" binding:"required"`
}

// 管理员更换产品合伙人

func (UserApi) AdminUpdateProductUserView(c *gin.Context) {
	var req UserProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		res.FailWithError(err, &req, c)
		return
	}

	if req.SellerSku == "" || req.UserID == "" {
		res.FailWithMessage("seller_sku 和 user_id 为必填项", c)
		return
	}

	newUserID, err := strconv.ParseUint(req.UserID, 10, 32)
	if err != nil {
		res.FailWithMessage("无效的用户ID", c)
		return
	}

	var userName string

	// 使用事务保证数据一致性
	err = global.DB.Transaction(func(tx *gorm.DB) error {
		// ✅ 1️⃣ 在事务中查询 user_name（连接复用）
		err := tx.
			Model(&models.UserModel{}).
			Where("id = ?", uint(newUserID)).
			Select("user_name").
			Scan(&userName).
			Error
		if err != nil {
			return errors.New("未找到该用户")
		}
		if userName == "" {
			return errors.New("该用户未设置用户名")
		}

		// ✅ 2️⃣ 更新 user_product
		result := tx.
			Model(&models.UserProduct{}).
			Where("seller_sku = ?", req.SellerSku).
			Updates(map[string]interface{}{
				"user_id":   uint(newUserID),
				"user_name": userName,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.New("未找到 seller_sku 为 " + req.SellerSku + " 的产品")
		}

		// ✅ 3️⃣ 使用 Upsert 替代 Delete + Insert
		newBinding := models.UserSellerSkuModel{
			UserID:    uint(newUserID),
			SellerSku: req.SellerSku,
		}

		err = tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "seller_sku"}}, // 假设 seller_sku 是唯一键
			DoUpdates: clause.Assignments(map[string]interface{}{
				"user_id": uint(newUserID),
			}),
		}).Create(&newBinding).Error

		if err != nil {
			return errors.New("更新绑定关系失败: " + err.Error())
		}

		return nil
	})

	if err != nil {
		res.FailWithMessage("更新失败: "+err.Error(), c)
		return
	}

	res.OkWithData(gin.H{
		"message":       "合伙人更换成功",
		"seller_sku":    req.SellerSku,
		"new_user_id":   newUserID,
		"new_user_name": userName,
	}, c)
}

type BatchUpdateProductUserRequest struct {
	UserID   string        `json:"user_id" binding:"required"`
	Products []ProductItem `json:"products" binding:"required,min=1,dive"`
}

type ProductItem struct {
	SellerSku string `json:"seller_sku" binding:"required"`
}

// AdminBatchUpdateProductUserView 管理员批量更换产品合伙人
// AdminBatchUpdateProductUserView 管理员批量更换产品合伙人（高性能版）
func (UserApi) AdminBatchUpdateProductUserView(c *gin.Context) {
	var req BatchUpdateProductUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		res.FailWithError(err, &req, c)
		return
	}

	// 转换 UserID
	newUserID, err := strconv.ParseUint(req.UserID, 10, 32)
	if err != nil {
		res.FailWithMessage("无效的用户ID", c)
		return
	}

	var userName string
	// 查询新用户是否存在并获取用户名
	err = global.DB.
		Model(&models.UserModel{}).
		Where("id = ?", uint(newUserID)).
		Select("user_name").
		Scan(&userName).
		Error

	if err != nil || userName == "" {
		res.FailWithMessage("未找到有效用户", c)
		return
	}

	// 去重并提取 seller_skus
	sellerSkus := make([]string, 0, len(req.Products))
	seen := make(map[string]struct{})
	for _, p := range req.Products {
		if _, exists := seen[p.SellerSku]; !exists {
			seen[p.SellerSku] = struct{}{}
			sellerSkus = append(sellerSkus, p.SellerSku)
		}
	}

	if len(sellerSkus) == 0 {
		res.FailWithMessage("未提供有效的 seller_sku", c)
		return
	}

	// 记录原始总数
	total := len(sellerSkus)

	// 使用事务处理所有变更
	var notFoundSkus []string
	var dbErr error

	err = global.DB.Transaction(func(tx *gorm.DB) error {
		// 1️⃣ 批量更新 user_product 表
		result := tx.
			Model(&models.UserProduct{}).
			Where("seller_sku IN ?", sellerSkus).
			Updates(map[string]interface{}{
				"user_id":   uint(newUserID),
				"user_name": userName,
			})

		if result.Error != nil {
			return result.Error
		}

		// 记录未更新到的 SKU（即不存在的）
		if result.RowsAffected < int64(total) {
			// 查询哪些 SKU 实际存在
			var existingSkus []string
			err := tx.Model(&models.UserProduct{}).
				Where("seller_sku IN ?", sellerSkus).
				Pluck("seller_sku", &existingSkus).Error
			if err != nil {
				return err
			}

			// 找出不存在的
			existingMap := make(map[string]bool)
			for _, sku := range existingSkus {
				existingMap[sku] = true
			}
			for _, sku := range sellerSkus {
				if !existingMap[sku] {
					notFoundSkus = append(notFoundSkus, sku)
				}
			}
		}

		// 2️⃣ 批量删除旧的绑定关系
		if err := tx.Where("seller_sku IN ?", sellerSkus).Delete(&models.UserSellerSkuModel{}).Error; err != nil {
			return err
		}

		// 3️⃣ 批量插入新的绑定关系
		var bindings []models.UserSellerSkuModel
		for _, sku := range sellerSkus {
			if !contains(notFoundSkus, sku) { // 只插入存在的 SKU
				bindings = append(bindings, models.UserSellerSkuModel{
					UserID:    uint(newUserID),
					SellerSku: sku,
				})
			}
		}

		if len(bindings) > 0 {
			// 使用 CreateInBatches 提高插入性能
			dbErr = tx.CreateInBatches(bindings, 100).Error
			if dbErr != nil {
				return dbErr
			}
		}

		return nil
	})

	// === 构造响应 ===
	successCount := total - len(notFoundSkus)
	failCount := len(notFoundSkus)

	var message string
	if failCount == 0 {
		message = "全部成功"
	} else if successCount == 0 {
		message = "全部失败"
	} else {
		message = fmt.Sprintf("部分成功（%d 成功，%d 失败）", successCount, failCount)
	}

	responseData := gin.H{
		"message":       message,
		"total":         total,
		"success_count": successCount,
		"fail_count":    failCount,
		"success_list":  subtractStringSlice(sellerSkus, notFoundSkus),
		"fail_list":     notFoundSkus,
		"new_user": gin.H{
			"user_id":   newUserID,
			"user_name": userName,
		},
	}

	// 状态码逻辑
	if failCount == 0 {
		res.OkWithData(responseData, c)
	} else if successCount == 0 {
		res.FailWithMessage("所有 seller_sku 均未找到", c)
	} else {
		res.OkWithData(responseData, c) // 部分成功仍返回 200
	}
}

// contains 判断字符串是否在切片中
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// subtractStringSlice a - b
func subtractStringSlice(a, b []string) []string {
	bMap := make(map[string]bool)
	for _, s := range b {
		bMap[s] = true
	}
	var result []string
	for _, s := range a {
		if !bMap[s] {
			result = append(result, s)
		}
	}
	return result
}

// 管理员修改用户绑定的sku的产品
func (UserApi) AdminEditUserProductView(c *gin.Context) {
	var req EditUserProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		res.FailWithError(err, &req, c)
		return
	}

	if req.JumiaSku == "" {
		res.FailWithMessage("无效的 jumia_sku", c)
		return
	}

	var userProduct models.UserProduct
	err := global.DB.Where("jumia_sku = ? ", req.JumiaSku).First(&userProduct).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			res.FailWithMessage("该产品不存在", c)
		} else {
			res.FailWithMessage("查询产品失败: "+err.Error(), c)
		}
		return
	}

	// ✅ 构建更新字段：只更新“传了”的字段
	updateData := make(map[string]interface{})

	if req.NameZh != nil {
		updateData["name_zh"] = *req.NameZh
	}
	if req.Inventory != nil {
		updateData["inventory"] = *req.Inventory
	}
	if req.BuyUrl != nil {
		updateData["buy_url"] = *req.BuyUrl
	}
	if req.SellUrl != nil {
		updateData["sell_url"] = *req.SellUrl
	}

	// 如果没有字段要更新，可直接返回
	if len(updateData) == 0 {
		res.OkWithMessage("无更新数据", c)
		return
	}

	// ✅ 4. 使用 Select 或 Updates 避免零值覆盖
	err = global.DB.Model(&userProduct).Updates(updateData).Error
	if err != nil {
		res.FailWithMessage("更新产品失败: "+err.Error(), c)
		return
	}

	res.OkWithMessage("修改产品数据成功", c)
}
