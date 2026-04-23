package user_api

import (
	"backend/global"
	"backend/models"
	"backend/models/res"
	"backend/service/common"
	"backend/untils/jwts"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 用户获取订单列表
func (UserApi) GetOrderListView(c *gin.Context) {
	var pageInfo models.PageInfo
	if err := c.ShouldBindQuery(&pageInfo); err != nil {
		res.FailWithMessage("请求参数无效: "+err.Error(), c)
		return
	}

	// 获取用户 ID
	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.CustomClaims)

	// 查询用户绑定的 seller_sku 列表
	var userSellerSkus []models.UserSellerSkuModel
	err := global.DB.Where("user_id = ?", claims.UserID).Pluck("seller_sku", &userSellerSkus).Error
	if err != nil {
		global.Log.Errorf("查询用户 %d 的 seller_sku 失败: %v", claims.UserID, err)
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

	// 当前分页查询的是 OrderItem
	query := models.OrderItem{}
	option := common.Option{
		PageInfo: pageInfo,
		Likes: []string{
			"jumia_sku",
			"order_number",
			"seller_sku",
			"country_name",
			"shipping_country_name",
			"status",
		},
		CustomCond: func(db *gorm.DB) *gorm.DB {
			db = db.Where("seller_sku IN ?", sellerSkus)

			// 精确匹配 country_name 和 status
			if pageInfo.CountryName != "" {
				db = db.Where("country_name = ?", pageInfo.CountryName)
			}
			if pageInfo.Status != "" {
				db = db.Where("status = ?", pageInfo.Status)
			}
			// 处理日期范围
			startDate := pageInfo.StartDate
			endDate := pageInfo.EndDate
			if startDate != "" && len(startDate) == 10 {
				startDate += " 00:00:00.000"
			}
			if endDate != "" && len(endDate) == 10 {
				endDate += " 23:59:59.999"
			}
			if startDate != "" && endDate != "" {
				db = db.Where("updated_at BETWEEN ? AND ?", startDate, endDate)
			} else if startDate != "" {
				db = db.Where("updated_at >= ?", startDate)
			} else if endDate != "" {
				db = db.Where("updated_at <= ?", endDate)
			}
			return db
		},
	}

	// 调用通用分页查询 OrderItem
	list, count, err := common.ComList(query, option)
	if err != nil {
		global.Log.Errorf("分页查询订单失败: %v", err)
		res.FailWithMessage("查询失败", c)
		return
	}

	// ✅ 步骤1: 把 list 转成 []OrderItem
	orderItems := list
	if len(orderItems) == 0 {
		res.OkWithList([]models.Order{}, 0, c)
		return
	}

	// ✅ 步骤2: 提取 order_ids
	var orderIDs []string
	for _, item := range orderItems {
		orderIDs = append(orderIDs, item.OrderID)
	}

	// 去重 orderIDs
	seen := make(map[string]struct{})
	var uniqueOrderIDs []string
	for _, id := range orderIDs {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			uniqueOrderIDs = append(uniqueOrderIDs, id)
		}
	}

	// ✅ 步骤3: 查询这些 order_id 对应的 Order 主记录
	var orders []models.Order
	err = global.DB.
		Where("id IN ?", uniqueOrderIDs).
		Find(&orders).Error

	if err != nil {
		global.Log.Errorf("查询 Orders 失败: %v", err)
		res.FailWithMessage("查询失败", c)
		return
	}

	// ✅ 步骤4: 查询这些订单下的所有权限内 OrderItems（也可直接用上面的 list，但要按 order_id 分组）
	var allItems []models.OrderItem
	err = global.DB.
		Where("order_id IN ? AND seller_sku IN ?", uniqueOrderIDs, sellerSkus).
		Find(&allItems).Error

	if err != nil {
		global.Log.Errorf("查询 OrderItems 失败: %v", err)
		res.FailWithMessage("查询失败", c)
		return
	}

	// ✅ 步骤5: 构建 order_id -> []OrderItem 映射
	itemMap := make(map[string][]models.OrderItem)
	for _, item := range allItems {
		itemMap[item.OrderID] = append(itemMap[item.OrderID], item)
	}

	// ✅ 步骤6: 给每个 Order 填充 OrderItems
	for i := range orders {
		orders[i].OrderItems = itemMap[orders[i].ID]
	}

	// ✅ 步骤7: 返回最终结果（Order 列表 + 总数）
	global.Log.Infof("用户 %d 查询订单列表，数量: %d", claims.UserID, count)
	res.OkWithList(orders, count, c) // 注意：count 是 OrderItem 的总数，不是 Order 的去重数
}

