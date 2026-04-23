package service_api

import (
	"backend/global"
	"backend/models"
	"backend/models/res"
	"backend/untils/jwts"
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"strings"
	"time"
)

type UserDashboardResponse struct {
	SocialAmount    int64 `json:"social_amount"`     // 社媒销售金额（整数）
	TodaySalesCount int64 `json:"today_sales_count"` // 出售数量（条数码）
}

// UserDashBoardView 用户看板：支持国家和时间筛选
func (ServiceApi) UserDashBoardView(c *gin.Context) {
	var pageInfo models.PageInfo
	if err := c.ShouldBindQuery(&pageInfo); err != nil {
		res.FailWithMessage("请求参数无效: "+err.Error(), c)
		return
	}
	if pageInfo.CountryName == "" {
		res.FailWithMessage("请指定国家", c)
		return
	}

	// 获取当前用户
	_claims, _ := c.Get("claims")
	claims, _ := _claims.(*jwts.CustomClaims)

	userID := claims.UserID
	userName := claims.UserName

	// 格式化时间范围
	startDate, endDate := formatDates(pageInfo.StartDate, pageInfo.EndDate)
	if startDate == "" && endDate == "" {
		// 默认今天
		now := time.Now()
		startDate = now.Format("2006-01-02 00:00:00")
		endDate = now.Format("2006-01-02 23:59:59")
	}

	var result UserDashboardResponse

	// ==========================
	// 1. 统计社媒销售金额（按国家 + 时间 + person = userName）
	// ==========================
	socialAmount, err := getUserSocialAmountByCountry(userName, pageInfo.CountryName, startDate, endDate)
	if err != nil {
		global.Log.Errorf("用户 %s 社媒金额统计失败: %v", userName, err)
	}
	result.SocialAmount = socialAmount

	// ==========================
	// 2. 统计平台订单项数量（条数码）
	// ==========================
	count, err := getUserPlatformSalesCount(userID, pageInfo.CountryName, startDate, endDate)
	if err != nil {
		global.Log.Errorf("用户 %d 平台销量统计失败: %v", userID, err)
	}
	result.TodaySalesCount = count

	// ==========================
	// 返回结果
	// ==========================
	res.OkWithData(result, c)
}

// getUserSocialAmountByCountry 根据国家统计用户社媒销售额
func getUserSocialAmountByCountry(userName, countryName, startDate, endDate string) (int64, error) {
	var totalAmount float64
	var err error

	switch strings.ToLower(countryName) {
	case "ghana", "gh":
		err = global.DB.Model(&models.CustomizeOrderGH{}).
			Select("SUM(CAST(amount AS DECIMAL(15,2)))").
			Where("person = ? AND date BETWEEN ? AND ?", userName, startDate, endDate).
			Scan(&totalAmount).Error

	case "kenya", "ke":
		err = global.DB.Model(&models.CustomizeOrderKE{}).
			Select("SUM(CAST(price AS DECIMAL(15,2)) * CAST(qty AS DECIMAL(15,2)))").
			Where("person = ? AND order_date BETWEEN ? AND ?", userName, startDate, endDate).
			Scan(&totalAmount).Error

	case "nigeria", "ng":
		err = global.DB.Model(&models.CustomizeOrderNG{}).
			Select("SUM(CAST(price AS DECIMAL(15,2)) * CAST(qty AS DECIMAL(15,2)))").
			Where("person = ? AND time BETWEEN ? AND ?", userName, startDate, endDate).
			Scan(&totalAmount).Error

	default:
		return 0, nil // 国家不支持，返回 0
	}

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}

	return int64(totalAmount), nil
}

// getUserPlatformSalesCount 统计用户在指定国家、时间内的平台销售条数
func getUserPlatformSalesCount(userID uint, countryName, startDate, endDate string) (int64, error) {
	var count int64

	// 1. 查询用户绑定的所有 seller_sku
	var sellerSkus []string
	err := global.DB.Model(&models.UserSellerSkuModel{}).
		Where("user_id = ?", userID).
		Pluck("seller_sku", &sellerSkus).Error

	if err != nil {
		return 0, err
	}

	if len(sellerSkus) == 0 {
		return 0, nil // 未绑定 SKU
	}

	// 2. 统计 order_items 中匹配的记录（已发货）
	err = global.DB.Model(&models.OrderItem{}).
		Where("seller_sku IN ? AND country_name = ? AND status = 'DELIVERED'", sellerSkus, countryName).
		Where("updated_at BETWEEN ? AND ?", startDate, endDate).
		Count(&count).Error

	if err != nil {
		return 0, err
	}

	return count, nil
}
