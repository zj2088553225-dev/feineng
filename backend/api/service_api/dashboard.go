package service_api

import (
	"backend/global"
	"backend/models"
	"backend/models/res"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"strings"
)

type PieItem struct {
	Name  string `json:"name"`
	Value int64  `json:"value"`
}

// AdminDashBoardView 控制面板数据统计返回
func (ServiceApi) AdminDashBoardView(c *gin.Context) {
	var pageInfo models.PageInfo
	if err := c.ShouldBindQuery(&pageInfo); err != nil {
		res.FailWithMessage("请求参数无效: "+err.Error(), c)
		return
	}
	if pageInfo.CountryName == "" {
		res.FailWithMessage("请优先选择请求国家", c)
		return
	}

	var result struct {
		SocialSales    []PieItem `json:"social_sales"`
		PlatformOrders []PieItem `json:"platform_orders"`
	}

	// ==========================
	// 图表1：社媒订单 - 按 person 聚合金额
	// ==========================

	// 获取聚合查询（已包含 SELECT 和 GROUP BY）
	db := getSocialOrderAmountDB(pageInfo)
	if db == nil {
		res.FailWithMessage("当前国家暂不支持", c)
		return
	}

	var socialData []struct {
		Person      string  `json:"person"`
		TotalAmount float64 `json:"total_amount" gorm:"column:total_amount"`
	}

	err := db.Scan(&socialData).Error
	if err != nil {
		global.Log.Errorf("社媒订单聚合失败: %v", err)
		res.FailWithMessage("数据查询失败", c)
		return
	}

	for _, item := range socialData {
		result.SocialSales = append(result.SocialSales, PieItem{
			Name:  item.Person,
			Value: int64(item.TotalAmount), // 转为整数，如需小数可改为 float64
		})
	}

	// ==========================
	// 图表2：平台订单 - 按 UserName 聚合订单数量
	// ==========================
	var platformData []struct {
		UserName string
		Count    int64
	}

	subQuery := global.DB.
		Table("order_items").
		Select("users.user_name, COUNT(*) as count"). // ✅ 关键：加上 COUNT(*)
		Joins("LEFT JOIN user_seller_sku_models ON order_items.seller_sku = user_seller_sku_models.seller_sku").
		Joins("LEFT JOIN user_models users ON user_seller_sku_models.user_id = users.id").
		Where("order_items.country_name = ? AND order_items.status = ?", pageInfo.CountryName, "DELIVERED")

	startDate, endDate := formatDates(pageInfo.StartDate, pageInfo.EndDate)
	if startDate != "" {
		subQuery = subQuery.Where("order_items.updated_at >= ?", startDate)
	}
	if endDate != "" {
		subQuery = subQuery.Where("order_items.updated_at <= ?", endDate)
	}

	err = subQuery.
		Group("users.user_name").
		Order("count DESC").
		Scan(&platformData).Error

	if err != nil {
		global.Log.Errorf("平台订单聚合失败: %v", err)
		res.FailWithMessage("订单数据查询失败", c)
		return
	}

	for _, item := range platformData {
		result.PlatformOrders = append(result.PlatformOrders, PieItem{
			Name:  item.UserName,
			Value: item.Count,
		})
	}

	// ==========================
	// 返回结果
	// ==========================
	res.OkWithData(result, c)
}

// getSocialOrderAmountDB 返回按 person 聚合的金额查询（含 SELECT 和 GROUP BY）
func getSocialOrderAmountDB(pageInfo models.PageInfo) *gorm.DB {
	startDate, endDate := formatDates(pageInfo.StartDate, pageInfo.EndDate)

	var db *gorm.DB

	switch strings.ToLower(pageInfo.CountryName) {
	case "ghana", "gh":
		// GH: Amount 是总价
		db = global.DB.Model(&models.CustomizeOrderGH{}).
			Select("person, SUM(CAST(amount AS DECIMAL(15,2))) as total_amount").
			Group("person").
			Where("person != '' AND person IS NOT NULL")

		if startDate != "" {
			db = db.Where("date >= ?", startDate)
		}
		if endDate != "" {
			db = db.Where("date <= ?", endDate)
		}

	case "kenya", "ke":
		// KE: Price * Qty
		db = global.DB.Model(&models.CustomizeOrderKE{}).
			Select("person, SUM(CAST(price AS DECIMAL(15,2)) * CAST(qty AS DECIMAL(15,2))) as total_amount").
			Group("person").
			Where("person != '' AND person IS NOT NULL")

		if startDate != "" {
			db = db.Where("order_date >= ?", startDate)
		}
		if endDate != "" {
			db = db.Where("order_date <= ?", endDate)
		}

	case "nigeria", "ng":
		// NG: Price * Qty
		db = global.DB.Model(&models.CustomizeOrderNG{}).
			Select("person, SUM(CAST(price AS DECIMAL(15,2)) * CAST(qty AS DECIMAL(15,2))) as total_amount").
			Group("person").
			Where("person != '' AND person IS NOT NULL")

		if startDate != "" {
			db = db.Where("time >= ?", startDate)
		}
		if endDate != "" {
			db = db.Where("time <= ?", endDate)
		}

	default:
		return nil
	}

	return db
}

// formatDates 补全时间（开始 00:00:00，结束 23:59:59）
func formatDates(start, end string) (string, string) {
	if start != "" && len(start) == 10 {
		start += " 00:00:00"
	}
	if end != "" && len(end) == 10 {
		end += " 23:59:59"
	}
	return start, end
}
