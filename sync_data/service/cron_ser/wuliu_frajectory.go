package cron_ser

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"gorm.io/gorm/clause"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"sync_data/global"
	"sync_data/models"
	"time"

	"gorm.io/gorm"
)

func SyncAllAccountsOrdersTrajectory() {
	//SyncWuliuToken()
	// 清空物流轨迹表，不然重复数据
	if err := global.DB.Exec("TRUNCATE TABLE ces_fbj_order_trajectories").Error; err != nil {
		global.Log.Errorf("清空 ces_fbj_order_trajectories 表失败: %v", err)
		global.DB.Where("id = ?", 10).Updates(models.ServiceStatus{
			Status:  "错误",
			Message: err.Error(),
		})
		return
	}
	time.Sleep(1 * time.Second)
	var orders_juzhi []models.CesFbjOrder
	global.DB.Where("create_by = ?", "湖南聚智跨境电子商务有限公司").Find(&orders_juzhi)
	err := SyncAllOrdersTrajectory(orders_juzhi, "SEA2025041745", "U0VBMjAyNTA0MTc0NQ==")
	if err != nil {
		global.Log.Error(err.Error())
		global.DB.Where("id = ?", 10).Updates(models.ServiceStatus{
			Status:  "错误",
			Message: err.Error(),
		})
		return
	}
	global.Log.Info("同步-聚智-物流轨迹数据成功")
	time.Sleep(1 * time.Second)
	var orders_yunchi []models.CesFbjOrder
	global.DB.Where("create_by = ?", "湖南云驰跨境电子商务有限公司").Find(&orders_yunchi)
	err = SyncAllOrdersTrajectory(orders_yunchi, "SEA2024100104", "U0VBMjAyNDEwMDEwNA==")
	if err != nil {
		global.Log.Error(err.Error())
		global.DB.Where("id = ?", 10).Updates(models.ServiceStatus{
			Status:  "错误",
			Message: err.Error(),
		})
		return
	}
	global.Log.Info("同步-云驰-物流轨迹数据成功")
	time.Sleep(1 * time.Second)
	var orders_yunchi2 []models.CesFbjOrder
	global.DB.Where("create_by = ?", "湖南聚智跨境电子商务有限公司陈彰武").Find(&orders_yunchi2)
	err = SyncAllOrdersTrajectory(orders_yunchi2, "SEA2025010142", "U0VBMjAyNTAxMDE0Mg==")
	if err != nil {
		global.Log.Error(err.Error())
		global.DB.Where("id = ?", 10).Updates(models.ServiceStatus{
			Status:  "错误",
			Message: err.Error(),
		})
		return
	}
	global.Log.Info("同步-聚智陈彰武-物流轨迹数据成功")
	global.DB.Where("id = ?", 10).Updates(models.ServiceStatus{
		Status:  "正常",
		Message: "定时更新物流轨迹成功",
	})
}

// ------------------------- 高性能同步物流轨迹 -------------------------
func SyncAllOrdersTrajectory(orders []models.CesFbjOrder, appKey, appSecret string) error {
	var trajectories []models.CesFbjOrderTrajectory
	var mu sync.Mutex // 保护 trajectories

	var allErrors []error
	var errMu sync.Mutex // 保护 allErrors

	var wg sync.WaitGroup
	sem := make(chan struct{}, 10) // 并发限制：最多10个并发

	// 如果没有订单，直接返回
	if len(orders) == 0 {
		return nil
	}

	for _, order := range orders {
		wg.Add(1)
		sem <- struct{}{} // 获取信号量

		go func(order models.CesFbjOrder) {
			defer wg.Done()
			defer func() { <-sem }() // 释放信号量

			so := order.HBL
			timestamp := time.Now().Unix()
			sign := GenerateSign(appKey, appSecret, timestamp)
			if so == "" {
				global.Log.Info("由于订单没有so，无法查询物流轨迹")
				return
			}
			// 1. 调用物流接口
			respStr, err := PostWuliu(appKey, sign, fmt.Sprintf("%d", timestamp), so)
			if err != nil {
				errMsg := fmt.Errorf("查询物流轨迹失败: HBL=%s, 订单ID=%s, 错误=%v", so, order.ID, err)
				global.Log.Warn(errMsg.Error())
				errMu.Lock()
				allErrors = append(allErrors, errMsg)
				errMu.Unlock()
				return
			}

			// 2. 解析 JSON 响应
			var resp WuliuTrackResp
			if err := json.Unmarshal([]byte(respStr), &resp); err != nil {
				errMsg := fmt.Errorf("解析物流轨迹响应失败: HBL=%s, 订单ID=%s, 错误=%v, 响应=%s", so, order.ID, err, truncateString(respStr, 200))
				global.Log.Warn(errMsg.Error())
				global.Log.Warn(so)
				errMu.Lock()
				allErrors = append(allErrors, errMsg)
				errMu.Unlock()
				return
			}

			// 3. 检查结果中是否有该 SO
			items, ok := resp.Result[so]
			if !ok {
				//errMsg := fmt.Errorf("物流返回数据中未找到 SO: %s, 订单ID=%s", so, order.ID)
				global.Log.Warn(respStr)
				global.Log.Warnf("物流返回数据中未找到 SO: %s, 订单ID=%s", so, order.ID)
				//global.Log.Warn(errMsg.Error())
				errMu.Lock()
				//allErrors = append(allErrors, errMsg)
				errMu.Unlock()
				return
			}

			// 4. 解析时间并构建轨迹数据
			localTrajectories := make([]models.CesFbjOrderTrajectory, 0, len(items))
			loc, _ := time.LoadLocation("Asia/Shanghai")

			for _, item := range items {
				oplink, timestamp := ParseTrajectoryTimestamp(item.OpLink, item.Timestamp, loc)
				// 直接按 Asia/Shanghai 解析
				localTrajectories = append(localTrajectories, models.CesFbjOrderTrajectory{
					OrderID:   order.ID,
					SO:        so,
					OpLink:    oplink,
					Timestamp: timestamp,
				})
			}

			// 5. 安全地添加到总结果
			mu.Lock()
			trajectories = append(trajectories, localTrajectories...)
			mu.Unlock()
		}(order)
	}

	// 等待所有 goroutine 完成
	wg.Wait()

	// === 处理错误汇总 ===
	if len(allErrors) > 0 {
		// 可选：记录所有错误
		for _, e := range allErrors {
			global.Log.Errorf("物流同步失败项: %v", e)
		}

		return fmt.Errorf("共 %d 个订单同步失败:\n  %v", len(allErrors), allErrors)
	}

	// === 保存成功的结果 ===
	if len(trajectories) == 0 {
		global.Log.Info("所有订单物流轨迹为空，无需保存")
		return nil
	}

	// 分批保存，防止 SQL 过长
	const batchSize = 100
	for i := 0; i < len(trajectories); i += batchSize {
		end := i + batchSize
		if end > len(trajectories) {
			end = len(trajectories)
		}
		if err := BatchSaveTrajectory(trajectories[i:end]); err != nil {
			return fmt.Errorf("批量保存轨迹失败 (批次 %d-%d): %v", i, end, err)
		}
	}

	global.Log.Infof("物流轨迹同步完成: 成功处理 %d 个订单，共保存 %d 条轨迹", len(orders), len(trajectories))
	return nil
}

