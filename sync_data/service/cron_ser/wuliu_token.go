package cron_ser

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/anti-captcha/anticaptcha-go"
	"github.com/avast/retry-go/v4"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync_data/core"
	"sync_data/global"
	"sync_data/models"
	"time"

	"github.com/chromedp/chromedp"
)

// 同步物流token
func SyncWuliuToken() {
	const maxRetries = 3 // 每个账号最多重试 3 次（即共尝试 4 次）
	global.Log.Info("🚀 开始获取 物流 Token...")
	var token TokenData
	token, err := retryGetWuliuToken("湖南聚智跨境电子商务有限公司", "Czw652982$", maxRetries)
	if err != nil {
		errMsg := fmt.Sprintf("❌ 最终失败：获取 物流 Token1 失败，重试 %d 次均失败: %v", maxRetries, err)
		global.Log.Error(errMsg)
		global.DB.Where("id = ?", 8).Updates(models.ServiceStatus{
			Status:  "错误",
			Message: errMsg,
		})
		return
	}
	global.Config.Wuliu.Token = token.Value
	core.SetYaml()
	global.Log.Info(global.Config.Wuliu.Token)
	time.Sleep(1 * time.Second)

	global.Log.Info("🚀 开始获取 物流 Token2...")
	var token_two TokenData
	token_two, err = retryGetWuliuToken("湖南云驰跨境电子商务有限公司", "geByz8q@9J", maxRetries)
	if err != nil {
		errMsg := fmt.Sprintf("❌ 最终失败：获取 物流 Token2 失败，重试 %d 次均失败: %v", maxRetries, err)
		global.Log.Error(errMsg)
		global.DB.Where("id = ?", 8).Updates(models.ServiceStatus{
			Status:  "错误",
			Message: errMsg,
		})
		return
	}
	global.Config.Wuliu.TokenTwo = token_two.Value
	core.SetYaml()
	global.Log.Info(global.Config.Wuliu.TokenTwo)

	global.Log.Info("🚀 开始获取 物流 Token3...")
	var token_three TokenData
	token_three, err = retryGetWuliuToken("湖南聚智跨境电子商务有限公司陈彰武", "Czw652982$", maxRetries)
	if err != nil {
		errMsg := fmt.Sprintf("❌ 最终失败：获取 物流 Token3 失败，重试 %d 次均失败: %v", maxRetries, err)
		global.Log.Error(errMsg)
		global.DB.Where("id = ?", 8).Updates(models.ServiceStatus{
			Status:  "错误",
			Message: errMsg,
		})
		return
	}
	global.Config.Wuliu.TokenThree = token_three.Value
	core.SetYaml()
	global.Log.Info(global.Config.Wuliu.TokenTwo)
	global.DB.Where("id = ?", 8).Updates(models.ServiceStatus{
		Status:  "正常",
		Message: "定时更新物流token成功",
	})
}

// retryGetWuliuToken 封装带重试的 token 获取
func retryGetWuliuToken(account, password string, maxAttempts uint) (TokenData, error) {
	var token TokenData
	err := retry.Do(
		func() error {
			var err error
			token, err = GetWuliuToken(account, password)
			if err != nil {
				global.Log.Warn("账号 %s 获取物流 Token 失败，准备重试: %v", account, err)
				return err // 返回错误会触发重试
			}
			return nil // 成功则退出重试
		},
		retry.Attempts(maxAttempts),
		retry.Delay(3*time.Second), // 每次重试间隔 3 秒
		retry.LastErrorOnly(true),  // 日志只显示最后一次错误
		retry.OnRetry(func(n uint, err error) {
			global.Log.Info("🔁 账号 %s 第 %d 次重试，错误: %v", account, n, err)
		}),
	)
	return token, err
}

// 外层 JSON 的结构
type TokenData struct {
	Value  string `json:"value"`
	Expire int64  `json:"expire"`
}

