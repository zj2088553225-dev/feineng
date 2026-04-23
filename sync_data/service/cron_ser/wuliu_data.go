// cron_ser/sync_wuliu.go
package cron_ser

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync_data/global"
	"sync_data/models"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ------------------------- 同步主函数 -------------------------
func SyncAllAccountsWuliuData() {
	SyncWuliuToken()
	tokens := []string{
		global.Config.Wuliu.Token,      // 第一个账号
		global.Config.Wuliu.TokenTwo,   // 第二个账号
		global.Config.Wuliu.TokenThree, // 第三个账号
	}

	for _, token := range tokens {
		if err := SyncDataByToken(token); err != nil {
			global.Log.Errorf("Token %s 同步失败: %v", token, err)
			global.DB.Where("id = ?", 9).Updates(models.ServiceStatus{
				Status:  "错误",
				Message: fmt.Sprintf("同步失败: %v", err),
			})
			return
		} else {
			global.Log.Infof("Token %s 同步完成", token)
		}
	}

	// 全部成功
	global.DB.Where("id = ?", 9).Updates(models.ServiceStatus{
		Status:  "正常",
		Message: "定时同步物流数据成功",
	})
}

// ------------------------- 单 Token 同步 -------------------------
func SyncDataByToken(token string) error {
	const pageSize = 50                  // 提高每页数量，减少请求次数
	const delay = 500 * time.Millisecond // 防抖，避免被限流

	// 第一次请求：获取总页数和第一页 ID
	firstPage, err := fetchOrdersPage(1, pageSize, token)
	if err != nil {
		return fmt.Errorf("第一次物流请求失败: %w", err)
	}

	var allIDs []string
	for _, rec := range firstPage.Result.Records {
		allIDs = append(allIDs, rec.ID)
	}

	// 获取剩余页
	for page := 2; page <= firstPage.Result.Pages; page++ {
		time.Sleep(delay) // 防抖
		pageData, err := fetchOrdersPage(page, pageSize, token)
		if err != nil {
			global.Log.Warnf("获取第 %d 页失败: %v", page, err)
			continue
		}
		for _, rec := range pageData.Result.Records {
			allIDs = append(allIDs, rec.ID)
		}
	}

	global.Log.Infof("Token %s 总共获取到 %d 个订单ID", token, len(allIDs))

	var orders []models.CesFbjOrder
	var cargos []models.CesFbjCargoInfo
	var packages []models.CesFbjPackage
	var commodities []models.CesFbjCommodityInfo

	// 获取每个订单详情
	for _, id := range allIDs {
		time.Sleep(100 * time.Millisecond) // 单订单请求也加点延迟
		detail, err := fetchOrderDetail(id, token)
		if err != nil {
			global.Log.Warnf("获取订单 %s 详情失败: %v", id, err)
			continue
		}
		var detailAddition *OrderDetailAdditionResp
		var gw float64
		var volWeight float64

		detailAddition, err = fetchOrderDetailAddition(id, token)
		if err != nil {
			global.Log.Warnf("获取订单 %s 详情补充失败: %v，将使用默认值 0, hbl:%s", id, err, detail.Result.CesFbjOrder.HBL)
			// ❌ 不要 continue！继续处理主订单
		} else {
			// 成功才赋真实值
			gw = detailAddition.Result.ERPData.GW
			volWeight = detailAddition.Result.ERPData.VolWeight
		}
		// 直接从 detail 和 detailAddition 中提取你需要的数据
		orderDetail := detail.Result
		order := detail.Result.CesFbjOrder
		order.TotalRoughWeightStorage = gw
		order.TotalCBMStorage = volWeight
		orders = append(orders, order)

		for _, cargo := range orderDetail.CesFbjCargoInfoList {
			cargo.OrderID = orderDetail.ID
			cargos = append(cargos, cargo.CesFbjCargoInfo)

			for _, pkg := range cargo.CesFbjPackageList {
				pkg.CargoID = cargo.ID
				packages = append(packages, pkg.CesFbjPackage)

				for _, commodity := range pkg.CesFbjCommodityInfoList {
					commodity.PackageID = pkg.ID
					commodity.CreateTime = orderDetail.CesFbjOrder.CreateTime
					commodity.CreateBy = orderDetail.CesFbjOrder.CreateBy
					commodity.UpdateTime = orderDetail.CesFbjOrder.UpdateTime
					commodity.UpdateBy = orderDetail.CesFbjOrder.UpdateBy
					commodity.HBL = orderDetail.CesFbjOrder.HBL
					commodities = append(commodities, commodity)
				}
			}
		}
	}

	// 批量保存（分批）
	if err := BatchSaveData(orders, cargos, packages, commodities); err != nil {
		return fmt.Errorf("批量保存订单数据失败: %w", err)
	}

	global.Log.Infof("Token %s 批量保存完成: 订单 %d, 货物 %d, 包裹 %d, 商品 %d",
		token, len(orders), len(cargos), len(packages), len(commodities))

	return nil
}

