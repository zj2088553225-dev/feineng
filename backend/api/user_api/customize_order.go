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
func (UserApi) UserCustomizeOrderView(c *gin.Context) {

	// 获取用户 ID
	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.CustomClaims)
	var pageInfo models.PageInfo
	if err := c.ShouldBindQuery(&pageInfo); err != nil {
		res.FailWithMessage("请求参数无效: "+err.Error(), c)
		return
	}
	if pageInfo.CustomizeOrderType == "" {
		res.FailWithMessage("请优先选择请求国家", c)
		return
	}
	var option common.Option
	if pageInfo.CustomizeOrderType == "gh" {
		query := models.CustomizeOrderGH{}
		// 当前分页查询的是
		option = common.Option{
			PageInfo: pageInfo,
			Likes: []string{
				"jumia_sku",
				"gh_id",
				"order_numb",
				"order_number",
				"phone_number",
			},
			CustomCond: func(db *gorm.DB) *gorm.DB {
				db = db.Where("person = ?", claims.UserName)
				if pageInfo.OrderStatus != "" && pageInfo.OrderStatus != "other" {
					db = db.Where("order_done = ?", pageInfo.OrderStatus)
				}
				if pageInfo.OrderStatus == "other" {
					db = db.Where("order_done != ? and order_done != ?", "YES", "No Answer")
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
					db = db.Where("date BETWEEN ? AND ?", startDate, endDate)
				} else if startDate != "" {
					db = db.Where("date >= ?", startDate)
				} else if endDate != "" {
					db = db.Where("date <= ?", endDate)
				}
				db = db.Order("gh_id ASC")
				return db
			},
		}
		list, count, err := common.ComList(query, option)
		if err != nil {
			global.Log.Errorf("分页查询加纳社媒订单失败: %v", err)
			res.FailWithMessage("查询失败", c)
			return
		}
		res.OkWithList(list, count, c)
	} else if pageInfo.CustomizeOrderType == "ke" {
		query := models.CustomizeOrderKE{}
		// 当前分页查询的是
		option = common.Option{
			PageInfo: pageInfo,
			Likes: []string{
				"jumia_sku",
				"ke_id",
				"id",
				"phone_number_2",
				"phone_number",
				"order_number",
			},
			CustomCond: func(db *gorm.DB) *gorm.DB {
				// 精确匹配合伙人名
				db = db.Where("person = ?", claims.UserName)

				if pageInfo.OrderStatus != "" && pageInfo.OrderStatus != "other" {
					db = db.Where("order_status = ?", pageInfo.OrderStatus)
				}
				if pageInfo.OrderStatus == "other" {
					db = db.Where("order_status != ? and order_status != ?", "YES", "No Answer")
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
					db = db.Where("order_date BETWEEN ? AND ?", startDate, endDate)
				} else if startDate != "" {
					db = db.Where("order_date >= ?", startDate)
				} else if endDate != "" {
					db = db.Where("order_date <= ?", endDate)
				}
				db = db.Order("ke_id ASC")
				return db
			},
		}
		list, count, err := common.ComList(query, option)
		if err != nil {
			global.Log.Errorf("分页查询肯尼亚社媒订单失败: %v", err)
			res.FailWithMessage("查询失败", c)
			return
		}
		res.OkWithList(list, count, c)
	} else if pageInfo.CustomizeOrderType == "ng" {
		query := models.CustomizeOrderNG{}
		// 当前分页查询的是
		option = common.Option{
			PageInfo: pageInfo,
			Likes: []string{
				"jumia_sku",
				"ng_id",
				"id",
				"phone_number",
				"phone_number_2",
				"order_number",
			},
			CustomCond: func(db *gorm.DB) *gorm.DB {
				// 精确匹配合伙人名
				db = db.Where("person = ?", claims.UserName)

				if pageInfo.OrderStatus != "" && pageInfo.OrderStatus != "other" {
					db = db.Where("order_status = ?", pageInfo.OrderStatus)
				}
				if pageInfo.OrderStatus == "other" {
					db = db.Where("order_status != ? and order_status != ?", "YES", "No Answer")
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
					db = db.Where("time BETWEEN ? AND ?", startDate, endDate)
				} else if startDate != "" {
					db = db.Where("time >= ?", startDate)
				} else if endDate != "" {
					db = db.Where("time <= ?", endDate)
				}
				db = db.Order("ng_id ASC")
				return db
			},
		}
		list, count, err := common.ComList(query, option)
		if err != nil {
			global.Log.Errorf("分页查询尼日利亚社媒订单失败: %v", err)
			res.FailWithMessage("查询失败", c)
			return
		}
		res.OkWithList(list, count, c)
	} else {
		res.FailWithMessage("当前国家暂不支持", c)
	}
}

