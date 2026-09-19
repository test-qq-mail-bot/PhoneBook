# PhoneBook 电话通讯录系统（源代码）

轻量级内部电话通讯录 Web 应用。核心设计：**单 Go 二进制 + 纯原生 HTML/CSS/JS 前端 + YAML 文件存储，无数据库、无登录鉴权、无写入接口**。

- 服务器只提供静态页面与**只读**数据（`GET /` 与 `/api/data`）。
- 所有编辑（新增/修改联系人、导出 YAML）发生在**浏览器内存**中，刷新页面即恢复服务器数据。
- 真正的持久化变更由管理员在服务器上手动替换 `config/contacts.yaml` 完成。

当前内置版本号：`20260919-V3`（见 `version.go`）。

---

## 特性

- 零前端依赖：前端为单文件 `frontend/index.html`，未使用任何框架或第三方库。
- 零运行时依赖：`go.mod` 中唯一第三方依赖为 `gopkg.in/yaml.v3`。
- HTTPS 优先：首次运行自动生成有效期 10 年的 RSA 2048 自签证书（SAN 自动包含本机网卡 IP）。
- 访问控制：基于 CIDR 的白名单（`allowed_networks`）+ 黑名单（`blocked_networks`）。
- 双栈监听：同时监听 IPv4（`0.0.0.0`）与 IPv6（`[::]`），默认回环 `127.0.0.0/8`、`::1` 可直接访问。
- 容错：YAML 解析错误不崩溃，页面与日志给出行号级提示；`contacts.yaml` 缺失时返回 `status: missing` 并提示修复。
- 可安装为系统服务：`--install` / `--uninstall`（Linux systemd / Windows sc）。

---

## 技术栈

- 语言：Go 1.23+
- 存储：YAML（`config/config.yaml` 配置与字段定义；`config/contacts.yaml` 通讯录数据）
- 前端：原生 HTML/CSS/JS（内嵌于二进制，非独立服务）
- 依赖：仅 `gopkg.in/yaml.v3`（Go module）

---

## 目录结构

```
源代码/
├── main.go            程序入口：CLI 参数、启动横幅、双栈监听、信号退出
├── server.go          HTTP 路由与处理器（index / api/data / favicon）
├── config.go          配置加载、默认配置、字段定义
├── contacts.go        通讯录加载、YAML 容错、错误行号定位
├── access.go          CIDR 访问控制（白名单/黑名单、双栈过滤）
├── cert.go            TLS 自签证书生成、SAN 写入本机网卡 IP
├── service.go         系统服务安装/卸载（systemd / Windows sc）
├── log.go             日志（按日切分、级别：系统/访问/警告/错误/DEBUG）
├── readme.go          运行时生成的面向最终用户的 README 内容
├── version.go         版本号常量（唯一权威来源）
├── frontend_embed.go  由 gen_frontend.py 生成的「前端 HTML 的 base64 内嵌」
├── gen_frontend.py    将 frontend/index.html 重新生成 frontend_embed.go
├── go.mod / go.sum   Go module 定义
├── frontend/
│   └── index.html     前端页面（唯一前端源文件）
└── winres/
    └── winres.json     Windows 可执行文件「详细信息」版本信息源
```

---

## 环境要求

- Go 工具链 1.23 或更高。
- （仅当构建带版本资源的 Windows exe 时）需要 `go-winres`：

  ```bash
  go install github.com/tc-hib/go-winres@latest
  # 确保 $GOPATH/bin 已加入 PATH
  ```

---

## 构建

### 1. 本地构建（当前平台）

```bash
cd 源代码
go build -o PhoneBook .
```

### 2. Windows amd64 构建

Windows EXE 在构建前需要根据 `winres/winres.json` 生成 Windows 版本资源。`rsrc_windows_amd64.syso` 是构建过程生成的中间文件，**不需要提交到 GitHub**，并已加入 `.gitignore`。

```bash
cd 源代码
# 生成 Windows 版本资源文件（rsrc_windows_amd64.syso）
go-winres make --arch amd64 --out rsrc
# 编译 Windows amd64
GOOS=windows GOARCH=amd64 go build -o PhoneBook.exe .
```

### 3. Linux amd64 构建

```bash
cd 源代码
GOOS=linux GOARCH=amd64 go build -o PhoneBook .
```

