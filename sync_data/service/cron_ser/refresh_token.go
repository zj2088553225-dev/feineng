package cron_ser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync_data/core"
	"sync_data/global"
	"sync_data/models"
	"time"
)

func SyncJumiaApiToken() {
	global.Log.Info("定时同步Jumiaapitoken开始")

	// 串行执行，保证一个完成后再执行下一个
	PostRefreshToken()
	time.Sleep(10 * time.Second)
	PostRefreshTokenTwo()
}

// 刷新token和refresh_token
func PostRefreshToken() {

	// 目标URL
	apiURL := "https://vendor-api.jumia.com/token"

	// 构造表单数据
	formData := url.Values{}
	formData.Set("client_id", global.Config.Jumia.ClientId)
	formData.Set("client_secret", global.Config.Jumia.ClientSecret)
	formData.Set("grant_type", global.Config.Jumia.GrantType)
	formData.Set("redirect_url", global.Config.Jumia.RedirectUrl)
	formData.Set("refresh_token", global.Config.Jumia.RefreshToken)

	// 创建请求体
	requestBody := bytes.NewBufferString(formData.Encode())

	// 创建HTTP请求
	req, err := http.NewRequest("POST", apiURL, requestBody)
	if err != nil {
		global.DB.Where("id = ?", 1).Updates(models.ServiceStatus{
			Status:  "错误",
			Message: err.Error(),
			// UpdatedAt 会自动更新（如果是 time.Time 类型）
		})
		global.Log.Errorf(err.Error())
		return
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	// 发送请求
	client := &http.Client{
		Timeout: 180 * time.Second,
		Transport: &http.Transport{
			// 可选：限制最大连接数
			MaxIdleConnsPerHost: 5,
		},
	}

	req.Header.Set("User-Agent", "Jumia-Token-Refresher/1.0 (Account-1)")
	resp, err := client.Do(req)
	if err != nil {
		global.DB.Where("id = ?", 1).Updates(models.ServiceStatus{
			Status:  "错误",
			Message: err.Error(),
			// UpdatedAt 会自动更新（如果是 time.Time 类型）
		})
		global.Log.Errorf(err.Error())
		return
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	//global.Log.Infof("Jumia Token响应: status=%d, body=%s", resp.StatusCode, string(body))
	if err != nil {
		global.DB.Where("id = ?", 1).Updates(models.ServiceStatus{
			Status:  "错误",
			Message: err.Error(),
			// UpdatedAt 会自动更新（如果是 time.Time 类型）
		})
		global.Log.Errorf(err.Error())
		return
	}

	//fmt.Printf("状态码: %d\n响应数据: %s\n", resp.StatusCode, string(body))
	// 提取令牌
	accessToken, refreshToken, err := ExtractTokens(body)
	if err != nil {
		global.Log.Error(err.Error())
		global.DB.Where("id = ?", 1).Updates(models.ServiceStatus{
			Status:  "错误",
			Message: err.Error(),
		})
		global.Log.Errorf("token2令牌提取失败: %v", err)
		return
	}

	global.Config.Jumia.RefreshToken = refreshToken
	global.Config.Jumia.AccessToken = accessToken

	core.SetYaml()
	global.Log.Infoln("刷新token1成功")

	global.DB.Where("id = ?", 1).Updates(models.ServiceStatus{
		Status:  "正常",
		Message: fmt.Sprintf("定时同步jumia_api_token1成功"),
	})
}

// 刷新token和refresh_token
func PostRefreshTokenTwo() {

	// 目标URL
	apiURL := "https://vendor-api.jumia.com/token"

	// 构造表单数据
	formData := url.Values{}
	formData.Set("client_id", global.Config.JumiaTwo.ClientId)
	formData.Set("client_secret", global.Config.JumiaTwo.ClientSecret)
	formData.Set("grant_type", global.Config.JumiaTwo.GrantType)
	formData.Set("redirect_url", global.Config.JumiaTwo.RedirectUrl)
	formData.Set("refresh_token", global.Config.JumiaTwo.RefreshToken)

	// 创建请求体
	requestBody := bytes.NewBufferString(formData.Encode())

	// 创建HTTP请求
	req, err := http.NewRequest("POST", apiURL, requestBody)
	if err != nil {
		global.DB.Where("id = ?", 12).Updates(models.ServiceStatus{
			Status:  "错误",
			Message: err.Error(),
			// UpdatedAt 会自动更新（如果是 time.Time 类型）
		})
		global.Log.Errorf(err.Error())
		return
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	// 发送请求
	client := &http.Client{
		Timeout: 180 * time.Second,
		Transport: &http.Transport{
			// 可选：限制最大连接数
			MaxIdleConnsPerHost: 5,
		},
	}

	req.Header.Set("User-Agent", "Jumia-Token-Refresher/1.0 (Account-2)")
	resp, err := client.Do(req)
	if err != nil {
		global.DB.Where("id = ?", 12).Updates(models.ServiceStatus{
			Status:  "错误",
			Message: err.Error(),
			// UpdatedAt 会自动更新（如果是 time.Time 类型）
		})
		global.Log.Errorf(err.Error())
		return
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	global.Log.Infof("Jumia Token响应: status=%d, body=%s", resp.StatusCode, string(body))
	if err != nil {
		global.DB.Where("id = ?", 12).Updates(models.ServiceStatus{
			Status:  "错误",
			Message: err.Error(),
			// UpdatedAt 会自动更新（如果是 time.Time 类型）
		})
		global.Log.Errorf(err.Error())
		return
	}

	//fmt.Printf("状态码: %d\n响应数据: %s\n", resp.StatusCode, string(body))
	// 提取令牌
	accessToken, refreshToken, err := ExtractTokens(body)
	if err != nil {

		global.DB.Where("id = ?", 12).Updates(models.ServiceStatus{
			Status:  "错误",
			Message: err.Error(),
			// UpdatedAt 会自动更新（如果是 time.Time 类型）
		})
		//global.Log.Infof("DB update rows=%d, err=%v", result.RowsAffected, result.Error)
		global.Log.Errorf("token2令牌提取失败: %v", err)
		return
	}

	//fmt.Printf("Access Token: %s\nRefresh Token: %s\n",
	//	accessToken, refreshToken)
	global.Config.JumiaTwo.RefreshToken = refreshToken
	global.Config.JumiaTwo.AccessToken = accessToken

	core.SetYaml()
	global.Log.Infoln("刷新token2成功")
	global.DB.Where("id = ?", 12).Updates(models.ServiceStatus{
		Status:  "正常",
		Message: "定时同步jumia_api_token2成功",
	})

}

// OAuthResponse 定义完整的OAuth2.0响应结构
type OAuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	// 其他字段可根据需要添加
}

func ExtractTokens(jsonData []byte) (accessToken, refreshToken string, err error) {
	var resp OAuthResponse
	if err := json.Unmarshal(jsonData, &resp); err != nil {
		return "", "", fmt.Errorf("JSON解析失败: %w", err)
	}

	// 验证必要字段
	if resp.AccessToken == "" {
		return "", "", fmt.Errorf("access_token为空")
	}
	if resp.RefreshToken == "" {
		return "", "", fmt.Errorf("refresh_token为空")
	}
	global.Log.Info(resp.AccessToken)
	return resp.AccessToken, resp.RefreshToken, nil
}
