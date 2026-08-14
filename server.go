package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// APIConfig /api/data 返回的配置部分
type APIConfig struct {
	AppName string     `json:"app_name"`
	Debug   bool       `json:"debug"`
	Fields  []FieldDef `json:"fields"`
}

// APIResponse /api/data 返回结构
type APIResponse struct {
	Status   string              `json:"status"`
	Config   APIConfig           `json:"config"`
	Contacts []map[string]string `json:"contacts"`
	Errors   []YAMLError         `json:"errors,omitempty"`
}

// App 应用运行时上下文
type App struct {
	Cfg          *Config
	ContactsPath string
	Logger       *Logger
	AC           *AccessControl
	Mux          *http.ServeMux
}

// indexHandler 返回内嵌的前端 HTML 页面
func (a *App) indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && r.URL.Path != "/index.html" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(indexHTML)
	if a.Logger != nil {
		a.Logger.Access(fmt.Sprintf("IP: %s | 路径: %s | 状态: 200", r.RemoteAddr, r.URL.Path))
	}
}

// faviconHandler 浏览器会自动请求 /favicon.ico，返回 204 避免控制台 404 噪声
func (a *App) faviconHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

// dataHandler 返回配置 + 通讯录数据（或错误信息）
func (a *App) dataHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	res := LoadContacts(a.ContactsPath, a.Cfg)
	status := "ok"
	contacts := res.Contacts
	if res.Fatal {
		// 文件不存在与语法错误区分：前者允许前端新增/导出以修复，不禁用功能
		if res.Missing {
			status = "missing"
			if a.Logger != nil {
				a.Logger.System(fmt.Sprintf("contacts.yaml 不存在（可新增/导出修复）: %s", summarizeErrors(res.Errors)))
			}
		} else {
			status = "error"
			if a.Logger != nil {
				a.Logger.Error(fmt.Sprintf("contacts.yaml 解析失败: %s", summarizeErrors(res.Errors)))
			}
		}
		contacts = []map[string]string{}
	} else if len(res.Errors) > 0 {
		if a.Logger != nil {
			a.Logger.Warn(fmt.Sprintf("contacts.yaml 存在警告: %s", summarizeErrors(res.Errors)))
		}
	}
	resp := APIResponse{
		Status:   status,
		Config:   APIConfig{AppName: a.Cfg.AppName, Debug: a.Cfg.Debug, Fields: a.Cfg.Fields},
		Contacts: contacts,
		Errors:   res.Errors,
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(resp)
	if a.Logger != nil {
		a.Logger.Access(fmt.Sprintf("IP: %s | 路径: %s | 状态: 200", r.RemoteAddr, r.URL.Path))
		if a.Cfg.Debug {
			a.Logger.Debug(fmt.Sprintf("请求 /api/data，返回 status=%s，联系人数=%d", status, len(contacts)))
		}
	}
}