> 注意：每次修改 `winres/winres.json` 后，重新构建 Windows EXE 前都应执行 `go-winres make --arch amd64 --out rsrc`，以生成最新的 `rsrc_windows_amd64.syso`。该文件仅用于 Windows 构建，不应提交到 GitHub。

---

## GitHub Actions 自动构建与发布

推送代码到 `main`/ `master` 后，GitHub Actions 会自动构建 Linux amd64 与 Windows amd64 可执行文件，并上传到当前 Workflow 的 Artifacts。

当 `version.go` 中对应版本号的 GitHub Release 尚不存在时，Workflow 会自动创建 Release，并附带：

- `PhoneBook`（Linux amd64）
- `PhoneBook.exe`（Windows amd64）

例如当前版本 `20260919-V3` 会对应 Release 标签 `v20260919-V3`。同一版本后续普通提交不会重复创建 Release；需要重新发布同一版本时，可在 Actions 页面手动执行 `Run workflow` 并填写对应的 `release_tag`。

本项目的 GitHub Actions 使用 Ubuntu 24.04，并采用 Node.js 24 兼容的官方 Actions；Windows 发布资源按构建步骤动态生成，不提交 `rsrc_windows_amd64.syso`。

## 重要：修改前端后必须重生成内嵌

本项目**不使用 Go 官方的 `//go:embed`**（构建环境对该指令不可用），前端以 **base64 字符串**形式内嵌在 `frontend_embed.go` 的 `indexHTML` 变量中。因此：

> 任何时候修改了 `frontend/index.html`，都必须重跑生成脚本，否则改动**不会**生效（二进制仍内嵌旧内容）：

```bash
cd 源代码
python gen_frontend.py        # 重新生成 frontend_embed.go
go build -o PhoneBook .       # 再编译
```

`gen_frontend.py` 读取 `frontend/index.html`，输出 `frontend_embed.go`（包 `main`，变量 `indexHTML`）。

---

## 版本号规则

版本号遵循「变动日期-V几」，例如 2026-07-23 首版为 `20260723-V1`。

**升级版本号必须同步修改三处**，缺一不可：

1. `version.go` 中的 `Version` 常量。
2. `winres/winres.json` 的 `identity.version`、`file_version`、`product_version` 与 `info.FileVersion`/`info.ProductVersion`。
3. 在 Windows 构建前执行 `go-winres make --arch amd64 --out rsrc`，临时生成 `rsrc_windows_amd64.syso`；该文件不提交到 GitHub。

校验方式：

- Windows：查看 exe「详细信息 → 文件版本」。
- Linux：`strings -a -el <exe> | grep 2026`（UTF-16LE 资源串）；普通 `strings` 只能查到 Go 常量，不能判断资源版本。

---

## 运行

将编译产物放到任意目录，直接运行（前台）：

```bash
./PhoneBook          # Linux
PhoneBook.exe        # Windows
```

首次运行自动创建 `config/`、`logs/`、`certs/`、`README.md`，并生成默认配置与 5 条演示联系人。默认监听 `8443`、使用 HTTPS。`Ctrl+C` 停止。

CLI 参数：

```
PhoneBook                启动服务（前台）
PhoneBook -h, --help     显示帮助
PhoneBook -i, --install  安装为系统服务（开机自启）
PhoneBook -u, --uninstall 卸载系统服务
```

> 运行时在程序目录生成的 `README.md` 是面向最终用户的说明（内容见 `readme.go`），与本源码 README 用途不同。

---

## 配置与数据（摘要）

- `config/config.yaml`：端口、HTTPS 开关、TLS 路径、调试模式、访问白/黑名单、字段定义 `fields`（key/label/maxlen）。
- `config/contacts.yaml`：通讯录数据，每行一个联系人，字段名须与 `fields.key` 一致。
- `contacts.yaml` 不存在时程序返回 `status: missing`，页面提示并允许新增/导出以修复；解析错误不崩溃，给出行号级提示。
- 客户端不信任自签证书时浏览器会报 `tls: unknown certificate`，属正常现象：点击「高级 → 继续前往」即可（服务端日志已降噪）。

---

## 测试

```bash
cd 源代码
go vet ./...
go test ./...
```

测试覆盖 `config`、`contacts`、`access` 三套（共 15 个用例）。

---

## 说明

- 本仓库仅含源代码与构建资源。编译产物（可执行文件、运行期生成的 `config/`、`logs/`、`certs/`、`README.md`）由运行环境产生，不纳入版本控制。
- 本项目无登录/鉴权模块，不存在后台管理账号。
