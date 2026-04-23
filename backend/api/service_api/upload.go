package service_api

import (
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"backend/global"
	"backend/models/res"
	"backend/service/fill_customize_order"
	"backend/service/read_customize_order"
	"github.com/gin-gonic/gin"
)

// UploadRequest: 保留 form 和 json tag
type UploadRequest struct {
	Type string `form:"type" json:"type" binding:"required,oneof=gh ke ng"` // 保留 oneof，确保只有预定义类型
}

// SyncTaskStatus 定义异步同步任务的状态
type SyncTaskStatus struct {
	ID                string    `json:"id"`
	Type              string    `json:"type"`                 // 记录任务对应的文件类型
	Status            string    `json:"status"`               // "pending", "running", "completed", "failed"
	Progress          float64   `json:"progress"`             // 0.0 - 100.0
	Total             int       `json:"total"`                // 期望处理/更新的总记录数 (通常等于导入的行数)
	PersonUpdated     int       `json:"person_updated"`       // 实际成功处理/更新的记录数
	StatusUpdated     int       `json:"status_updated"`       // 实际成功处理/更新的记录数
	OrderNumberIsNull int       `json:"order_number_is_null"` // 实际成功处理/更新的记录数
	ErrorMessage      string    `json:"errorMessage,omitempty"`
	StartTime         time.Time `json:"startTime"`
	EndTime           time.Time `json:"endTime,omitempty"`
	// 可以根据需要添加更多字段
}

// SyncResult 用于 processXXXTask 函数返回处理结果
type SyncResult struct {
	ExpectedTotal       int // 期望处理的总条数 (例如，从CSV导入的条数)
	ActualPersonUpdated int // 实际成功更新的条数
	ActualStatusUpdated int // 实际成功更新的条数
	OrderNumberIsNull   int // 实际成功更新的条数
	Error               error
}

// 全局变量：存储所有同步任务的状态
// 注意：生产环境应使用 Redis 等持久化存储，此处使用内存 map 仅作演示
var (
	taskMutex sync.RWMutex
	tasks     = make(map[string]*SyncTaskStatus)
)

// generateTaskID 生成唯一的任务ID
func generateTaskID() string {
	return fmt.Sprintf("sync_%d_%d", time.Now().Unix(), os.Getpid())
}

// processGHTask 处理 type=gh 的具体任务逻辑
// 返回 SyncResult
func processGHTask(taskID, savePath string) SyncResult {
	result := SyncResult{}

	// 1. 执行耗时的数据库导入操作
	totalImported, err := read_customize_order.ImportGHCSVToMySQL(savePath)
	if err != nil {
		result.Error = fmt.Errorf("导入加纳CSV数据到数据库失败: %w", err)
		return result
	}
	result.ExpectedTotal = int(totalImported) // 期望更新的总数通常等于导入的总数
	global.Log.Infof("任务 [%s] 成功导入 %d 条加纳数据", taskID, totalImported)

	// 2. 执行耗时的同步更新字段操作
	fillResult, err := fill_customize_order.FillAllCustomizeOrderGH(100) // batchSize 可配置
	if err != nil {
		result.Error = fmt.Errorf("同步更新加纳订单字段失败: %w", err)
		// 即使同步过程出错，也记录下实际成功更新的数量
		result.ActualPersonUpdated = fillResult.UpdatedPerson
		result.ActualStatusUpdated = fillResult.UpdatedStatus
		result.OrderNumberIsNull = fillResult.EmptyOrderNumber
		return result
	}
	result.ActualPersonUpdated = fillResult.UpdatedPerson
	result.ActualStatusUpdated = fillResult.UpdatedStatus
	result.OrderNumberIsNull = fillResult.EmptyOrderNumber
	global.Log.Infof("任务 [%s] 成功同步更新 %d 条加纳订单字段", taskID, fillResult.TotalUpdated)

	return result
}

