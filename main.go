package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const helpText = `PhoneBook %s - 轻量级电话通讯录系统

用法:
  PhoneBook                  启动服务（前台运行）
  PhoneBook -h, --help       显示此帮助信息
  PhoneBook -i, --install    安装为系统服务（开机自启）
  PhoneBook -u, --uninstall  卸载系统服务

安装为服务:
  Linux:
    sudo ./PhoneBook --install
    管理命令:
      systemctl start PhoneBook
      systemctl stop PhoneBook
      systemctl status PhoneBook

  Windows（需管理员权限）:
    PhoneBook.exe --install
    管理命令:
      sc start PhoneBook
      sc stop PhoneBook
      sc query PhoneBook

配置文件:
  config/config.yaml     核心配置（端口、网段、字段定义等）
  config/contacts.yaml   通讯录数据

证书文件:
  certs/server.crt       TLS 证书
  certs/server.key       TLS 私钥

详细说明请查阅 README.md
`

func main() {
	help := flag.Bool("help", false, "显示帮助信息")
	h := flag.Bool("h", false, "显示帮助信息")
	install := flag.Bool("install", false, "安装为系统服务")
	i := flag.Bool("i", false, "安装为系统服务")
	uninstall := flag.Bool("uninstall", false, "卸载系统服务")
	u := flag.Bool("u", false, "卸载系统服务")
	flag.Parse()

	if *help || *h {
		fmt.Printf(helpText, Version)
		return
	}
	if *install || *i {
		if err := installService(); err != nil {
			fmt.Println("安装失败:", err)
			os.Exit(1)
		}
		fmt.Println("PhoneBook 服务已安装并设置为开机自启")
		return
	}
	if *uninstall || *u {
		if err := uninstallService(); err != nil {
			fmt.Println("卸载失败:", err)
			os.Exit(1)
		}
		fmt.Println("PhoneBook 服务已卸载")
		return
	}

	if err := run(); err != nil {
		fmt.Println("启动失败:", err)
		os.Exit(1)
	}
}

