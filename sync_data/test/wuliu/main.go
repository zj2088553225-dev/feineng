package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/anti-captcha/anticaptcha-go"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

// GetJumiaCenterToken 使用 chromedp 自动登录 Jumia 卖家中心，获取 userToken
func GetWuliuToken() (string, error) {
	// 设置 Chrome 启动参数
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false), // ✅ 生产环境设为 true；调试时改为 false 查看浏览器
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("blink-settings", "imagesEnabled=false"), // 禁用图片加快加载
		chromedp.Flag("ignore-certificate-errors", true),
		chromedp.UserAgent(`Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36`),
	)

	// 创建浏览器上下文
	ctx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	// 新建任务上下文
	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()

	// 设置总超时时间（3分钟）
	ctx, cancel = context.WithTimeout(ctx, 180*time.Second)
	defer cancel()

	var userToken string
	var title string

	// 定义自动化任务
	tasks1 := chromedp.Tasks{
		// 1. 导航到登录页
		chromedp.Navigate("https://ka.choicexp.com/user/login"),

		// 2. 等待 body 加载（确保页面开始渲染）
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Sleep(5 * time.Second), // 给 JavaScript 留出执行时间

		// 3. 打印页面标题（用于调试）
		chromedp.Title(&title),
		chromedp.ActionFunc(func(ctx context.Context) error {
			fmt.Printf("🌐 页面标题: %s\n", title)
			return nil
		}),

		// 5. 输入邮箱
		chromedp.WaitReady(`#username`, chromedp.ByQuery),
		chromedp.SendKeys(`#username`, "湖南聚智跨境电子商务有限公司"),
		chromedp.Sleep(2 * time.Second),

		// 6. 输入密码
		chromedp.WaitReady(`#password`, chromedp.ByQuery),
		chromedp.Sleep(2 * time.Second),
		chromedp.SendKeys(`#password`, "Czw652982$"),
	}

	// 执行任务1输入账号和密码
	err := chromedp.Run(ctx, tasks1)
	if err != nil {
		return "", fmt.Errorf("chromedp 执行失败: %w", err)
	}
	var imgSrc string
	err = chromedp.Run(ctx,
		// 等待 img 元素出现
		chromedp.WaitVisible(`#formLogin img`, chromedp.ByQuery),
		// 获取 src 属性
		chromedp.AttributeValue(`#formLogin img`, "src", &imgSrc, nil, chromedp.ByQuery),
	)
	if err != nil {
		panic(fmt.Sprintf("获取图片地址失败: %v", err))
	}

	//fmt.Println("图片 URL:", imgSrc)

	// 2. 下载图片
	imgData, err := downloadImage(imgSrc)
	if err != nil {
		panic(fmt.Sprintf("下载图片失败: %v", err))
	}

	// 保存临时文件用于 OCR（或直接传入字节流）
	tempImage := "captcha.jpg"
	if err := os.WriteFile(tempImage, imgData, 0644); err != nil {
		panic(fmt.Sprintf("保存图片失败: %v", err))
	}
	text, err := ImagCaptcha(tempImage)
	if err != nil {

		fmt.Println("识别验证码失败:", err.Error())
	}
	fmt.Println("识别的验证码:", text)
	tasks2 := chromedp.Tasks{
		// 6. 输入图像验证码
		chromedp.WaitReady(`#formLogin > div.ant-tabs.ant-tabs-top.ant-tabs-line > div.ant-tabs-content.ant-tabs-content-animated.ant-tabs-top-content > div > div:nth-child(4) > div.ant-col.ant-col-8 > img`, chromedp.ByQuery),

		chromedp.SendKeys(`#inputCode`, text, chromedp.ByQuery),
		chromedp.Sleep(1 * time.Second),
	}
	//开启任务2读取图片中验证码并输入
	err = chromedp.Run(ctx, tasks2)
	if err != nil {
		return "", fmt.Errorf("chromedp 执行失败: %w", err)
	}
	tasks3 := chromedp.Tasks{
		chromedp.Sleep(1 * time.Second),
		// 7. 点击登录按钮
		chromedp.Click(`#formLogin > div:nth-child(3) > div > div > span > button`, chromedp.ByID),
		chromedp.Sleep(5 * time.Second), // 等待跳转

		// 8. 等待登录成功后的标志性元素（请根据实际页面调整）
		// 示例：等待仪表盘根组件
		chromedp.WaitVisible(`#app > section > section > main > div > div > div > div.page-header > div > div > div > div:nth-child(2) > div.avatar > span > img`, chromedp.ByQuery),
		// 或者使用文本: //span[contains(text(), 'Dashboard')]
		chromedp.Sleep(3 * time.Second),

		// 9. 从 localStorage 获取 userToken
		chromedp.Evaluate(`localStorage.getItem('pro__Access-Token')`, &userToken),
	}
	//开启任务3输入图片中验证码并登录，获取token
	err = chromedp.Run(ctx, tasks3)
	if err != nil {
		return "", fmt.Errorf("chromedp 执行失败: %w", err)
	}

	if userToken == "" {
		return "", fmt.Errorf("获取到的 userToken 为空，请检查是否登录成功或 localStorage 中是否存在该 key")
	}

	return userToken, nil
}

