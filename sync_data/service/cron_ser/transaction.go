package cron_ser

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync_data/global"
	"sync_data/models"
	"sync_data/service/read_transaction_csv"
	"time"
)

// SyncJumiaTransactions 定时同步 Jumia 交易记录
func SyncJumiaTransactions() {
	filePath, err := GetCSVTransactions(global.Config.Jumia.ShopSid, global.Config.Jumia.JumiaCenterToken)
	if err != nil {
		global.Log.Errorf("同步 Jumia 交易记录失败: %v", err)
		global.DB.Where("id = ?", 6).Updates(models.ServiceStatus{
			Status:  "错误",
			Message: err.Error(),
		})
		return
	}

	global.Log.Infof("读取的文件地址为：%s", filePath)
	err = read_transaction_csv.ReadCSVTransactionToMysql(filePath)
	if err != nil {
		global.Log.Errorf("导入数据库失败: %v", err)
		global.DB.Where("id = ?", 6).Updates(models.ServiceStatus{
			Status:  "错误",
			Message: "导入数据库失败: " + err.Error(),
		})
		return
	}
	// ✅ 文件处理成功后删除文件
	if err := os.Remove(filePath); err != nil {
		global.Log.Warnf("删除临时文件失败: %s: %v", filePath, err)
		// 注意：这里用 Warnf，不 return，因为删除失败不应影响主流程
	} else {
		global.Log.Infof("✅ 临时文件已删除: %s", filePath)
	}

	filePath, err = GetCSVTransactions(global.Config.JumiaTwo.ShopSid, global.Config.JumiaTwo.JumiaCenterToken)
	if err != nil {
		global.Log.Errorf("同步 Jumia 交易记录失败: %v", err)
		global.DB.Where("id = ?", 6).Updates(models.ServiceStatus{
			Status:  "错误",
			Message: err.Error(),
		})
		return
	}

	global.Log.Infof("读取的文件地址为：%s", filePath)
	err = read_transaction_csv.ReadCSVTransactionToMysql(filePath)
	if err != nil {
		global.Log.Errorf("导入数据库失败: %v", err)
		global.DB.Where("id = ?", 6).Updates(models.ServiceStatus{
			Status:  "错误",
			Message: "导入数据库失败: " + err.Error(),
		})
		return
	}
	// ✅ 文件处理成功后删除文件
	if err := os.Remove(filePath); err != nil {
		global.Log.Warnf("删除临时文件失败: %s: %v", filePath, err)
		// 注意：这里用 Warnf，不 return，因为删除失败不应影响主流程
	} else {
		global.Log.Infof("✅ 临时文件已删除: %s", filePath)
	}
	// 更新成功状态
	global.DB.Where("id = ?", 6).Updates(models.ServiceStatus{
		Status:  "正常",
		Message: "定时同步交易记录成功",
	})
}

const (
	firstURL  = "https://api-vcs-services.jumia.com/api/transactions/exports"
	secondURL = "https://api-vcs-services.jumia.com/api/transactions/exports" // 移除分页参数，动态构建
)

// ExportRequest 创建导出任务的请求体
type ExportRequest struct {
	ShopSid                string   `json:"shopSid"`
	AccountStatementStatus []string `json:"accountStatementStatus"`
	StartAt                string   `json:"startAt"`
	EndAt                  string   `json:"endAt"`
	CountryCode            []string `json:"countryCode"`
}