// exeDir 返回可执行文件所在目录
func exeDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func run() error {
	root := exeDir()
	cfgDir := filepath.Join(root, "config")
	logDir := filepath.Join(root, "logs")
	certDir := filepath.Join(root, "certs")
	for _, d := range []string{cfgDir, logDir, certDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("创建目录 %s 失败: %w", d, err)
		}
	}

	cfgPath := filepath.Join(cfgDir, "config.yaml")
	contactsPath := filepath.Join(cfgDir, "contacts.yaml")

	// 4.2 初始化：仅当 config.yaml 与 contacts.yaml 都不存在时才生成
	cfgExists := fileExists(cfgPath)
	contactsExists := fileExists(contactsPath)
	if !cfgExists && !contactsExists {
		if err := os.WriteFile(cfgPath, []byte(defaultConfigYAML), 0644); err != nil {
			return fmt.Errorf("写入默认 config.yaml 失败: %w", err)
		}
		if err := os.WriteFile(contactsPath, []byte(defaultContactsYAML), 0644); err != nil {
			return fmt.Errorf("写入默认 contacts.yaml 失败: %w", err)
		}
	} else if cfgExists != contactsExists {
		// 仅其中一个存在：不生成任何文件，记录警告
		missing := "contacts.yaml"
		if !cfgExists {
			missing = "config.yaml"
		}
		// 警告会在加载阶段由 Logger 输出
		_ = missing
	}

	// 4.3 README.md 生成（已存在则不覆盖）
	readmePath := filepath.Join(root, "README.md")
	if !fileExists(readmePath) {
		_ = os.WriteFile(readmePath, []byte(defaultReadme), 0644)
	}

	logger := NewLogger(logDir)
	defer logger.Close()

	// 加载配置（错误则使用默认配置并记录警告）
	cfg, cfgErr := LoadConfig(cfgPath)
	if cfgErr != nil {
		logger.Warn(fmt.Sprintf("config.yaml 加载失败，使用内置默认配置: %v", cfgErr))
	}

	crtPath := filepath.Join(root, cfg.TLSCert)
	keyPath := filepath.Join(root, cfg.TLSKey)

	// 访问控制
	ac, acErr := NewAccessControl(cfg.AllowedNetworks, cfg.BlockedNetworks)
	if acErr != nil {
		logger.Warn(fmt.Sprintf("访问控制配置无效，使用默认网段: %v", acErr))
		def := defaultConfig()
		ac, _ = NewAccessControl(def.AllowedNetworks, def.BlockedNetworks)
	}

	// 证书
	var tlsCfg *tls.Config
	proto := "HTTP"
	if cfg.HTTPSEnabled {
		proto = "HTTPS"
		if !fileExists(crtPath) || !fileExists(keyPath) {
			if err := generateSelfSignedCert(crtPath, keyPath); err != nil {
				return fmt.Errorf("生成自签证书失败: %w", err)
			}
			logger.System("已生成自签 TLS 证书")
		}
		cert, err := tls.LoadX509KeyPair(crtPath, keyPath)
		if err != nil {
			// 9.4 HTTPS 模式下证书加载失败 → 退出
			logger.Error(fmt.Sprintf("TLS 证书加载失败: %v", err))
			return fmt.Errorf("TLS 证书加载失败: %w", err)
		}
		tlsCfg = &tls.Config{Certificates: []tls.Certificate{cert}}
	}

	// 启动横幅（4.6）
	printBanner(cfg, contactsPath, crtPath, logDir, proto, logger)

	app := &App{Cfg: cfg, ContactsPath: contactsPath, Logger: logger, AC: ac}
	mux := http.NewServeMux()
	mux.HandleFunc("/", app.indexHandler)
	mux.HandleFunc("/api/data", app.dataHandler)
	mux.HandleFunc("/favicon.ico", app.faviconHandler)
	app.Mux = mux

	onDrop := func(ip net.IP, reason string) {
		logger.Access(fmt.Sprintf("IP: %s | 状态: 丢弃 (%s)", ip.String(), reason))
		if cfg.Debug {
			logger.Debug(fmt.Sprintf("丢弃连接: %s 原因=%s", ip.String(), reason))
		}
	}

	// 双栈监听
	listeners := []net.Listener{}
	addrs := []struct {
		net  string
		addr string
	}{
		{"tcp4", fmt.Sprintf("0.0.0.0:%d", cfg.Port)},
		{"tcp6", fmt.Sprintf("[::]:%d", cfg.Port)},
	}
	for _, ad := range addrs {
		raw, err := net.Listen(ad.net, ad.addr)
		if err != nil {
			logger.Warn(fmt.Sprintf("监听 %s %s 失败: %v", ad.net, ad.addr, err))
			continue
		}
		listeners = append(listeners, raw)
	}
	if len(listeners) == 0 {
		err := fmt.Errorf("没有任何可用监听端口（%d）", cfg.Port)
		logger.Error(err.Error())
		return err
	}

	for _, raw := range listeners {
		go func(raw net.Listener) {
			fl := newFilterListener(raw, ac, onDrop)
			var ln net.Listener = fl
			if tlsCfg != nil {
				ln = tls.NewListener(fl, tlsCfg)
			}
			srv := &http.Server{Handler: mux, ReadHeaderTimeout: 15 * time.Second}
			if err := srv.Serve(ln); err != nil {
				// TLS 握手失败（如客户端不信任自签证书）属正常现象，
				// 降级为普通提示，避免刷成红色错误造成恐慌。
				if strings.Contains(err.Error(), "TLS handshake") {
					logger.System(fmt.Sprintf("TLS 握手失败（自签证书未被客户端信任，属正常现象；浏览器访问时点击\"高级\"->\"继续前往\"即可）: %v", err))
				} else {
					logger.Warn(fmt.Sprintf("服务停止: %v", err))
				}
			}
		}(raw)
	}

	// 等待停止信号
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	logger.System("收到停止信号，正在退出")
	return nil
}

