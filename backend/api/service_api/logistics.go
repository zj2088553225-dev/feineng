package service_api

import (
	"backend/global"
	"backend/models"
	"backend/models/ctype"
	"backend/models/res"
	"backend/service/common"
	"backend/untils/jwts"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type LogisticsListRow struct {
	ID                  string  `json:"id"`
	OrderID             string  `json:"orderId"`
	OrderNumber         string  `json:"orderNumber"`
	TrackingNumber      string  `json:"trackingNumber"`
	TrackingURL         string  `json:"trackingUrl"`
	Status              string  `json:"status"`
	OrderStatus         string  `json:"orderStatus"`
	ProductName         string  `json:"productName"`
	SellerSKU           string  `json:"sellerSku"`
	ImageURL            string  `json:"imageUrl"`
	CountryName         string  `json:"countryName"`
	ShippingCountryName string  `json:"shippingCountryName"`
	ShippingCity        string  `json:"shippingCity"`
	ShippingRegion      string  `json:"shippingRegion"`
	TotalShippingCost   float64 `json:"totalShippingCost"`
	NetProfit           float64 `json:"netProfit"`
	Currency            string  `json:"currency"`
	CreatedAt           any     `json:"createdAt"`
	UpdatedAt           any     `json:"updatedAt"`
}

func (ServiceApi) GetLogisticsListView(c *gin.Context) {
	var pageInfo models.PageInfo
	if err := c.ShouldBindQuery(&pageInfo); err != nil {
		res.FailWithMessage("请求参数无效: "+err.Error(), c)
		return
	}

	if pageInfo.Page <= 0 {
		pageInfo.Page = 1
	}
	if pageInfo.Limit <= 0 {
		pageInfo.Limit = 10
	}
	if pageInfo.Sort == "" {
		pageInfo.Sort = "updated_at desc"
	}

	claimsValue, ok := c.Get("claims")
	if !ok {
		res.FailWithMessage("用户信息不存在", c)
		return
	}
	claims := claimsValue.(*jwts.CustomClaims)

	isAdmin := claims.Role == int(ctype.PermissionAdmin) || claims.Role == int(ctype.PermissionAdminBranchCompany)

	var sellerSkus []string
	if !isAdmin || pageInfo.PartnerID > 0 {
		targetUserID := claims.UserID
		if isAdmin && pageInfo.PartnerID > 0 {
			targetUserID = pageInfo.PartnerID
		}

		if err := global.DB.Model(&models.UserSellerSkuModel{}).
			Where("user_id = ?", targetUserID).
			Pluck("seller_sku", &sellerSkus).Error; err != nil {
			global.Log.Errorf("查询用户 %d 的 seller_sku 失败: %v", targetUserID, err)
			res.FailWithMessage("查询失败，请稍后重试", c)
			return
		}
		if len(sellerSkus) == 0 {
			res.OkWithList([]LogisticsListRow{}, 0, c)
			return
		}
	}

	query := models.OrderItem{}
	option := common.Option{
		PageInfo: pageInfo,
		Likes: []string{
			"order_number",
			"tracking_number",
			"seller_sku",
			"product_name",
		},
		CustomCond: func(db *gorm.DB) *gorm.DB {
			if len(sellerSkus) > 0 {
				db = db.Where("seller_sku IN ?", sellerSkus)
			}
			if pageInfo.CountryName != "" {
				db = db.Where("country_name = ? OR shipping_country_name = ?", pageInfo.CountryName, pageInfo.CountryName)
			}
			if pageInfo.Status != "" {
				db = db.Where("status = ?", pageInfo.Status)
			}

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

	items, count, err := common.ComList(query, option)
	if err != nil {
		global.Log.Errorf("分页查询物流订单失败: %v", err)
		res.FailWithMessage("查询失败", c)
		return
	}
	if len(items) == 0 {
		res.OkWithList([]LogisticsListRow{}, 0, c)
		return
	}

	orderIDs := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		if _, exists := seen[item.OrderID]; exists {
			continue
		}
		seen[item.OrderID] = struct{}{}
		orderIDs = append(orderIDs, item.OrderID)
	}

	var orders []models.Order
	if err := global.DB.Where("id IN ?", orderIDs).Find(&orders).Error; err != nil {
		global.Log.Errorf("查询订单主表失败: %v", err)
		res.FailWithMessage("查询失败", c)
		return
	}

	orderMap := make(map[string]models.Order, len(orders))
	for _, order := range orders {
		orderMap[order.ID] = order
	}

	rows := make([]LogisticsListRow, 0, len(items))
	for _, item := range items {
		order := orderMap[item.OrderID]
		orderNumber := item.OrderNumber
		if orderNumber == "" {
			orderNumber = order.Number
		}
		shippingCountryName := item.ShippingCountryName
		if shippingCountryName == "" {
			shippingCountryName = order.ShippingCountryName
		}
		shippingCity := item.ShippingCity
		if shippingCity == "" {
			shippingCity = order.ShippingCity
		}
		shippingRegion := item.ShippingRegion
		if shippingRegion == "" {
			shippingRegion = order.ShippingRegion
		}
		currency := order.TotalAmountLocalCurrency
		if currency == "" {
			currency = order.CountryCurrency
		}

		rows = append(rows, LogisticsListRow{
			ID:                  item.ID,
			OrderID:             item.OrderID,
			OrderNumber:         orderNumber,
			TrackingNumber:      item.TrackingNumber,
			TrackingURL:         item.TrackingURL,
			Status:              item.Status,
			OrderStatus:         order.Status,
			ProductName:         item.ProductName,
			SellerSKU:           item.SellerSKU,
			ImageURL:            item.ImageURL,
			CountryName:         item.CountryName,
			ShippingCountryName: shippingCountryName,
			ShippingCity:        shippingCity,
			ShippingRegion:      shippingRegion,
			TotalShippingCost:   order.TotalShippingCost,
			NetProfit:           order.NetProfit,
			Currency:            currency,
			CreatedAt:           item.CreatedAt,
			UpdatedAt:           item.UpdatedAt,
		})
	}

	res.OkWithList(rows, count, c)
}