// processKETask 处理 type=ke 的具体任务逻辑 (占位符)
// 返回 SyncResult
func processKETask(taskID, savePath string) SyncResult {
	result := SyncResult{}

	// 1. 执行耗时的数据库导入操作
	totalImported, err := read_customize_order.ImportKECSVToMySQL(savePath)
	if err != nil {
		result.Error = fmt.Errorf("导入肯尼亚CSV数据到数据库失败: %w", err)
		return result
	}
	result.ExpectedTotal = int(totalImported) // 期望更新的总数通常等于导入的总数
	global.Log.Infof("任务 [%s] 成功导入 %d 条肯尼亚数据", taskID, totalImported)

	// 2. 执行耗时的同步更新字段操作
	fillResult, err := fill_customize_order.FillAllCustomizeOrderKE(100) // batchSize 可配置
	if err != nil {
		result.Error = fmt.Errorf("同步更新肯尼亚订单字段失败: %w", err)
		// 即使同步过程出错，也记录下实际成功更新的数量
		result.ActualPersonUpdated = fillResult.UpdatedPerson
		result.ActualStatusUpdated = fillResult.UpdatedStatus
		result.OrderNumberIsNull = fillResult.EmptyOrderNumber
		return result
	}
	result.ActualPersonUpdated = fillResult.UpdatedPerson
	result.ActualStatusUpdated = fillResult.UpdatedStatus
	result.OrderNumberIsNull = fillResult.EmptyOrderNumber
	global.Log.Infof("任务 [%s] 成功同步更新 %d 条肯尼亚订单字段", taskID, fillResult.TotalUpdated)
	return result
}

// processNGTask 处理 type=ng 的具体任务逻辑 (占位符)
// 返回 SyncResult
func processNGTask(taskID, savePath string) SyncResult {
	result := SyncResult{}

	// 1. 执行耗时的数据库导入操作
	totalImported, err := read_customize_order.ImportNGCSVToMySQL(savePath)
	if err != nil {
		result.Error = fmt.Errorf("导入尼日利亚CSV数据到数据库失败: %w", err)
		return result
	}
	result.ExpectedTotal = int(totalImported) // 期望更新的总数通常等于导入的总数
	global.Log.Infof("任务 [%s] 成功导入 %d 条尼日利亚数据", taskID, totalImported)

	// 2. 执行耗时的同步更新字段操作
	fillResult, err := fill_customize_order.FillAllCustomizeOrderNG(100) // batchSize 可配置
	if err != nil {
		result.Error = fmt.Errorf("同步更新尼日利亚订单字段失败: %w", err)
		// 即使同步过程出错，也记录下实际成功更新的数量
		result.ActualPersonUpdated = fillResult.UpdatedPerson
		result.ActualStatusUpdated = fillResult.UpdatedStatus
		result.OrderNumberIsNull = fillResult.EmptyOrderNumber
		return result
	}
	result.ActualPersonUpdated = fillResult.UpdatedPerson
	result.ActualStatusUpdated = fillResult.UpdatedStatus
	result.OrderNumberIsNull = fillResult.EmptyOrderNumber
	global.Log.Infof("任务 [%s] 成功同步更新 %d 条尼日利亚订单字段", taskID, fillResult.TotalUpdated)
	return result
}

