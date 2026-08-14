package main

// defaultReadme 首次运行自动生成的 README 内容（对应项目书第 15 节）
const defaultReadme = `# PhoneBook - 轻量级电话通讯录系统

## 简介
PhoneBook 是一个轻量级内部电话通讯录 Web 应用。核心设计哲学为“只读服务端 + 本地临时编辑”：
服务器仅提供静态页面与只读数据，所有编辑操作发生在浏览器内存中，刷新即恢复服务器数据。
真正的数据变更由管理员在服务器上手动替换 contacts.yaml 文件完成。

## 目录结构
- config/        配置文件夹
  - config.yaml     核心配置文件（端口、网段、字段定义、TLS 等）
  - contacts.yaml   通讯录数据文件
- logs/         日志文件夹（按日生成 PhoneBook-YYYY-MM-DD.log）
- certs/        TLS 证书文件夹（server.crt / server.key，首次自动生成自签证书）
- README.md     本说明文档

## 快速开始
将可执行文件放到任意目录，直接运行（前台）：
  ./PhoneBook            (Linux)
  PhoneBook.exe          (Windows)
首次运行会自动创建 config/、logs/、certs/ 与 README.md，并生成默认配置与 5 条演示联系人。
默认监听端口 8443，默认使用 HTTPS。

## 配置文件说明
### config.yaml
- app_name:        应用名称
- port:            监听端口（默认 8443，修改后需重启）
- https_enabled:   true 使用 HTTPS，false 使用 HTTP 明文
- tls_cert/tls_key: TLS 证书与私钥路径（仅 https_enabled=true 时生效）
- debug:           调试模式（true 时记录 [DEBUG] 日志，前端输出调试信息）
- allowed_networks:允许访问的 CIDR 网段（ipv4 / ipv6）
- blocked_networks:黑名单 CIDR 网段/IP（优先级高于白名单）
- fields:          通讯录字段定义（key 字段名 / label 表头中文 / maxlen 最大字符数）

### contacts.yaml
- 每行一个联系人，以 “  - ” 开头
- 字段名必须与 config.yaml 的 fields.key 一致
- 值建议用双引号包裹
- 字段值不得超过对应 maxlen
- 修改后保存并替换服务器文件即可，用户刷新网页可见

## 访问控制
仅 allowed_networks 中的 IP 可访问。不在列表中的连接将被直接丢弃（无任何响应）。
修改 allowed_networks 后需重启服务生效。

## 黑名单
blocked_networks 中的 IP/网段连接会被立即丢弃（无任何响应）。
黑名单优先级高于白名单：即使 IP 同时在 allowed 与 blocked 中也会被丢弃。
默认包含 0.0.0.1/32 与 255.255.255.255/32。留空（ipv6: []）不影响正常访问。

## HTTPS / TLS
首次运行且 https_enabled=true 时自动生成 RSA 2048、有效期 10 年的自签证书。
自定义证书：停止服务，将证书命名为 server.crt、私钥命名为 server.key 放入 certs/ 覆盖，重启即可。
关闭 HTTPS：将 https_enabled 改为 false 并重启，通过 http://IP:port 访问。

## 服务管理
### Linux (systemctl)
  sudo ./PhoneBook --install    安装并设为开机自启
  systemctl start PhoneBook     启动
  systemctl stop PhoneBook      停止
  systemctl status PhoneBook    状态
  sudo ./PhoneBook --uninstall  卸载

### Windows (sc)
  PhoneBook.exe --install       安装（需管理员权限）
  sc start PhoneBook            启动
  sc stop PhoneBook             停止
  sc query PhoneBook            状态
  PhoneBook.exe --uninstall     卸载（需管理员权限）

## CLI 命令
  PhoneBook                启动服务（前台）
  PhoneBook -h, --help     显示帮助
  PhoneBook -i, --install  安装为系统服务
  PhoneBook -u, --uninstall卸载系统服务

## 日志
日志位于 logs/PhoneBook-YYYY-MM-DD.log（UTF-8），按日切换。
级别：系统 / 访问 / 警告 / 错误 / DEBUG（仅 debug=true）。
每次请求记录 IP、路径、状态码；被丢弃连接记录 IP 与原因。

## 故障排查
- YAML 格式错误：程序不崩溃，控制台与页面提示错误位置；修正 contacts.yaml 后刷新即可。
- 证书问题：确认 certs/server.crt 与 server.key 存在且匹配；HTTPS 模式下缺失将拒绝启动。
- 端口占用：更换 config.yaml 的 port 并重启。
- 无法访问：检查源 IP 是否在 allowed_networks，是否被 blocked_networks 命中。

## 更新通讯录
1. 网页点击“导出 YAML”下载 contacts.yaml
2. 用文本编辑器修改联系人
3. SSH 登录服务器
4. 覆盖 config/contacts.yaml
5. 用户刷新网页即可看到新数据

## 版本信息
版本号：` + Version + `
编译日期：见可执行文件详细信息
`