func ImagCaptcha(filepath string) (text string, err error) {
	ac := anticaptcha.NewClient("10a08b8af36d99291ecab8281eff14e3")
	ac.IsVerbose = true
	balance, err := ac.GetBalance()
	if err != nil {
		log.Fatal(err)
		return "", err
	}
	fmt.Println("Balance:", balance)
	solution, err := ac.SolveImageFile(filepath, anticaptcha.ImageSettings{})
	if err != nil {
		log.Fatal(err)
		return "", nil
	}
	fmt.Println("Captcha Solution:", solution)
	return solution, nil
}

// main 函数：程序入口
func main() {
	fmt.Println("🚀 开始获取 物流 Token...")

	token, err := GetWuliuToken()
	if err != nil {
		log.Fatalf("❌ 获取 物流 Token 失败: %v", err)
	}

	fmt.Printf("✅ 成功获取 物流 Token: %s\n", token)

	// 示例调用
	resp, err := GetCesFbjOrdersList(1, 10, token)
	if err != nil {
		fmt.Println("请求失败:", err)
		return
	}
	fmt.Println("响应结果:", resp)
}

// 下载图片
// downloadImage 下载图片，可以是普通 URL 或 data:image/...;base64,...
func downloadImage(url string) ([]byte, error) {
	// 如果是 base64 图片
	if strings.HasPrefix(url, "data:image") {
		parts := strings.SplitN(url, ",", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid base64 image data")
		}
		return base64.StdEncoding.DecodeString(parts[1])
	}

	// 创建一个带超时的 HTTP 客户端
	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	// 创建请求并设置 UA
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; ImageDownloader/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 检查 HTTP 状态
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request failed: %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}

// GetCesFbjOrdersList 请求 cesFbjOrders 列表
func GetCesFbjOrdersList(pageNo, pageSize int, token string) (string, error) {
	// 生成当前秒级时间戳
	timestamp := time.Now().Unix()

	// 拼接 URL
	url := fmt.Sprintf(
		"https://ka.choicexp.com/api/cesFbjOrders/list?_t=%d&column=createTime&order=desc&field=id,,,createTime,createBy,dispatchDocuments,receiptId,status,type,pickingType,channelName,whCode,whAddress,action&pageNo=%d&pageSize=%d",
		timestamp, pageNo, pageSize,
	)

	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	// 设置 Headers
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Referer", "https://ka.choicexp.com/bookingManage/booking/CesFbxOrdersList")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36")
	req.Header.Set("X-Access-Token", token)
	req.Header.Set("lang-type", "zh-CN")
	req.Header.Set("sec-ch-ua", `"Not)A;Brand";v="8", "Chromium";v="138", "Google Chrome";v="138"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"Windows"`)
	req.Header.Set("tenant_id", "0")

	// 设置 Cookie
	req.Header.Set("Cookie", "x-hng=lang=zh-CN&domain=ka.choicexp.com")

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