// ExportResponse 导出任务响应
type ExportResponse struct {
	Sid          string `json:"sid"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
	ExportStatus string `json:"exportStatus"`
	Url          string `json:"url"`
	Type         string `json:"type"`
	Status       string `json:"status"` // completed, processing 等
}

// ExportListResponse 分页响应
type ExportListResponse struct {
	Content       []ExportResponse `json:"content"`
	TotalPages    int              `json:"totalPages"`
	TotalElements int              `json:"totalElements"`
	Number        int              `json:"number"`
	Last          bool             `json:"last"`
}

// getLast7Days 获取最近8天的起止日期
func getLast7Days() (start, end string) {
	now := time.Now()
	end = now.Format("2006-01-02")
	//todo 时间范围待严格筛选
	//由于之前的交易数据的状态可能更新，因此采取长时间范围更新交易数据，避免某个结算周期内的交易都是未结算
	start = now.AddDate(0, 0, -30).Format("2006-01-02")
	return
}

// GetCSVTransactions 创建导出任务、轮询状态、下载文件
func GetCSVTransactions(shopid, token string) (string, error) {
	client := &http.Client{}
	startAt, endAt := getLast7Days()
	//startAt := "2025-08-01"
	//endAt := "2025-08-16"

	// Step 1: 创建导出任务
	reqBody := ExportRequest{
		ShopSid:                shopid,
		AccountStatementStatus: []string{"PAID", "UNPAID"},
		StartAt:                startAt,
		EndAt:                  endAt,
		CountryCode:            []string{"GH", "KE", "NG"},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		global.Log.Errorf("JSON序列化失败: %v", err)
		return "", fmt.Errorf("请求数据序列化失败")
	}

	firstReq, err := http.NewRequest("POST", firstURL, bytes.NewBuffer(jsonData))
	if err != nil {
		global.Log.Errorf("创建请求失败: %v", err)
		return "", fmt.Errorf("创建请求失败")
	}
	setHeaders(firstReq, token)

	firstResp, err := client.Do(firstReq)
	if err != nil {
		global.Log.Errorf("请求发送失败: %v", err)
		return "", fmt.Errorf("网络请求失败")
	}
	defer firstResp.Body.Close()

	if firstResp.StatusCode != http.StatusOK &&
		firstResp.StatusCode != http.StatusCreated &&
		firstResp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(firstResp.Body)
		msg := fmt.Sprintf("创建任务失败: %d %s, 响应: %s", firstResp.StatusCode, firstResp.Status, string(body))
		global.Log.Error(msg)
		return "", fmt.Errorf(msg)
	}

	var firstResult ExportResponse
	if err := json.NewDecoder(firstResp.Body).Decode(&firstResult); err != nil {
		global.Log.Errorf("解析响应失败: %v", err)
		return "", fmt.Errorf("响应解析失败")
	}

	global.Log.Infof("导出任务创建成功, SID: %s, 状态: %s", firstResult.Sid, firstResult.ExportStatus)
	targetSID := firstResult.Sid

	// Step 2: 轮询状态（使用 context 控制超时）
	downloadURL, err := pollForCompletion(shopid, context.Background(), client, targetSID, token)
	if err != nil {
		return "", err
	}

	// Step 3: 下载文件
	downloadDir := "./downloads/transaction" // 默认路径
	filename := fmt.Sprintf("%s.csv", targetSID)
	destPath := filepath.Join(downloadDir, filename)

	filePath, err := downloadFile(downloadURL, destPath)
	if err != nil {
		global.Log.Errorf("文件下载失败: %v", err)
		return "", fmt.Errorf("下载失败: %w", err)
	}

	global.Log.Infof("🎉 文件下载成功: %s, 路径: %s", filename, filePath)
	return filePath, nil
}

// pollForCompletion 轮询直到任务完成或超时
func pollForCompletion(shopid string, ctx context.Context, client *http.Client, sid, token string) (string, error) {
	const maxWait = 10 * time.Minute
	const interval = 15 * time.Second

	ctx, cancel := context.WithTimeout(ctx, maxWait)
	defer cancel()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("轮询超时: 任务未在 %v 内完成", maxWait)
		case <-ticker.C:
			url, err := queryExportStatus(shopid, client, sid, token)
			if err != nil {
				global.Log.Warnf("轮询查询失败: %v", err)
				continue
			}
			if url != "" {
				return url, nil
			}
			global.Log.Debugf("任务 %s 仍在处理中...", sid)
		}
	}
}

// queryExportStatus 查询导出状态（支持分页）
// queryExportStatus 查询导出状态：遍历分页内容，检查 URL 是否包含 targetSID 且状态为 completed
func queryExportStatus(shopid string, client *http.Client, targetSID, token string) (string, error) {
	page := 0
	for {
		// 构建分页请求 URL
		url := fmt.Sprintf("%s?shopSid=%s&page=%d&size=20", secondURL, shopid, page)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return "", err
		}
		setHeaders(req, token)

		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close() // ✅ 及时关闭

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("查询失败: %d %s, 响应: %s", resp.StatusCode, resp.Status, string(body))
		}

		var result ExportListResponse
		if err := json.Unmarshal(body, &result); err != nil {
			return "", err
		}

		// 遍历内容，查找符合条件的任务
		for _, item := range result.Content {
			// ✅ 条件：状态 completed 且 URL 包含 targetSID
			if item.Status == "completed" && strings.Contains(item.Url, targetSID) {
				if item.Url == "" {
					return "", fmt.Errorf("任务已完成但下载链接为空 (targetSID=%s)", targetSID)
				}
				return item.Url, nil
			}
		}

		// 如果已经是最后一页，退出循环
		if result.Last {
			break
		}
		page++
	}

	// 没找到匹配的任务
	return "", nil
}

// setHeaders 设置通用请求头
func setHeaders(req *http.Request, token string) {
	req.Header.Set("accept", "application/json, text/plain, */*")
	req.Header.Set("accept-language", "en")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("content-type", "application/json")
	req.Header.Set("origin", "https://vendorcenter.jumia.com")
	req.Header.Set("priority", "u=1, i")
	req.Header.Set("referer", "https://vendorcenter.jumia.com/")
	req.Header.Set("sec-ch-ua", `"Not)A;Brand";v="8", "Chromium";v="138", "Google Chrome";v="138"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"Windows"`)
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "same-site")
	req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36")
}

// downloadFile 下载文件并返回路径
func downloadFile(url, destPath string) (string, error) {
	// 创建目录
	if err := os.MkdirAll(filepath.Dir(destPath), os.ModePerm); err != nil {
		return "", fmt.Errorf("创建目录失败: %w", err)
	}

	// 下载
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载失败: %d %s", resp.StatusCode, resp.Status)
	}

	// 保存文件
	out, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("创建文件失败: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", fmt.Errorf("文件写入失败: %w", err)
	}

	return destPath, nil
}
