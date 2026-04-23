package user_api

import (
	"backend/global"
	"backend/models"
	"backend/models/res"
	"backend/service/common"
	"backend/untils/jwts"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"time"
)

// ------------------------- 响应结构体 -------------------------

type CommodityResp struct {
	ID             string `json:"id"`
	CommoditySku   string `json:"commoditySku,omitempty"`
	ShopSku        string `json:"shopSku,omitempty"`
	CommodityCname string `json:"commodityCname,omitempty"`
}

type PackageResp struct {
	ID                  string           `json:"id"`
	Count               int              `json:"count,omitempty"`
	TotalCommodityCount int              `json:"totalCommodityCount,omitempty"`
	NetWeight           float64          `json:"netWeight,omitempty"`
	RoughWeight         float64          `json:"roughWeight,omitempty"`
	Length              float64          `json:"length,omitempty"`
	Width               float64          `json:"width,omitempty"`
	High                float64          `json:"high,omitempty"`
	Commodities         []*CommodityResp `json:"commodities,omitempty"`
}

type CargoResp struct {
	ID       string         `json:"id"`
	PO       string         `json:"po,omitempty"`
	ShopName string         `json:"shopName,omitempty"`
	Packages []*PackageResp `json:"packages,omitempty"`
}

type TrajectoryResp struct {
	ID        uint       `json:"id"`
	OpLink    string     `json:"opLink,omitempty"`
	Timestamp *time.Time `json:"timestamp,omitempty"`
}

type OrderResp struct {
	ID                      string            `json:"id"`
	CreateTime              models.CustomTime `json:"createTime,omitempty"`
	HBL                     string            `json:"hbl,omitempty"`
	ChannelName             string            `json:"channelName,omitempty"`
	Status                  string            `json:"status,omitempty"`
	StatusName              string            `json:"statusName,omitempty"`
	TotalCount              int               `json:"totalCount,omitempty"`
	TotalNetWeight          float64           `json:"totalNetWeight,omitempty"`
	TotalRoughWeight        float64           `json:"totalRoughWeight,omitempty"`
	TotalCBM                float64           `json:"totalCBM,omitempty"`
	TotalCommodityCount     int               `json:"totalCommodityCount,omitempty"`
	TotalRoughWeightStorage float64           `json:"totalRoughWeightStorage,omitempty"`
	TotalCBMStorage         float64           `json:"totalCBMStorage,omitempty"`
	Trajectories            []*TrajectoryResp `json:"trajectories,omitempty"`
	Cargos                  []*CargoResp      `json:"cargos,omitempty"`
}

// ------------------------- AdminWuliuView -------------------------

