package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

// FieldDef 通讯录字段定义（与 config.yaml 的 fields 对应）
type FieldDef struct {
	Key    string `yaml:"key" json:"key"`
	Label  string `yaml:"label" json:"label"`
	MaxLen int    `yaml:"maxlen" json:"maxlen"`
}

// NetworkGroup 网段分组（IPv4 / IPv6 的 CIDR 列表）
type NetworkGroup struct {
	IPv4 []string `yaml:"ipv4"`
	IPv6 []string `yaml:"ipv6"`
}

// Config 应用配置
type Config struct {
	AppName         string        `yaml:"app_name"`
	Port            int           `yaml:"port"`
	HTTPSEnabled    bool          `yaml:"https_enabled"`
	TLSCert         string        `yaml:"tls_cert"`
	TLSKey          string        `yaml:"tls_key"`
	Debug           bool          `yaml:"debug"`
	AllowedNetworks NetworkGroup  `yaml:"allowed_networks"`
	BlockedNetworks NetworkGroup  `yaml:"blocked_networks"`
	Fields          []FieldDef    `yaml:"fields"`
}

// defaultConfigYAML 首次运行自动生成的默认配置文件内容（与项目书 5.1 一致）
const defaultConfigYAML = `# ============================================================
# PhoneBook 核心配置文件
# 修改后需重启服务生效
# ============================================================

# 应用名称
app_name: "PhoneBook"

# 端口号
port: 8443

# 是否开启 HTTPS（true / false）
# true:  使用 certs/ 下的证书，通过 HTTPS 提供服务
# false: 使用 HTTP 明文提供服务
https_enabled: true

# TLS 证书路径（相对于程序根目录，仅 https_enabled 为 true 时生效）
tls_cert: "certs/server.crt"
tls_key: "certs/server.key"

# 调试模式（true / false）
# true:  后端日志记录 debug 信息；前端浏览器控制台实时输出调试信息
# false: 仅记录 系统/访问/警告/错误 级别日志
debug: false

# ============================================================
# 访问控制
# ============================================================

# 允许访问的网段（CIDR 格式）
# 不在此列表中的 IP 连接将被直接丢弃（无任何响应）
allowed_networks:
  ipv4:
    - "127.0.0.0/8"      # 本地回环
    - "10.0.0.0/8"        # A类私网
    - "172.16.0.0/12"     # B类私网
    - "192.168.0.0/16"    # C类私网
  ipv6:
    - "::1"               # IPv6 本地回环
    - "fe80::/10"         # 链路本地地址
    - "fd00::/8"          # IPv6 ULA 私网地址

# 黑名单网段/IP（CIDR 格式）
# 命中黑名单的 IP 连接将被直接丢弃（无任何响应）
# 黑名单优先级高于白名单：即使 IP 同时在 allowed 和 blocked 中，也会被丢弃
# 留空表示不启用黑名单
blocked_networks:
  ipv4:
    - "0.0.0.1/32"        # 非法源地址
    - "255.255.255.255/32" # 广播地址
  ipv6: []

# ============================================================
# 通讯录字段定义
# 修改此处可增减网页表单列
# key:     contacts.yaml 中对应的字段名
# label:   网页表头显示的中文名称
# maxlen:  该字段允许的最大字符数（输入限制）
# ============================================================
fields:
  - key: phone_id
    label: 电话编号
    maxlen: 20
  - key: name
    label: 姓名
    maxlen: 30
  - key: desk_location
    label: 办公桌位置
    maxlen: 80
  - key: department
    label: 部门
    maxlen: 50
  - key: position
    label: 职位
    maxlen: 50
  - key: remark
    label: 备注
    maxlen: 200
  - key: device_id
    label: 电话设备编号
    maxlen: 50
`

// defaultContactsYAML 首次运行自动生成的演示数据（与项目书 5.3 一致，含注释 + 5 条）
const defaultContactsYAML = `# ============================================================
# PhoneBook 通讯录数据文件
# ============================================================
# 使用说明：
#   1. 每行一个联系人，以 "  - " 开头（两个空格 + 短横线 + 空格）
#   2. 字段名称必须与 config.yaml 中 fields 定义的 key 完全一致
#   3. 值建议用双引号包裹，避免特殊字符导致解析错误
#   4. 每个字段的值不得超过 config.yaml 中定义的 maxlen 字符数
#   5. 修改完成后保存此文件，替换服务器上的同名文件即可
#   6. 请勿修改本注释区域
# ============================================================
# 字段说明：
#   phone_id       - 电话编号（最多20字符）
#   name           - 姓名（最多30字符）
#   desk_location  - 办公桌位置（最多80字符）
#   department     - 部门（最多50字符）
#   position       - 职位（最多50字符）
#   remark         - 备注（最多200字符，可为空）
#   device_id      - 电话设备编号（最多50字符）
# ============================================================

contacts:
  - phone_id: "0001"
    name: "张三"
    desk_location: "A栋3楼301"
    department: "技术部"
    position: "工程师"
    remark: ""
    device_id: "DEV-2024-001"

  - phone_id: "0002"
    name: "李四"
    desk_location: "A栋3楼302"
    department: "技术部"
    position: "高级工程师"
    remark: "负责网络维护"
    device_id: "DEV-2024-002"

  - phone_id: "0003"
    name: "王五"
    desk_location: "B栋1楼105"
    department: "行政部"
    position: "主管"
    remark: ""
    device_id: "DEV-2024-003"

  - phone_id: "0004"
    name: "赵六"
    desk_location: "B栋2楼210"
    department: "财务部"
    position: "会计"
    remark: "内线转 8004"
    device_id: "DEV-2024-004"

  - phone_id: "0005"
    name: "孙七"
    desk_location: "C栋1楼大厅"
    department: "前台"
    position: "接待"
    remark: ""
    device_id: "DEV-2024-005"
`

// defaultConfig 解析默认配置字符串，保证与生成的文件一致
func defaultConfig() *Config {
	var c Config
	_ = yaml.Unmarshal([]byte(defaultConfigYAML), &c)
	return &c
}

// LoadConfig 读取配置文件。发生任何错误（缺失或格式错误）时返回默认配置，
// 并返回非 nil 的 err 表示“使用了内置默认配置”。
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return defaultConfig(), err
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return defaultConfig(), err
	}
	return &c, nil
}
