package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/chromedp/chromedp"
)

// GetJumiaCenterToken 使用 chromedp 自动登录 Jumia 卖家中心，获取 userToken
func GetJumiaCenterToken() (string, error) {
	// 设置 Chrome 启动参数
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false), // ✅ 生产环境设为 true；调试时改为 false 查看浏览器
		chromedp.Flag("proxy-server", "socks5://127.0.0.1:7897"),
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
	tasks := chromedp.Tasks{
		// 1. 导航到登录页
		chromedp.Navigate("https://vendorcenter.jumia.com/sign-in"),

		// 2. 等待 body 加载（确保页面开始渲染）
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Sleep(3 * time.Second), // 给 JavaScript 留出执行时间

		// 3. 打印页面标题（用于调试）
		chromedp.Title(&title),
		chromedp.ActionFunc(func(ctx context.Context) error {
			fmt.Printf("🌐 页面标题: %s\n", title)
			return nil
		}),

		// 4. ✅ 使用 data-action 属性定位“Sign in with Email”按钮（最稳定）
		// HTML: <button data-action="keycloak-login" ...>
		chromedp.WaitVisible(`button[data-action="keycloak-login"]`, chromedp.ByQuery),
		chromedp.Sleep(1 * time.Second),
		chromedp.Click(`button[data-action="keycloak-login"]`, chromedp.ByQuery),
		chromedp.Sleep(2 * time.Second),

		// 5. 输入邮箱
		chromedp.WaitReady(`#username`, chromedp.ByQuery),
		chromedp.SendKeys(`#username`, "hdjurunning@gmail.com"),
		chromedp.Sleep(1 * time.Second),

		// 6. 输入密码
		chromedp.WaitReady(`#password`, chromedp.ByQuery),
		chromedp.SendKeys(`#password`, "xdE&Qw$y7#NDK8a"),
		chromedp.Sleep(1 * time.Second),

		// 7. 点击登录按钮
		chromedp.Click(`#kc-login`, chromedp.ByID),
		chromedp.Sleep(5 * time.Second), // 等待跳转

		// 8. 等待登录成功后的标志性元素（请根据实际页面调整）
		// 示例：等待仪表盘根组件
		chromedp.WaitVisible(`h1[data-cy="homepage-title"]`, chromedp.ByQuery),
		// 或者使用文本: //span[contains(text(), 'Dashboard')]
		chromedp.Sleep(3 * time.Second),

		// 9. 从 localStorage 获取 userToken
		chromedp.Evaluate(`localStorage.getItem('userToken')`, &userToken),
	}

	// 执行任务
	err := chromedp.Run(ctx, tasks)
	if err != nil {
		return "", fmt.Errorf("chromedp 执行失败: %w", err)
	}

	if userToken == "" {
		return "", fmt.Errorf("获取到的 userToken 为空，请检查是否登录成功或 localStorage 中是否存在该 key")
	}

	return userToken, nil
}

// main 函数：程序入口
func main() {
	fmt.Println("🚀 开始获取 Jumia Center Token...")

	token, err := GetJumiaCenterToken()
	if err != nil {
		log.Fatalf("❌ 获取 token 失败: %v", err)
	}

	fmt.Printf("✅ 成功获取 userToken: %s\n", token)

}
