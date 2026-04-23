package cron_ser

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync_data/models"
	"time"

	"sync_data/global" // 替换成你自己的日志包
)

func SendServiceStatus() {
	var serviceStatusList []models.ServiceStatus
	if err := global.DB.Find(&serviceStatusList).Error; err != nil {
		global.Log.Errorf("查询服务状态失败: %v", err)
		return
	}

	if len(serviceStatusList) == 0 {
		SendMessage("当前没有服务状态数据")
		return
	}

	// 拼接表格
	var builder strings.Builder
	builder.WriteString("```服务状态监控\n")
	builder.WriteString(fmt.Sprintf("%-20s %-10s %-20s %-19s\n", "服务", "状态", "描述", "更新时间"))
	builder.WriteString(strings.Repeat("-", 75) + "\n")

	for _, status := range serviceStatusList {
		builder.WriteString(fmt.Sprintf("%-20s %-10s %-20s %-19s\n",
			status.Service,
			status.Status,
			truncate(status.Message, 200), // 避免太长
			status.UpdatedAt.Format("01-02 15:04:05"),
		))
	}
	builder.WriteString("```")

	msg := builder.String()

	// Telegram 单条消息最大 4096 字，超长需要切分
	const maxLen = 4000
	for len(msg) > 0 {
		chunk := msg
		if len(chunk) > maxLen {
			chunk = msg[:maxLen]
			msg = msg[maxLen:]
		} else {
			msg = ""
		}

		SendMessageWithMarkdown(chunk)
		time.Sleep(time.Second)
	}
}

// 帮助函数：过长的描述截断
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// 发送 MarkdownV2 格式的消息
func SendMessageWithMarkdown(text string) {
	token := "7728250445:AAGjFRfCpMh5qHrXbvw1zpiH6d1zV4H_oVg"
	chatID := "-1003009735314"

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)

	data := url.Values{}
	data.Set("chat_id", chatID)
	data.Set("text", text)
	data.Set("parse_mode", "MarkdownV2")

	// 判断环境变量 USE_PROXY
	useProxy := os.Getenv("USE_PROXY") == "true"

	var transport *http.Transport
	if useProxy {
		proxyURL, _ := url.Parse("socks5://127.0.0.1:7897")
		transport = &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 10 * time.Second,
			}).DialContext,
		}
		global.Log.Info("使用本地 SOCKS5 代理发送 Telegram 消息")
	} else {
		transport = &http.Transport{}
		global.Log.Info("不使用代理，直接发送 Telegram 消息")
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   180 * time.Second,
	}

	maxRetry := 20
	for i := 0; i < maxRetry; i++ {
		resp, err := client.Post(
			apiURL,
			"application/x-www-form-urlencoded",
			bytes.NewBufferString(data.Encode()),
		)
		if err != nil {
			global.Log.Errorf("尝试第 %d 次发送 Telegram 消息失败: %s", i+1, err)
			time.Sleep(2 * time.Second)
			continue
		}

		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		global.Log.Infof("发送 Telegram 消息成功: %s, Telegram 返回: %s", text, string(body))
		break
	}
}

func SendMessage(text string) {
	token := "7728250445:AAGjFRfCpMh5qHrXbvw1zpiH6d1zV4H_oVg"
	chatID := "-1003009735314"

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)

	data := url.Values{}
	data.Set("chat_id", chatID)
	data.Set("text", text)

	// 判断环境变量 USE_PROXY
	useProxy := os.Getenv("USE_PROXY") == "true"

	var transport *http.Transport
	if useProxy {
		proxyURL, _ := url.Parse("socks5://127.0.0.1:7897")
		transport = &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 10 * time.Second,
			}).DialContext,
		}
		global.Log.Info("使用本地 SOCKS5 代理发送 Telegram 消息")
	} else {
		transport = &http.Transport{}
		global.Log.Info("不使用代理，直接发送 Telegram 消息")
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   180 * time.Second,
	}

	maxRetry := 3
	for i := 0; i < maxRetry; i++ {
		resp, err := client.Post(apiURL, "application/x-www-form-urlencoded", bytes.NewBufferString(data.Encode()))
		if err != nil {
			global.Log.Errorf("尝试第 %d 次发送tg消息失败: %s", i+1, err)
			time.Sleep(2 * time.Second)
			continue
		}

		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		global.Log.Infof("发送tg消息成功: %s, Telegram返回: %s", text, string(body))
		break
	}
}