// ------------------------- 批量保存（分批插入）-------------------------
const BatchChunkSize = 200

// ChunkSlice 泛型分块函数
func ChunkSlice[T any](slice []T, size int) [][]T {
	var chunks [][]T
	for i := 0; i < len(slice); i += size {
		end := i + size
		if end > len(slice) {
			end = len(slice)
		}
		chunks = append(chunks, slice[i:end])
	}
	return chunks
}

func BatchSaveData(orders []models.CesFbjOrder, cargos []models.CesFbjCargoInfo, packages []models.CesFbjPackage, commodities []models.CesFbjCommodityInfo) error {
	db := global.DB
	return db.Transaction(func(tx *gorm.DB) error {
		var err error

		// 1. 订单
		if len(orders) > 0 {
			for _, chunk := range ChunkSlice(orders, BatchChunkSize) {
				if err = tx.Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "id"}},
					DoUpdates: clause.AssignmentColumns(getOrderUpdateColumns()),
				}).Create(&chunk).Error; err != nil {
					return fmt.Errorf("订单分批保存失败: %w", err)
				}
			}
		}

		// 2. 货物
		if len(cargos) > 0 {
			for _, chunk := range ChunkSlice(cargos, BatchChunkSize) {
				if err = tx.Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "id"}},
					DoUpdates: clause.AssignmentColumns(getCargoUpdateColumns()),
				}).Create(&chunk).Error; err != nil {
					return fmt.Errorf("货物分批保存失败: %w", err)
				}
			}
		}

		// 3. 包裹
		if len(packages) > 0 {
			for _, chunk := range ChunkSlice(packages, BatchChunkSize) {
				if err = tx.Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "id"}},
					DoUpdates: clause.AssignmentColumns(getPackageUpdateColumns()),
				}).Create(&chunk).Error; err != nil {
					return fmt.Errorf("包裹分批保存失败: %w", err)
				}
			}
		}

		// 4. 商品
		if len(commodities) > 0 {
			for _, chunk := range ChunkSlice(commodities, BatchChunkSize) {
				if err = tx.Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "id"}},
					DoUpdates: clause.AssignmentColumns(getCommodityUpdateColumns()),
				}).Create(&chunk).Error; err != nil {
					return fmt.Errorf("商品分批保存失败: %w", err)
				}
			}
		}

		return nil
	})
}

// ------------------------- 请求逻辑 -------------------------
type OrdersListResp struct {
	Code   int `json:"code"`
	Result struct {
		Records []struct {
			ID string `json:"id"`
		} `json:"records"`
		Total   int `json:"total"`
		Size    int `json:"size"`
		Current int `json:"current"`
		Pages   int `json:"pages"`
	} `json:"result"`
}

type OrderDetailResp struct {
	Code   int               `json:"code"`
	Result CesFbjOrderDetail `json:"result"`
}

func fetchOrdersPage(pageNo, pageSize int, token string) (*OrdersListResp, error) {
	timestamp := time.Now().Unix()
	url := fmt.Sprintf(
		"https://ka.choicexp.com/api/cesFbjOrders/list?_t=%d&column=createTime&order=desc&field=id&pageNo=%d&pageSize=%d",
		timestamp, pageNo, pageSize,
	)

	req, _ := http.NewRequest("GET", url, nil)
	setCommonHeaders(req, token)

	resp, err := (&http.Client{Timeout: 180 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取订单列表响应失败: %w", err)
	}

	var data OrdersListResp
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("解析订单列表失败: %w, 响应: %s", err, string(body))
	}

	if data.Code != 200 {
		return nil, fmt.Errorf("API 请求失败, code=%d", data.Code)
	}

	return &data, nil
}

func fetchOrderDetail(id, token string) (*OrderDetailResp, error) {
	timestamp := time.Now().Unix()
	url := fmt.Sprintf(
		"https://ka.choicexp.com/api/cesFbjOrders/queryDetailsById?_t=%d&id=%s",
		timestamp, id,
	)

	req, _ := http.NewRequest("GET", url, nil)
	setCommonHeaders(req, token)

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取订单详情响应失败: %w", err)
	}

	var detail OrderDetailResp
	if err := json.Unmarshal(body, &detail); err != nil {
		return nil, fmt.Errorf("解析订单详情失败: %w, 响应: %s", err, string(body))
	}

	if detail.Code != 200 {
		return nil, fmt.Errorf("详情API请求失败, code=%d", detail.Code)
	}

	return &detail, nil
}

// 请求响应结构体
type OrderDetailAdditionResp struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message"`
	Code      int         `json:"code"`
	Result    OrderResult `json:"result"`
	Timestamp int64       `json:"timestamp"`
}