// 管理员获取全部的订单列表
func (UserApi) GetAllOrderListView(c *gin.Context) {
	var pageInfo models.PageInfo
	if err := c.ShouldBindQuery(&pageInfo); err != nil {
		res.FailWithMessage("请求参数无效: "+err.Error(), c)
		return
	}
	query := models.OrderItem{}
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

		// 当前分页查询的是 OrderItem

		option = common.Option{
			PageInfo: pageInfo,
			Likes: []string{
				"jumia_sku",
				"order_number",
				"seller_sku",
				"country_name",
				"shipping_country_name",
				"status",
			},
			CustomCond: func(db *gorm.DB) *gorm.DB {
				db = db.Where("seller_sku IN ?", sellerSkus)

				// 精确匹配 country_name 和 status
				if pageInfo.CountryName != "" {
					db = db.Where("country_name = ?", pageInfo.CountryName)
				}
				if pageInfo.Status != "" {
					db = db.Where("status = ?", pageInfo.Status)
				}
				// 处理日期范围
				startDate := pageInfo.StartDate
				endDate := pageInfo.EndDate
				if startDate != "" && len(startDate) == 10 {
					startDate += " 00:00:00.000"
				}
				if endDate != "" && len(endDate) == 10 {
					endDate += " 23:59:59.999"
				}
				if startDate != "" && endDate != "" {
					db = db.Where("updated_at BETWEEN ? AND ?", startDate, endDate)
				} else if startDate != "" {
					db = db.Where("updated_at >= ?", startDate)
				} else if endDate != "" {
					db = db.Where("updated_at <= ?", endDate)
				}
				return db
			},
		}
	} else {
		option = common.Option{
			PageInfo: pageInfo,
			Likes: []string{
				"jumia_sku",
				"order_number",
				"seller_sku",
				"country_name",
				"shipping_country_name",
				"status",
			},
			CustomCond: func(db *gorm.DB) *gorm.DB {
				// 精确匹配 country_name 和 status
				if pageInfo.CountryName != "" {
					db = db.Where("country_name = ?", pageInfo.CountryName)
				}
				if pageInfo.Status != "" {
					db = db.Where("status = ?", pageInfo.Status)
				}
				// 处理日期范围
				startDate := pageInfo.StartDate
				endDate := pageInfo.EndDate
				if startDate != "" && len(startDate) == 10 {
					startDate += " 00:00:00.000"
				}
				if endDate != "" && len(endDate) == 10 {
					endDate += " 23:59:59.999"
				}
				if startDate != "" && endDate != "" {
					db = db.Where("updated_at BETWEEN ? AND ?", startDate, endDate)
				} else if startDate != "" {
					db = db.Where("updated_at >= ?", startDate)
				} else if endDate != "" {
					db = db.Where("updated_at <= ?", endDate)
				}
				return db
			},
		}

		// 调用通用分页查询
	}
	list, count, err := common.ComList(query, option)

	if err != nil {
		global.Log.Errorf("分页查询订单失败: %v", err)
		res.FailWithMessage("查询失败", c)
		return
	}
	// ✅ 步骤1: 把 list 转成 []OrderItem
	orderItems := list
	if len(orderItems) == 0 {
		res.OkWithList([]models.Order{}, 0, c)
		return
	}

	// ✅ 步骤2: 提取 order_ids
	var orderIDs []string
	for _, item := range orderItems {
		orderIDs = append(orderIDs, item.OrderID)
	}

	// 去重 orderIDs
	seen := make(map[string]struct{})
	var uniqueOrderIDs []string
	for _, id := range orderIDs {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			uniqueOrderIDs = append(uniqueOrderIDs, id)
		}
	}

	// ✅ 步骤3: 查询这些 order_id 对应的 Order 主记录
	var orders []models.Order
	err = global.DB.
		Where("id IN ?", uniqueOrderIDs).
		Find(&orders).Error

	if err != nil {
		global.Log.Errorf("查询 Orders 失败: %v", err)
		res.FailWithMessage("查询失败", c)
		return
	}

	// ✅ 步骤4: 查询这些订单下的所有权限内 OrderItems（也可直接用上面的 list，但要按 order_id 分组）
	var allItems []models.OrderItem
	err = global.DB.
		Where("order_id IN ?", uniqueOrderIDs).
		Find(&allItems).Error

	if err != nil {
		global.Log.Errorf("查询 OrderItems 失败: %v", err)
		res.FailWithMessage("查询失败", c)
		return
	}

	// ✅ 步骤5: 构建 order_id -> []OrderItem 映射
	itemMap := make(map[string][]models.OrderItem)
	for _, item := range allItems {
		itemMap[item.OrderID] = append(itemMap[item.OrderID], item)
	}

	// ✅ 步骤6: 给每个 Order 填充 OrderItems
	for i := range orders {
		orders[i].OrderItems = itemMap[orders[i].ID]
	}

	// ✅ 步骤7: 返回最终结果（Order 列表 + 总数）
	global.Log.Infof("管理员查询订单列表，数量: %d", count)
	res.OkWithList(orders, count, c) // 注意：count 是 OrderItem 的总数，不是 Order 的去重数
}