func (UserApi) UserWuliuView(c *gin.Context) {
	var pageInfo models.PageInfo
	if err := c.ShouldBindQuery(&pageInfo); err != nil {
		res.FailWithMessage("请求参数无效: "+err.Error(), c)
		return
	}
	// 获取用户 ID
	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.CustomClaims)
	var query models.CesFbjCommodityInfo
	var option common.Option

	var userSellerSkus []models.UserSellerSkuModel
	if err := global.DB.Where("user_id = ?", claims.UserID).Pluck("seller_sku", &userSellerSkus).Error; err != nil {
		res.FailWithMessage("查询失败，请稍后重试", c)
		return
	}
	if len(userSellerSkus) == 0 {
		res.OkWithList([]OrderResp{}, 0, c)
		return
	}
	sellerSkus := make([]string, len(userSellerSkus))
	for i, item := range userSellerSkus {
		sellerSkus[i] = item.SellerSku
	}
	option = common.Option{
		PageInfo: pageInfo,
		Likes:    []string{"commodity_sku", "shop_sku", "commodity_cname", "hbl"},
		CustomCond: func(db *gorm.DB) *gorm.DB {
			db = db.Where("commodity_sku IN ? or shop_sku IN ?", sellerSkus, sellerSkus)
			//if key != "" {
			//	db = db.Or("shop_sku = ?", key)
			//}
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
				db = db.Where("create_time BETWEEN ? AND ?", startDate, endDate)
			} else if startDate != "" {
				db = db.Where("create_time >= ?", startDate)
			} else if endDate != "" {
				db = db.Where("create_time <= ?", endDate)
			}
			return db
		},
	}

	// 1️⃣ 查询匹配的 commodities
	commodities, count, err := common.ComList(query, option)
	if err != nil {
		res.OkWithList([]OrderResp{}, 0, c)
		return
	}

	// 2️⃣ 获取所有 cargoIDs
	var cargoIDs []string
	for _, c := range commodities {
		if c.PackageID != "" {
			var pkg models.CesFbjPackage
			if err := global.DB.Where("id = ?", c.PackageID).First(&pkg).Error; err == nil {
				if pkg.CargoID != "" {
					cargoIDs = append(cargoIDs, pkg.CargoID)
				}
			}
		}
	}

	// 3️⃣ 查询所有 cargos
	var cargos []models.CesFbjCargoInfo
	if len(cargoIDs) > 0 {
		if err := global.DB.Where("id IN ?", cargoIDs).Find(&cargos).Error; err != nil {
			res.FailWithMessage("加载关联数据失败", c)
			return
		}
	}

	// 4️⃣ 查询 cargos 下所有 packages（完整列表）
	var allPackages []models.CesFbjPackage
	if len(cargoIDs) > 0 {
		if err := global.DB.Where("cargo_id IN ?", cargoIDs).Find(&allPackages).Error; err != nil {
			res.FailWithMessage("加载关联数据失败", c)
			return
		}
	}

	// 5️⃣ 查询 packages 对应的全部 commodities
	var allPackageIDs []string
	for _, p := range allPackages {
		allPackageIDs = append(allPackageIDs, p.ID)
	}
	var allCommodities []models.CesFbjCommodityInfo
	if len(allPackageIDs) > 0 {
		if err := global.DB.Where("package_id IN ?", allPackageIDs).Find(&allCommodities).Error; err != nil {
			res.FailWithMessage("加载关联数据失败", c)
			return
		}
	}

	// 6️⃣ 构建 packageID -> commodities map
	packageCommodityMap := make(map[string][]*CommodityResp)
	for _, c := range allCommodities {
		cr := &CommodityResp{
			ID:             c.ID,
			CommoditySku:   c.CommoditySku,
			ShopSku:        c.ShopSku,
			CommodityCname: c.CommodityCname,
		}
		packageCommodityMap[c.PackageID] = append(packageCommodityMap[c.PackageID], cr)
	}

	// 7️⃣ 构建 packageID -> PackageResp map
	packageMap := make(map[string]*PackageResp)
	for _, p := range allPackages {
		pr := &PackageResp{
			ID:                  p.ID,
			Count:               p.Count,
			TotalCommodityCount: p.TotalCommodityCount,
			NetWeight:           p.NetWeight,
			RoughWeight:         p.RoughWeight,
			Length:              p.Length,
			Width:               p.Width,
			High:                p.High,
			Commodities:         packageCommodityMap[p.ID],
		}
		packageMap[p.ID] = pr
	}

	// 8️⃣ 构建 cargoID -> CargoResp map
	cargoMap := make(map[string]*CargoResp)
	for _, c := range cargos {
		cr := &CargoResp{
			ID:       c.ID,
			PO:       c.PO,
			ShopName: c.ShopName,
		}
		for _, p := range allPackages {
			if p.CargoID == c.ID {
				cr.Packages = append(cr.Packages, packageMap[p.ID])
			}
		}
		cargoMap[c.ID] = cr
	}

	// 9️⃣ 获取所有 orderIDs
	orderIDMap := make(map[string]struct{})
	for _, c := range cargos {
		if c.OrderID != "" {
			orderIDMap[c.OrderID] = struct{}{}
		}
	}
	var orderIDs []string
	for k := range orderIDMap {
		orderIDs = append(orderIDs, k)
	}

	// 🔟 查询 Orders
	var orders []models.CesFbjOrder
	if len(orderIDs) > 0 {
		if err := global.DB.Where("id IN ?", orderIDs).Find(&orders).Error; err != nil {
			res.FailWithMessage("加载关联数据失败", c)
			return
		}
	}

	// 1️⃣1️⃣ 查询 Trajectories
	var trajectories []models.CesFbjOrderTrajectory
	if len(orderIDs) > 0 {
		if err := global.DB.Where("order_id IN ?", orderIDs).Order("timestamp ASC").Find(&trajectories).Error; err != nil {
			res.FailWithMessage("加载关联数据失败", c)
			return
		}
	}

	// 1️⃣2️⃣ 构建 orderID -> OrderResp map
	orderMap := make(map[string]*OrderResp)
	for _, o := range orders {
		orderMap[o.ID] = &OrderResp{
			ID:                      o.ID,
			CreateTime:              o.CreateTime,
			HBL:                     o.HBL,
			ChannelName:             o.ChannelName,
			Status:                  o.Status,
			StatusName:              o.StatusName,
			TotalCount:              o.TotalCount,
			TotalNetWeight:          o.TotalNetWeight,
			TotalRoughWeight:        o.TotalRoughWeight,
			TotalCBM:                o.TotalCBM,
			TotalCommodityCount:     o.TotalCommodityCount,
			TotalRoughWeightStorage: o.TotalRoughWeightStorage,
			TotalCBMStorage:         o.TotalCBMStorage,
		}
	}

	// 1️⃣3️⃣ Trajectories -> Order
	for _, t := range trajectories {
		if o, ok := orderMap[t.OrderID]; ok {
			o.Trajectories = append(o.Trajectories, &TrajectoryResp{
				ID:        t.ID,
				OpLink:    t.OpLink,
				Timestamp: t.Timestamp,
			})
		}
	}

	// 1️⃣4️⃣ Cargos -> Order
	for _, c := range cargos {
		if o, ok := orderMap[c.OrderID]; ok {
			o.Cargos = append(o.Cargos, cargoMap[c.ID])
		}
	}

	// 1️⃣5️⃣ 构建最终返回
	var respList []OrderResp
	for _, o := range orderMap {
		respList = append(respList, *o)
	}

	res.OkWithList(respList, count, c)
}
func (UserApi) AdminWuliuView(c *gin.Context) {
	var pageInfo models.PageInfo
	if err := c.ShouldBindQuery(&pageInfo); err != nil {
		res.FailWithMessage("请求参数无效: "+err.Error(), c)
		return
	}

	var query models.CesFbjCommodityInfo
	var option common.Option

	// 过滤条件
	if pageInfo.PartnerID > 0 {
		var userSellerSkus []models.UserSellerSkuModel
		if err := global.DB.Where("user_id = ?", pageInfo.PartnerID).Pluck("seller_sku", &userSellerSkus).Error; err != nil {
			res.FailWithMessage("查询失败，请稍后重试", c)
			return
		}
		if len(userSellerSkus) == 0 {
			res.OkWithList([]OrderResp{}, 0, c)
			return
		}
		sellerSkus := make([]string, len(userSellerSkus))
		for i, item := range userSellerSkus {
			sellerSkus[i] = item.SellerSku
		}
		option = common.Option{
			PageInfo: pageInfo,
			Likes:    []string{"commodity_sku", "shop_sku", "commodity_cname", "hbl"},
			CustomCond: func(db *gorm.DB) *gorm.DB {
				db = db.Where("commodity_sku IN ? or shop_sku IN ?", sellerSkus, sellerSkus)
				//if key != "" {
				//	db = db.Or("shop_sku = ?", key)
				//}
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
					db = db.Where("create_time BETWEEN ? AND ?", startDate, endDate)
				} else if startDate != "" {
					db = db.Where("create_time >= ?", startDate)
				} else if endDate != "" {
					db = db.Where("create_time <= ?", endDate)
				}
				return db
			},
		}
	} else {
		option = common.Option{
			PageInfo: pageInfo,
			Likes:    []string{"commodity_sku", "shop_sku", "commodity_cname", "hbl"},
			CustomCond: func(db *gorm.DB) *gorm.DB {
				//if key != "" {
				//	db = db.Where("shop_sku = ?", key)
				//}
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
					db = db.Where("create_time BETWEEN ? AND ?", startDate, endDate)
				} else if startDate != "" {
					db = db.Where("create_time >= ?", startDate)
				} else if endDate != "" {
					db = db.Where("create_time <= ?", endDate)
				}
				return db
			},
		}
	}

	// 1️⃣ 查询匹配的 commodities
	commodities, count, err := common.ComList(query, option)
	if err != nil {
		res.OkWithList([]OrderResp{}, 0, c)
		return
	}

	// 2️⃣ 获取所有 cargoIDs
	var cargoIDs []string
	for _, c := range commodities {
		if c.PackageID != "" {
			var pkg models.CesFbjPackage
			if err := global.DB.Where("id = ?", c.PackageID).First(&pkg).Error; err == nil {
				if pkg.CargoID != "" {
					cargoIDs = append(cargoIDs, pkg.CargoID)
				}
			}
		}
	}

	// 3️⃣ 查询所有 cargos
	var cargos []models.CesFbjCargoInfo
	if len(cargoIDs) > 0 {
		if err := global.DB.Where("id IN ?", cargoIDs).Find(&cargos).Error; err != nil {
			res.FailWithMessage("加载关联数据失败", c)
			return
		}
	}

	// 4️⃣ 查询 cargos 下所有 packages（完整列表）
	var allPackages []models.CesFbjPackage
	if len(cargoIDs) > 0 {
		if err := global.DB.Where("cargo_id IN ?", cargoIDs).Find(&allPackages).Error; err != nil {
			res.FailWithMessage("加载关联数据失败", c)
			return
		}
	}

	// 5️⃣ 查询 packages 对应的全部 commodities
	var allPackageIDs []string
	for _, p := range allPackages {
		allPackageIDs = append(allPackageIDs, p.ID)
	}
	var allCommodities []models.CesFbjCommodityInfo
	if len(allPackageIDs) > 0 {
		if err := global.DB.Where("package_id IN ?", allPackageIDs).Find(&allCommodities).Error; err != nil {
			res.FailWithMessage("加载关联数据失败", c)
			return
		}
	}

	// 6️⃣ 构建 packageID -> commodities map
	packageCommodityMap := make(map[string][]*CommodityResp)
	for _, c := range allCommodities {
		cr := &CommodityResp{
			ID:             c.ID,
			CommoditySku:   c.CommoditySku,
			ShopSku:        c.ShopSku,
			CommodityCname: c.CommodityCname,
		}
		packageCommodityMap[c.PackageID] = append(packageCommodityMap[c.PackageID], cr)
	}

	// 7️⃣ 构建 packageID -> PackageResp map
	packageMap := make(map[string]*PackageResp)
	for _, p := range allPackages {
		pr := &PackageResp{
			ID:                  p.ID,
			Count:               p.Count,
			TotalCommodityCount: p.TotalCommodityCount,
			NetWeight:           p.NetWeight,
			RoughWeight:         p.RoughWeight,
			Length:              p.Length,
			Width:               p.Width,
			High:                p.High,
			Commodities:         packageCommodityMap[p.ID],
		}
		packageMap[p.ID] = pr
	}

	// 8️⃣ 构建 cargoID -> CargoResp map
	cargoMap := make(map[string]*CargoResp)
	for _, c := range cargos {
		cr := &CargoResp{
			ID:       c.ID,
			PO:       c.PO,
			ShopName: c.ShopName,
		}
		for _, p := range allPackages {
			if p.CargoID == c.ID {
				cr.Packages = append(cr.Packages, packageMap[p.ID])
			}
		}
		cargoMap[c.ID] = cr
	}

	// 9️⃣ 获取所有 orderIDs
	orderIDMap := make(map[string]struct{})
	for _, c := range cargos {
		if c.OrderID != "" {
			orderIDMap[c.OrderID] = struct{}{}
		}
	}
	var orderIDs []string
	for k := range orderIDMap {
		orderIDs = append(orderIDs, k)
	}

	// 🔟 查询 Orders
	var orders []models.CesFbjOrder
	if len(orderIDs) > 0 {
		if err := global.DB.Where("id IN ?", orderIDs).Find(&orders).Error; err != nil {
			res.FailWithMessage("加载关联数据失败", c)
			return
		}
	}

	// 1️⃣1️⃣ 查询 Trajectories
	var trajectories []models.CesFbjOrderTrajectory
	if len(orderIDs) > 0 {
		if err := global.DB.Where("order_id IN ?", orderIDs).Order("timestamp ASC").Find(&trajectories).Error; err != nil {
			res.FailWithMessage("加载关联数据失败", c)
			return
		}
	}

	// 1️⃣2️⃣ 构建 orderID -> OrderResp map
	orderMap := make(map[string]*OrderResp)
	for _, o := range orders {
		orderMap[o.ID] = &OrderResp{
			ID:                      o.ID,
			CreateTime:              o.CreateTime,
			HBL:                     o.HBL,
			ChannelName:             o.ChannelName,
			Status:                  o.Status,
			StatusName:              o.StatusName,
			TotalCount:              o.TotalCount,
			TotalNetWeight:          o.TotalNetWeight,
			TotalRoughWeight:        o.TotalRoughWeight,
			TotalCBM:                o.TotalCBM,
			TotalCommodityCount:     o.TotalCommodityCount,
			TotalRoughWeightStorage: o.TotalRoughWeightStorage,
			TotalCBMStorage:         o.TotalCBMStorage,
		}
	}

	// 1️⃣3️⃣ Trajectories -> Order
	for _, t := range trajectories {
		if o, ok := orderMap[t.OrderID]; ok {
			o.Trajectories = append(o.Trajectories, &TrajectoryResp{
				ID:        t.ID,
				OpLink:    t.OpLink,
				Timestamp: t.Timestamp,
			})
		}
	}

	// 1️⃣4️⃣ Cargos -> Order
	for _, c := range cargos {
		if o, ok := orderMap[c.OrderID]; ok {
			o.Cargos = append(o.Cargos, cargoMap[c.ID])
		}
	}

	// 1️⃣5️⃣ 构建最终返回
	var respList []OrderResp
	for _, o := range orderMap {
		respList = append(respList, *o)
	}
	global.Log.Info(count)
	res.OkWithList(respList, count, c)
}