// (ServiceApi) UploadCSV 处理CSV文件上传并启动后台同步任务
func (ServiceApi) UploadCSV(c *gin.Context) {
	var req UploadRequest

	// 1. 使用 ShouldBind 绑定 form 数据 (包括 type 字段)
	if err := c.ShouldBind(&req); err != nil {
		res.FailWithError(err, &req, c)
		return
	}

	// 2. 获取上传的文件流和文件头
	file, fileHeader, err := c.Request.FormFile("file")
	if err != nil {
		global.Log.Errorf("获取上传文件失败: %v", err)
		res.FailWithMessage("获取上传文件失败", c)
		return
	}
	defer file.Close()

	// 3. 验证文件类型
	if !isValidCSVFile(fileHeader, c) {
		return
	}

	// 4. 准备服务器端保存路径
	uploadDir := "./uploads/csv"
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		global.Log.Errorf("创建上传目录失败: %v", err)
		res.FailWithMessage("服务器内部错误: 创建目录失败", c)
		return
	}

	// 生成唯一文件名
	filename := fmt.Sprintf("upload_%d_%s", time.Now().UnixNano(), filepath.Base(fileHeader.Filename))
	savePath := filepath.Join(uploadDir, filename)

	// 5. 保存文件到服务器
	if err := c.SaveUploadedFile(fileHeader, savePath); err != nil {
		global.Log.Errorf("保存文件到服务器失败: %v", err)
		res.FailWithMessage("保存文件失败", c)
		return
	}
	global.Log.Infof("文件已成功保存到: %s", savePath)

	// 6. 生成唯一的任务ID
	taskID := generateTaskID()

	// 7. 创建任务状态对象 (初始状态为 pending)
	task := &SyncTaskStatus{
		ID:                taskID,
		Type:              req.Type, // 记录任务类型
		Status:            "pending",
		Progress:          0,
		Total:             0,
		PersonUpdated:     0,
		StatusUpdated:     0,
		OrderNumberIsNull: 0,
		StartTime:         time.Now(),
	}

	// 8. 将任务状态存入全局 map
	taskMutex.Lock()
	tasks[taskID] = task
	taskMutex.Unlock()

	// ********** 关键：启动异步 Goroutine 执行耗时操作 **********
	go func() {
		defer func() {
			if r := recover(); r != nil {
				taskMutex.Lock()
				if task, exists := tasks[taskID]; exists {
					task.Status = "failed"
					task.ErrorMessage = fmt.Sprintf("Panic: %v", r)
					task.EndTime = time.Now()
				}
				taskMutex.Unlock()
				global.Log.Errorf("Sync task %s panicked: %v", taskID, r)
			}
		}()

		taskMutex.Lock()
		task.Status = "running"
		taskMutex.Unlock()
		global.Log.Infof("异步任务 [%s] 开始执行 (类型: %s)", taskID, req.Type)

		var syncResult SyncResult
		var err error

		// --- 核心：根据 req.Type 调用不同的处理函数 ---
		switch req.Type {
		case "gh":
			syncResult = processGHTask(taskID, savePath)
			err = syncResult.Error
		case "ke":
			syncResult = processKETask(taskID, savePath)
			err = syncResult.Error
		case "ng":
			syncResult = processNGTask(taskID, savePath)
			err = syncResult.Error
		default:
			// 理论上不会走到这里，因为 binding:"oneof=..." 已经验证
			err = fmt.Errorf("不支持的文件类型: %s", req.Type)
		}
		// ----------------------------------------------------

		// 更新任务状态 (在同一个锁内完成最终状态更新和文件清理)
		taskMutex.Lock()
		defer taskMutex.Unlock()

		if err != nil {
			global.Log.Errorf("任务 [%s] 执行失败: %v", taskID, err)
			task.Status = "failed"
			task.ErrorMessage = err.Error()
		} else {
			global.Log.Infof("任务 [%s] 成功完成", taskID)
			task.Status = "completed"
			task.Progress = 100.0
		}
		// ✅ 精确设置：Total 为期望总数，Processed 为实际成功更新数
		task.Total = syncResult.ExpectedTotal
		task.PersonUpdated = syncResult.ActualPersonUpdated
		task.StatusUpdated = syncResult.ActualStatusUpdated
		task.OrderNumberIsNull = syncResult.OrderNumberIsNull
		task.EndTime = time.Now()

		// 11. 清理临时文件
		if rmErr := os.Remove(savePath); rmErr != nil {
			global.Log.Errorf("任务 [%s] 执行完成后删除临时文件失败: %s, 错误: %v", taskID, savePath, rmErr)
		} else {
			global.Log.Infof("任务 [%s] 已成功删除临时文件: %s", taskID, savePath)
		}
	}()
	// ***********************************************************

	// 返回响应
	res.Ok(
		gin.H{
			"message":   "文件上传和处理已开始，请在后台完成",
			"taskID":    taskID,
			"type":      req.Type, // 返回类型，方便前端知晓
			"statusURL": fmt.Sprintf("/api/service/status/%s", taskID),
			"file":      fileHeader.Filename,
		},
		"请求已接受",
		c,
	)
}

// (ServiceApi) GetSyncCSVStatus 查询指定同步任务的状态
func (ServiceApi) GetSyncCSVStatus(c *gin.Context) {
	taskID := c.Param("taskID")

	taskMutex.RLock()
	task, exists := tasks[taskID]
	taskMutex.RUnlock()

	if !exists {
		res.FailWithMessage("任务不存在或已过期", c)
		return
	}

	res.OkWithData(task, c)
}

// isValidCSVFile 检查上传的文件是否为 CSV 文件
func isValidCSVFile(fileHeader *multipart.FileHeader, c *gin.Context) bool {
	mimeType := fileHeader.Header.Get("Content-Type")
	if mimeType != "text/csv" && mimeType != "application/vnd.ms-excel" {
		global.Log.Warnf("不支持的文件 MIME 类型: %s, 文件名: %s", mimeType, fileHeader.Filename)
		res.FailWithMessage("仅支持上传 CSV 文件", c)
		return false
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext != ".csv" {
		global.Log.Warnf("不支持的文件扩展名: %s, 文件名: %s", ext, fileHeader.Filename)
		res.FailWithMessage("仅支持上传 .csv 扩展名的文件", c)
		return false
	}

	return true
}
