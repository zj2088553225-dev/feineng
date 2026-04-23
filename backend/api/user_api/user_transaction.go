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

// 用户产看自己的交易记录
func (UserApi) UserTransactionView(c *gin.Context) {
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
		res.OkWithList([]models.Transaction{}, 0, c)
		return
	}
	// 提取 seller_sku 字符串切片
	var sellerSkus []string
	for _, item := range userSellerSkus {
		sellerSkus = append(sellerSkus, item.SellerSku)
	}
	// 当前分页查询的是 OrderItem
	query := models.Transaction{}
	option := common.Option{
		PageInfo: pageInfo,
		Likes: []string{
			"transaction_date",
			"seller_sku",
			"jumia_sku",
			"order_no",
		},
		CustomCond: func(db *gorm.DB) *gorm.DB {
			db = db.Where("seller_sku IN ?", sellerSkus)

			// 精确匹配 country_name 和 status
			if pageInfo.CountryCode != "" {
				db = db.Where("country_code = ?", pageInfo.CountryCode)
			}
			if pageInfo.PaidStatus != "" {
				db = db.Where("paid_status = ?", pageInfo.PaidStatus)
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
				db = db.Where("transaction_date BETWEEN ? AND ?", startDate, endDate)
			} else if startDate != "" {
				db = db.Where("transaction_date >= ?", startDate)
			} else if endDate != "" {
				db = db.Where("transaction_date <= ?", endDate)
			}
			if pageInfo.TransactionType != "" {
				db = db.Where("transaction_type = ?", pageInfo.TransactionType)
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
	res.OkWithList(list, count, c)
}

// 管理员产看所有用户的的交易记录
func (UserApi) AdminTransactionView(c *gin.Context) {
	var pageInfo models.PageInfo
	if err := c.ShouldBindQuery(&pageInfo); err != nil {
		res.FailWithMessage("请求参数无效: "+err.Error(), c)
		return
	}
	query := models.Transaction{}
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
				"transaction_date",
				"seller_sku",
				"jumia_sku",
				"paid_status",
				"country_code",
				"order_no",
			},
			CustomCond: func(db *gorm.DB) *gorm.DB {
				db = db.Where("seller_sku IN ?", sellerSkus)

				// 精确匹配 country_name 和 status
				if pageInfo.CountryCode != "" {
					db = db.Where("country_code = ?", pageInfo.CountryCode)
				}
				if pageInfo.PaidStatus != "" {
					db = db.Where("paid_status = ?", pageInfo.PaidStatus)
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
					db = db.Where("transaction_date BETWEEN ? AND ?", startDate, endDate)
				} else if startDate != "" {
					db = db.Where("transaction_date >= ?", startDate)
				} else if endDate != "" {
					db = db.Where("transaction_date <= ?", endDate)
				}
				if pageInfo.TransactionType != "" {
					db = db.Where("transaction_type = ?", pageInfo.TransactionType)
				}
				return db
			},
		}
	} else {
		// 当前分页查询的是
		option = common.Option{
			PageInfo: pageInfo,
			Likes: []string{
				"transaction_date",
				"seller_sku",
				"jumia_sku",
				"order_no",
			},
			CustomCond: func(db *gorm.DB) *gorm.DB {
				// 精确匹配 country_name 和 status
				if pageInfo.CountryCode != "" {
					db = db.Where("country_code = ?", pageInfo.CountryCode)
				}
				if pageInfo.PaidStatus != "" {
					db = db.Where("paid_status = ?", pageInfo.PaidStatus)
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
					db = db.Where("transaction_date BETWEEN ? AND ?", startDate, endDate)
				} else if startDate != "" {
					db = db.Where("transaction_date >= ?", startDate)
				} else if endDate != "" {
					db = db.Where("transaction_date <= ?", endDate)
				}
				if pageInfo.TransactionType != "" {
					db = db.Where("transaction_type = ?", pageInfo.TransactionType)
				}
				return db
			},
		}
	}
	// 调用通用分页查询 OrderItem
	list, count, err := common.ComList(query, option)
	if err != nil {
		global.Log.Errorf("分页查询订单失败: %v", err)
		res.FailWithMessage("查询失败", c)
		return
	}
	res.OkWithList(list, count, c)
}
