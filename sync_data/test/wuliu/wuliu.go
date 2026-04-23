package main

import (
	"crypto/md5"
	"fmt"
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

func main() {
	appKey := "SEA2024100104"
	appSecret := "U0VBMjAyNDEwMDEwNA=="
	timestamp := time.Now().Unix()

	//timestamp = 1753769833610
	sign := GenerateSign(appKey, appSecret, timestamp)
	fmt.Println("生成的签名 sign:", sign)
	fmt.Println("生成的签名 timestamp:", timestamp)
}
