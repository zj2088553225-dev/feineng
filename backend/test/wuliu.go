package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// GenerateSign 生成签名 sign
func GenerateSign(appKey, appSecret string, timestamp int64) string {
	// 第一步：拼接 appKey 和 timestamp，使用 "xxx" 分隔
	step1 := "appKey" + appKey + "timestamp" + fmt.Sprintf("%d", timestamp)

	// 第二步：在前后添加 appSecret
	step2 := appSecret + step1 + appSecret

	// 第三步：MD5 加密
	fmt.Printf("%s\n", step2)
	hash := md5.Sum([]byte(step2))
	fmt.Printf("%x\n", hash)
	sign := strings.ToLower(fmt.Sprintf("%x", hash))

	return sign
}

// 刷新token和refresh_token
type WuliuRequest struct {
	SO string `json:"SO"`
}

func PostWuliu(appKey, sign, timestamp string, so string) (string, error) {
	url := "https://ka.choicexp.com/api/fbx/v1/tracks"
	// 3. 构造请求体：{"SO": "SNG004599"}
	requestBody := WuliuRequest{SO: so}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("JSON 序列化失败: %v", err)
	}

	// 📦 创建请求，使用 strings.NewReader 直接传字符串
	req, err := http.NewRequest("POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		log.Printf("创建请求失败: %v", err)
		return "", err
	}

	// 📝 设置请求头
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("appKey", appKey)
	req.Header.Set("timestamp", timestamp)
	req.Header.Set("sign", sign)

	// 🚀 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("请求失败: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	// 📥 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("读取响应失败: %v", err)
		return "", err
	}

	// 🔍 打印请求和响应（调试用）
	log.Printf("📤 发送 URL: %s", url)
	log.Printf("📤 发送 Body: %s", jsonData)
	log.Printf("📥 响应状态: %d", resp.StatusCode)
	log.Printf("📥 响应 Body: %s", string(body))

	return string(body), nil
}
func main() {
	appKey := "SEA2025041745"
	appSecret := "U0VBMjAyNTA0MTc0NQ=="
	timestamp := time.Now().Unix()

	//timestamp = 1753769833610
	sign := GenerateSign(appKey, appSecret, timestamp)
	fmt.Println("生成的签名 sign:", sign)
	fmt.Println("生成的签名 timestamp:", timestamp)
	//str_timestamp := fmt.Sprintf("%d", timestamp)
	//response, err := PostWuliu(appKey, sign, str_timestamp, "SNG004599")
	//if err != nil {
	//	fmt.Println(err)
	//}
	//fmt.Println(response)
}
