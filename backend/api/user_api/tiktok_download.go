package user_api

import (
	"backend/models/res"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/gin-gonic/gin"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// TiktokDownloadView 处理 TikTok 视频解析请求
func (UserApi) TiktokDownloadView(c *gin.Context) {
	type Request struct {
		URL string `json:"url" binding:"required"`
	}

	var req Request
	if err := c.ShouldBindJSON(&req); err != nil {
		res.FailWithMessage("参数错误", c)
		return
	}

	// 基础校验
	if !strings.Contains(req.URL, "tiktok.com") || !strings.Contains(req.URL, "/video/") {
		res.FailWithMessage("请输入有效的 TikTok 视频链接", c)
		return
	}

	// 调用解析服务（使用 tiksave.io）
	hdURL, err := GetTikTokHDVideoURL(req.URL)
	if err != nil {
		log.Printf("解析失败: %v", err)
		res.FailWithMessage("解析失败: "+err.Error(), c)
		return
	}

	// 返回成功响应
	res.OkWithData(map[string]interface{}{
		"hd":   hdURL,
		"desc": "高清无水印视频",
	}, c)
}

// GetTikTokHDVideoURL 解析 TikTok 视频并返回高清无水印地址
func GetTikTokHDVideoURL(rawURL string) (string, error) {
	// Step 1: 请求 tiksave.io 解析接口
	parsedURL := url.QueryEscape(rawURL)

	client := &http.Client{}
	req, _ := http.NewRequest("POST", "https://tiksave.io/api/ajaxSearch", bytes.NewBufferString("q="+parsedURL+"&lang=zh-cn"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Referer", "https://tiksave.io/zh-cn")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// 解析 JSON 响应
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if result["status"] != "ok" {
		msg, _ := result["message"].(string)
		return "", fmt.Errorf("解析失败: %s", msg)
	}

	htmlStr, _ := result["data"].(string)
	if htmlStr == "" {
		return "", fmt.Errorf("未获取到解析内容")
	}

	// Step 2: 用 goquery 解析返回的 HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlStr))
	if err != nil {
		return "", err
	}

	// 方案1: 优先找 HD 下载链接（通常指向 CDN）
	var hdURL string
	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		text := s.Text()

		// 判断是否为 HD 下载链接
		if strings.Contains(text, "HD") && strings.Contains(href, "dl.snapcdn.app") {
			hdURL = href
		}
	})

	if hdURL != "" {
		// 跟随重定向获取真实地址
		realURL, err := ResolveRedirect(hdURL)
		if err == nil {
			return realURL, nil
		}
		log.Printf("HD链接跳转失败: %v", err)
	}

	// 方案2: 回退：从弹窗视频中提取 data-src
	videoSrc, exists := doc.Find("#popup_play video").Attr("data-src")
	if exists && videoSrc != "" {
		return videoSrc, nil
	}

	// 方案3: 尝试从 script 中提取 playAddr
	return ExtractVideoURLFromScript(htmlStr)
}

// ResolveRedirect 跟随重定向获取最终 URL
func ResolveRedirect(urlStr string) (string, error) {

	for i := 0; i < 5; i++ { // 最多 5 次跳转
		req, _ := http.NewRequest("HEAD", urlStr, nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 300 && resp.StatusCode < 400 {
			if loc := resp.Header.Get("Location"); loc != "" {
				urlStr = loc
				continue
			}
		}
		return resp.Request.URL.String(), nil
	}
	return urlStr, nil
}

// ExtractVideoURLFromScript 尝试从 script 中提取 playAddr（备用）
func ExtractVideoURLFromScript(html string) (string, error) {
	re := regexp.MustCompile(`"playAddr":"(https?://[^"]+)"`)
	matches := re.FindStringSubmatch(html)
	if len(matches) > 1 {
		decoded, _ := url.QueryUnescape(matches[1])
		return decoded, nil
	}
	return "", fmt.Errorf("未能提取到视频地址")
}