// GetJumiaCenterToken 使用 chromedp 自动登录 Jumia 卖家中心，获取 userToken
func GetWuliuToken(username, password string) (token TokenData, err error) {

	// 创建一个临时目录用于用户数据（每次运行都不同）
	userDataDir, err := os.MkdirTemp("", "chromedp-userdata-*")
	if err != nil {
		return TokenData{}, fmt.Errorf("创建临时用户目录失败: %w", err)
	}
	defer os.RemoveAll(userDataDir) // 任务结束后自动清理
	execPath, err := getChromeExecPath()
	if err != nil {
		return TokenData{}, fmt.Errorf("浏览器检查失败: %w", err)
	}
	// 设置 Chrome 启动参数
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		// TODO
		chromedp.ExecPath(execPath),     // 跨平台使用代码
		chromedp.Flag("headless", true), // ✅ 生产环境设为 true；调试时改为 false 查看浏览器
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("blink-settings", "imagesEnabled=true"), // 不禁用图片加快加载
		chromedp.Flag("ignore-certificate-errors", true),
		chromedp.UserAgent(`Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36`),
		// 👇 关键：指定全新的用户数据目录
		chromedp.Flag("user-data-dir", userDataDir),
		// 👇 可选：禁用缓存
		chromedp.Flag("disk-cache-dir", "/dev/null"),
		chromedp.Flag("disable-application-cache", true),
		chromedp.Flag("disable-cache", true),
	)
	// ✅ 1. 先创建带超时的 context
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	// ✅ 2. 在超时 context 下创建 allocator
	ctx, allocatorCancel := chromedp.NewExecAllocator(ctx, opts...)
	defer allocatorCancel()

	// ✅ 3. 在超时 context 下创建 browser context
	ctx, browserCancel := chromedp.NewContext(ctx)
	defer browserCancel()

	// ✅ 现在所有的操作（启动 Chrome、创建上下文、执行任务）都在 180 秒超时保护下
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
		chromedp.SendKeys(`#username`, username),
		chromedp.Sleep(2 * time.Second),

		// 6. 输入密码
		chromedp.WaitReady(`#password`, chromedp.ByQuery),
		chromedp.Sleep(2 * time.Second),
		chromedp.SendKeys(`#password`, password),
	}

	// 执行任务1输入账号和密码
	err = chromedp.Run(ctx, tasks1)
	if err != nil {
		return TokenData{}, fmt.Errorf("chromedp 执行失败，输入账号密码失败: %w", err)
	}
	var imgSrc string
	err = chromedp.Run(ctx,
		// 等待 img 元素出现
		chromedp.WaitVisible(`#formLogin img`, chromedp.ByQuery),
		// 获取 src 属性
		chromedp.AttributeValue(`#formLogin img`, "src", &imgSrc, nil, chromedp.ByQuery),
	)
	if err != nil {
		return TokenData{}, fmt.Errorf("chromedp 执行失败，获取验证码图片信息失败: %w", err)
	}
	// 2. 下载图片
	imgData, err := downloadImage(imgSrc)
	if err != nil {
		return TokenData{}, fmt.Errorf("下载图片验证码失败: %w", err)
	}
	// 保存临时文件用于 OCR（或直接传入字节流）
	tempImage := "captcha.jpg"
	if err := os.WriteFile(tempImage, imgData, 0644); err != nil {
		return TokenData{}, fmt.Errorf("保存图片失败: %v", err)
	}
	// 确保函数退出时删除临时文件
	defer func() {
		_ = os.Remove(tempImage) // 忽略删除错误（如文件不存在）
	}()
	text, err := ImagCaptcha(tempImage)
	if err != nil {
		return TokenData{}, fmt.Errorf("识别验证码失败: %v", err.Error())
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
		return TokenData{}, fmt.Errorf("chromedp 执行失败: %w", err)
	}
	var tokenStr string

	//任务3输入验证码并登录，确认登陆成功后获取token
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
		chromedp.Evaluate(`localStorage.getItem('pro__Access-Token')`, &tokenStr),
	}
	//开启任务3输入图片中验证码并登录，获取token
	err = chromedp.Run(ctx, tasks3)
	if err != nil {
		return TokenData{}, fmt.Errorf("chromedp 执行失败，输入验证码并登录，确认登陆成功后获取token: %w", err)
	}
	if tokenStr == "" || tokenStr == "null" {
		return TokenData{}, fmt.Errorf("localStorage 中未找到 pro__Access-Token")
	}

	// 判断是 JSON 对象还是纯 JWT 字符串
	// 尝试解析为 JSON 对象
	var tokenData TokenData
	err = json.Unmarshal([]byte(tokenStr), &tokenData)
	if err != nil {
		return TokenData{}, fmt.Errorf("解析 JSON 失败: %w", err)
	}
	return tokenData, nil
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