// ------------------------- 批量保存轨迹 -------------------------
func BatchSaveTrajectory(trajectories []models.CesFbjOrderTrajectory) error {
	if len(trajectories) == 0 {
		return nil
	}
	db := global.DB
	return db.Transaction(func(tx *gorm.DB) error {
		return tx.Clauses(
			clause.OnConflict{
				Columns:   []clause.Column{{Name: "order_id"}, {Name: "so"}, {Name: "op_link"}},
				DoUpdates: clause.AssignmentColumns([]string{"timestamp", "updated_at"}),
			},
		).Create(&trajectories).Error
	})
}

// ------------------------- 请求结构体 -------------------------
type WuliuTrackResp struct {
	Success   bool                        `json:"success"`
	Message   string                      `json:"message"`
	Code      int                         `json:"code"`
	Result    map[string][]WuliuTrackItem `json:"result"`
	Timestamp int64                       `json:"timestamp"`
}

type WuliuTrackItem struct {
	OpLink    string `json:"opLink"`
	Timestamp string `json:"timestamp"`
}

// ------------------------- 生成签名 -------------------------
func GenerateSign(appKey, appSecret string, timestamp int64) string {
	step1 := "appKey" + appKey + "timestamp" + fmt.Sprintf("%d", timestamp)
	step2 := appSecret + step1 + appSecret
	hash := md5.Sum([]byte(step2))
	sign := strings.ToLower(fmt.Sprintf("%x", hash))
	return sign
}

// ------------------------- 请求物流轨迹 -------------------------
type WuliuRequest struct {
	SO string `json:"SO"`
}

func PostWuliu(appKey, sign, timestamp string, so string) (string, error) {
	url := "https://ka.choicexp.com/api/fbx/v1/tracks"
	requestBody := WuliuRequest{SO: so}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("JSON 序列化失败: %v", err)
	}

	req, err := http.NewRequest("POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		log.Printf("创建请求失败: %v", err)
		return "", err
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("appKey", appKey)
	req.Header.Set("timestamp", timestamp)
	req.Header.Set("sign", sign)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("请求失败: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("读取响应失败: %v", err)
		return "", err
	}

	return string(body), nil
}

// ParseTrajectoryTimestamp 统一解析轨迹时间
func ParseTrajectoryTimestamp(opLink, rawTs string, loc *time.Location) (string, *time.Time) {
	// 提取时间字符串（支持带时分秒、带时分、仅日期）
	re := regexp.MustCompile(`\d{4}-\d{2}-\d{2}(?: \d{2}:\d{2}(?::\d{2})?)?`)
	timeStr := re.FindString(rawTs)

	// 如果有时间，去掉它作为备注
	remark := strings.TrimSpace(strings.Replace(rawTs, timeStr, "", 1))

	// 拼接 opLink
	finalOpLink := opLink
	if remark != "" {
		finalOpLink = fmt.Sprintf("%s，%s", opLink, remark)
	}

	// 尝试解析时间
	var ts *time.Time
	if timeStr != "" {
		formats := []string{
			"2006-01-02 15:04:05",
			"2006-01-02 15:04",
			"2006-01-02",
		}
		for _, f := range formats {
			if t, err := time.ParseInLocation(f, timeStr, loc); err == nil {
				ts = &t
				break
			}
		}
	}

	return finalOpLink, ts
}