// printBanner 打印启动横幅（不显示任何访问地址）
func printBanner(cfg *Config, contactsPath, crtPath, logDir, proto string, logger *Logger) {
	line := func() { fmt.Println("--------------------------------------------") }
	border := func() { fmt.Println("============================================") }
	border()
	fmt.Printf("  %s %s\n", AppName, Version)
	border()
	fmt.Printf("  监听端口:     %d\n", cfg.Port)
	fmt.Printf("  协议:         %s\n", proto)
	fmt.Println("  允许访问网段:")
	for _, n := range cfg.AllowedNetworks.IPv4 {
		if n != "" {
			fmt.Printf("    IPv4: %s\n", n)
		}
	}
	for _, n := range cfg.AllowedNetworks.IPv6 {
		if n != "" {
			fmt.Printf("    IPv6: %s\n", n)
		}
	}
	fmt.Println("  黑名单网段:")
	for _, n := range cfg.BlockedNetworks.IPv4 {
		if n != "" {
			fmt.Printf("    IPv4: %s\n", n)
		}
	}
	for _, n := range cfg.BlockedNetworks.IPv6 {
		if n != "" {
			fmt.Printf("    IPv6: %s\n", n)
		}
	}

	cfgOK := "已加载"
	if cfgErr := fileExists(filepath.Join(exeDir(), "config", "config.yaml")); !cfgErr {
		cfgOK = "缺失"
	}
	// 数据文件状态
	res := LoadContacts(contactsPath, cfg)
	dataStatus := "已加载"
	if res.Fatal {
		dataStatus = "解析失败"
	}
	fmt.Printf("  配置文件:     config/config.yaml  [%s]\n", cfgOK)
	if res.Fatal {
		fmt.Println("  数据文件:     config/contacts.yaml  [解析失败]")
		for _, e := range res.Errors {
			if e.Line > 0 {
				fmt.Printf("    -> 第%d行: %s\n", e.Line, e.Message)
			} else {
				fmt.Printf("    -> %s\n", e.Message)
			}
		}
		fmt.Println("    -> 程序继续运行，但通讯录数据为空")
	} else {
		fmt.Printf("  数据文件:     config/contacts.yaml  [%s] (%d条)\n", dataStatus, len(res.Contacts))
	}
	if cfg.HTTPSEnabled {
		fmt.Printf("  TLS证书:      %s  [%s]\n", cfg.TLSCert, boolToOK(fileExists(crtPath)))
	} else {
		fmt.Println("  TLS证书:      未启用（HTTP 明文）")
	}
	fmt.Printf("  日志文件:     %s/PhoneBook-%s.log\n", logDir, time.Now().Format("2006-01-02"))
	fmt.Printf("  调试模式:     %s\n", boolToClose(cfg.Debug))
	line()
	fmt.Println("  按 Ctrl+C 停止服务")
	border()

	logger.System(fmt.Sprintf("PhoneBook 启动 (版本 %s)", Version))
	logger.System(fmt.Sprintf("监听端口: %d | 协议: %s", cfg.Port, proto))
	logger.System(fmt.Sprintf("允许网段: %s", strings.Join(nonEmpty(cfg.AllowedNetworks.IPv4), ", ")+", "+strings.Join(nonEmpty(cfg.AllowedNetworks.IPv6), ", ")))
	logger.System(fmt.Sprintf("黑名单: %s", strings.Join(nonEmpty(cfg.BlockedNetworks.IPv4), ", ")+", "+strings.Join(nonEmpty(cfg.BlockedNetworks.IPv6), ", ")))
	if res.Fatal {
		logger.Error(fmt.Sprintf("contacts.yaml 解析失败: %s", summarizeErrors(res.Errors)))
	} else {
		logger.System(fmt.Sprintf("配置文件加载成功；通讯录加载 %d 条", len(res.Contacts)))
	}
	if cfg.HTTPSEnabled {
		logger.System("TLS 证书加载成功")
	}
}

func boolToOK(b bool) string {
	if b {
		return "已加载"
	}
	return "缺失"
}
func boolToClose(b bool) string {
	if b {
		return "开启"
	}
	return "关闭"
}
func nonEmpty(s []string) []string {
	out := []string{}
	for _, v := range s {
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}
