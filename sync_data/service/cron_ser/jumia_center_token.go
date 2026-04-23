package cron_ser

import (
	"context"
	"fmt"
	"github.com/avast/retry-go/v4"
	"github.com/chromedp/chromedp"
	"os"
	"os/exec"
	"runtime"
	"sync_data/core"
	"sync_data/global"
	"sync_data/models"
	"time"
)

// 同步今日订单以及订单详情
func SyncJumiaCenterToken() {
	const maxRetries = 3 // 每个账号最多重试 3 次（即共尝试 4 次）
	global.Log.Info("【JumiaCenterToken更新】✅ 开始获取 token")
	token, err := retryGetJumiaCenterToken("hdjurunning@gmail.com", "xdE&Qw$y7#NDK8a", maxRetries)
	if err != nil {
		errMsg := fmt.Sprintf("❌ 最终失败：获取 物流 JumiaCenterToken1 失败，重试 %d 次均失败: %v", maxRetries, err)
		global.Log.Error(errMsg)
		global.DB.Where("id = ?", 2).Updates(models.ServiceStatus{
			Status:  "错误",
			Message: errMsg,
			// UpdatedAt 会自动更新（如果是 time.Time 类型）
		})
		global.Log.Error("【JumiaCenterToken更新】获取 token 失败: %v", err)
		return
	}
	global.Config.Jumia.JumiaCenterToken = token
	var token_two string
	token_two, err = retryGetJumiaCenterToken("tuningdiliberti56@gmail.com", "KTEDbLjr4tvA99p$", maxRetries)
	if err != nil {
		errMsg := fmt.Sprintf("❌ 最终失败：获取 物流 JumiaCenterToken2 失败，重试 %d 次均失败: %v", maxRetries, err)
		global.Log.Error(errMsg)
		global.DB.Where("id = ?", 2).Updates(models.ServiceStatus{
			Status:  "错误",
			Message: errMsg,
			// UpdatedAt 会自动更新（如果是 time.Time 类型）
		})
		global.Log.Error("【JumiaCenterToken更新】获取 token 失败: %v", err)
		return
	}
	global.Config.JumiaTwo.JumiaCenterToken = token_two
	core.SetYaml()

	global.Log.Info("【JumiaCenterToken更新】完成")
	global.DB.Where("id = ?", 2).Updates(models.ServiceStatus{
		Status:  "正常",
		Message: fmt.Sprintf("定时同步jumia_center_token成功"),
	})
}

// retryGetWuliuToken 封装带重试的 token 获取
func retryGetJumiaCenterToken(account, password string, maxAttempts uint) (string, error) {
	var token string
	err := retry.Do(
		func() error {
			var err error
			token, err = GetJumiaCenterToken(account, password)
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
func getChromeExecPath() (string, error) {
	var name string
	switch runtime.GOOS {
	case "windows":
		// 常见安装路径，优先尝试
		paths := []string{
			`C:\Program Files\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				return p, nil
			}
		}
		// 如果不在固定路径，尝试从 PATH 中找 chrome.exe
		name = "chrome.exe"
	case "darwin":
		name = "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
	default: // linux, freebsd, etc.
		name = "google-chrome"
	}

	// 尝试从 PATH 中查找
	execPath, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("未找到 Chrome/Chromium 浏览器，请安装 Chrome 或 Chromium")
	}
	return execPath, nil
}

// --- 4. 获取 Token ---
func GetJumiaCenterToken(username, password string) (string, error) {
	// 创建一个临时目录用于用户数据（每次运行都不同）
	userDataDir, err := os.MkdirTemp("", "chromedp-userdata-*")
	if err != nil {
		return "", fmt.Errorf("创建临时用户目录失败: %w", err)
	}
	defer os.RemoveAll(userDataDir) // 任务结束后自动清理

	execPath, err := getChromeExecPath()
	if err != nil {
		return "", fmt.Errorf("浏览器检查失败: %w", err)
	}

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		// TODO
		chromedp.ExecPath(execPath), // 跨平台使用代码
		chromedp.Flag("headless", true),
		// TODO  在服务器上不需要这一行
		//chromedp.Flag("proxy-server", "socks5://127.0.0.1:7897"),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("blink-settings", "imagesEnabled=false"),
		chromedp.Flag("ignore-certificate-errors", true),
		chromedp.UserAgent(`Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36`),
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

	var userToken, title string

	tasks := chromedp.Tasks{
		chromedp.Navigate("https://vendorcenter.jumia.com/sign-in"),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Sleep(3 * time.Second),
		chromedp.Title(&title),
		chromedp.ActionFunc(func(ctx context.Context) error {
			global.Log.Info("🌐 页面标题: %s", title)
			return nil
		}),
		chromedp.WaitVisible(`button[data-action="jumia-idp-login"]`, chromedp.ByQuery),
		chromedp.Sleep(1 * time.Second),
		chromedp.Click(`button[data-action="jumia-idp-login"]`, chromedp.ByQuery),
		chromedp.Sleep(2 * time.Second),
		chromedp.WaitReady(`#username`, chromedp.ByQuery),
		chromedp.SendKeys(`#username`, username),
		chromedp.Sleep(1 * time.Second),
		chromedp.WaitReady(`#password`, chromedp.ByQuery),
		chromedp.SendKeys(`#password`, password),
		chromedp.Sleep(1 * time.Second),
		chromedp.Click(`//button[contains(., 'Sign In')]`, chromedp.BySearch),
		chromedp.Sleep(5 * time.Second),
		chromedp.WaitVisible(`h1[data-cy="homepage-title"]`, chromedp.ByQuery),
		chromedp.Sleep(3 * time.Second),
		chromedp.Evaluate(`localStorage.getItem('userToken')`, &userToken),
	}

	err = chromedp.Run(ctx, tasks)
	if err != nil {
		return "", fmt.Errorf("chromedp 执行失败: %w", err)
	}

	if userToken == "" {
		return "", fmt.Errorf("获取到的 userToken 为空，请检查是否登录成功")
	}

	return userToken, nil
}