// 管理员产看所有用户的的交易记录
func (UserApi) AdminCustomizeOrderView(c *gin.Context) {
	var pageInfo models.PageInfo
	if err := c.ShouldBindQuery(&pageInfo); err != nil {
		res.FailWithMessage("请求参数无效: "+err.Error(), c)
		return
	}
	if pageInfo.CustomizeOrderType == "" {
		res.FailWithMessage("请优先选择请求国家", c)
		return
	}
	var option common.Option
	if pageInfo.CustomizeOrderType == "gh" {
		query := models.CustomizeOrderGH{}
		// 当前分页查询的是
		option = common.Option{
			PageInfo: pageInfo,
			Likes: []string{
				"jumia_sku",
				"gh_id",
				"order_numb",
				"order_number",
				"phone_number",
			},
			CustomCond: func(db *gorm.DB) *gorm.DB {
				// 精确匹配合伙人名
				if pageInfo.Person != "" {
					db = db.Where("person = ?", pageInfo.Person)
				}
				if pageInfo.OrderStatus != "" && pageInfo.OrderStatus != "other" {
					db = db.Where("order_done = ?", pageInfo.OrderStatus)
				}
				if pageInfo.OrderStatus == "other" {
					db = db.Where("order_done != ? and order_done != ?", "YES", "No Answer")
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
					db = db.Where("date BETWEEN ? AND ?", startDate, endDate)
				} else if startDate != "" {
					db = db.Where("date >= ?", startDate)
				} else if endDate != "" {
					db = db.Where("date <= ?", endDate)
				}
				db = db.Order("gh_id ASC")
				return db
			},
		}
		list, count, err := common.ComList(query, option)
		if err != nil {
			global.Log.Errorf("分页查询加纳社媒订单失败: %v", err)
			res.FailWithMessage("查询失败", c)
			return
		}
		res.OkWithList(list, count, c)
	} else if pageInfo.CustomizeOrderType == "ke" {
		query := models.CustomizeOrderKE{}
		// 当前分页查询的是
		option = common.Option{
			PageInfo: pageInfo,
			Likes: []string{
				"jumia_sku",
				"ke_id",
				"id",
				"phone_number_2",
				"phone_number",
				"order_number",
			},
			CustomCond: func(db *gorm.DB) *gorm.DB {
				// 精确匹配合伙人名
				if pageInfo.Person != "" {
					db = db.Where("person = ?", pageInfo.Person)
				}
				if pageInfo.OrderStatus != "" && pageInfo.OrderStatus != "other" {
					db = db.Where("order_status = ?", pageInfo.OrderStatus)
				}
				if pageInfo.OrderStatus == "other" {
					db = db.Where("order_status != ? and order_status != ?", "YES", "No Answer")
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
					db = db.Where("call_date BETWEEN ? AND ?", startDate, endDate)
				} else if startDate != "" {
					db = db.Where("call_date >= ?", startDate)
				} else if endDate != "" {
					db = db.Where("call_date <= ?", endDate)
				}
				db = db.Order("ke_id ASC")
				return db
			},
		}
		list, count, err := common.ComList(query, option)
		if err != nil {
			global.Log.Errorf("分页查询肯尼亚社媒订单失败: %v", err)
			res.FailWithMessage("查询失败", c)
			return
		}
		res.OkWithList(list, count, c)
	} else if pageInfo.CustomizeOrderType == "ng" {
		query := models.CustomizeOrderNG{}
		// 当前分页查询的是
		option = common.Option{
			PageInfo: pageInfo,
			Likes: []string{
				"jumia_sku",
				"ng_id",
				"id",
				"phone_number",
				"phone_number_2",
				"order_number",
			},
			CustomCond: func(db *gorm.DB) *gorm.DB {
				// 精确匹配合伙人名
				if pageInfo.Person != "" {
					db = db.Where("person = ?", pageInfo.Person)
				}
				if pageInfo.OrderStatus != "" && pageInfo.OrderStatus != "other" {
					db = db.Where("order_status = ?", pageInfo.OrderStatus)
				}
				if pageInfo.OrderStatus == "other" {
					db = db.Where("order_status != ? and order_status != ?", "YES", "No Answer")
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
					db = db.Where("date BETWEEN ? AND ?", startDate, endDate)
				} else if startDate != "" {
					db = db.Where("date >= ?", startDate)
				} else if endDate != "" {
					db = db.Where("date <= ?", endDate)
				}
				db = db.Order("ng_id ASC")
				return db
			},
		}
		list, count, err := common.ComList(query, option)
		if err != nil {
			global.Log.Errorf("分页查询尼日利亚社媒订单失败: %v", err)
			res.FailWithMessage("查询失败", c)
			return
		}
		res.OkWithList(list, count, c)
	} else {
		res.FailWithMessage("当前国家暂不支持", c)
	}

}