type OrderResult struct {
	FBJData ERPData `json:"FBJ_DATA"` // 注意：虽然是 FBJ，但结构和 ERP 类似
	ERPData ERPData `json:"ERP_DATA"`
}

type ERPData struct {
	GW        float64    `json:"GW"`
	VolWeight float64    `json:"VOL_WEIGHT"` // 注意：JSON 中是大写下划线，Go 中转为 CamelCase
	Status    string     `json:"STATUS"`
	ReceiptID string     `json:"RECEIPT_ID"`
	Volume    float64    `json:"VOLUME"`
	Count     int        `json:"COUNT"`
	AirHBL    string     `json:"AIR_HBL"`
	SizeList  []SizeItem `json:"SIZE_LIST"`
}

type SizeItem struct {
	High        float64 `json:"HIGH"`
	VolWeight   float64 `json:"VOL_WEIGHT"`
	Num         string  `json:"NUM"`
	Length      float64 `json:"LENGTH"`
	Piece       int     `json:"PIECE"`
	Width       float64 `json:"WIDTH"`
	Weight      float64 `json:"WEIGHT"`
	Volumn      float64 `json:"VOLUMN"`
	TotalVolumn float64 `json:"TOTAL_VOLUMN,omitempty"` // FBJ_DATA 专用
	TotalWeight float64 `json:"TOTAL_WEIGHT,omitempty"` // FBJ_DATA 专用
}

func fetchOrderDetailAddition(id, token string) (*OrderDetailAdditionResp, error) {
	timestamp := time.Now().Unix()
	url := fmt.Sprintf(
		"https://ka.choicexp.com/api/cesFbjOrders/queryOrderDateById?_t=%d&id=%s",
		timestamp, id,
	)

	req, _ := http.NewRequest("GET", url, nil)
	setCommonHeaders(req, token)

	resp, err := (&http.Client{Timeout: 180 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取订单详情响应失败: %w", err)
	}

	var detailAddition OrderDetailAdditionResp
	if err := json.Unmarshal(body, &detailAddition); err != nil {
		return nil, fmt.Errorf("解析订单详情失败: %w, 响应: %s", err, string(body))
	}

	if detailAddition.Code != 200 {
		return nil, fmt.Errorf("详情API请求失败, code=%d", detailAddition.Code)
	}

	return &detailAddition, nil
}
func setCommonHeaders(req *http.Request, token string) {
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Referer", "https://ka.choicexp.com/bookingManage/booking/CesFbxOrdersList")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
	req.Header.Set("X-Access-Token", token)
	req.Header.Set("lang-type", "zh-CN")
	req.Header.Set("tenant_id", "0")
}

// ------------------------- 数据结构 -------------------------
type CesFbjOrderDetail struct {
	models.CesFbjOrder
	CesFbjCargoInfoList []CesFbjCargoInfoWithPackage `json:"cesFbjCargoInfoList"`
}

type CesFbjCargoInfoWithPackage struct {
	models.CesFbjCargoInfo
	CesFbjPackageList []CesFbjPackageWithCommodity `json:"cesFbjPackageList"`
}

type CesFbjPackageWithCommodity struct {
	models.CesFbjPackage
	CesFbjCommodityInfoList []models.CesFbjCommodityInfo `json:"cesFbjCommodityInfoList"`
}

// ------------------------- 更新字段列表 -------------------------
func getOrderUpdateColumns() []string {
	return []string{
		"create_by", "create_time", "update_by", "update_time", "hbl", "type", "type_name",
		"channel_id", "channel_code", "channel_name", "wh_id", "wh_code", "wh_name", "wh_address",
		"picking_type", "choice_wh_id", "choice_wh_contact", "choice_wh_phone", "choice_wh_address",
		"declare_service", "service_type", "cust_email", "expe_storage_time", "real_storage_time",
		"dispatch_documents", "order_tracking", "receipt_id", "status", "status_name",
		"total_count", "total_net_weight", "total_rough_weight", "total_cbm", "total_commodity_count",
		"total_rough_weight_storage", "total_cbm_storage",
	}
}

func getCargoUpdateColumns() []string {
	return []string{"create_by", "create_time", "shop_id", "seller_id", "shop_name", "po", "amends", "order_id"}
}

func getPackageUpdateColumns() []string {
	return []string{
		"create_by", "create_time", "box_no", "length", "high", "width", "rough_weight",
		"total_rough_weight", "net_weight", "total_net_weight", "count", "total_commodity_count",
		"cargo_id", "value",
	}
}

func getCommodityUpdateColumns() []string {
	return []string{
		"commodity_id", "commodity_name", "commodity_sku", "shop_sku", "commodity_cname",
		"commodity_ename", "commodity_attribute", "commodity_attribute_name", "commodity_count",
		"package_id", "brand_name", "is_brand",
		"create_by", "create_time", "update_by", "update_time", "hbl",
	}
}
