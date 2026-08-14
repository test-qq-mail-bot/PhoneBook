package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// YAMLError 描述一个 YAML 解析/校验错误（精确到文件与行号）
type YAMLError struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Message string `json:"message"`
}

// LoadResult 通讯录加载结果
type LoadResult struct {
	Contacts []map[string]string
	Errors   []YAMLError
	// Fatal 表示发生了致命错误（语法错误 / 缺少 contacts 键 / 文件缺失），
	// 此时 Contacts 应为空，前端展示错误面板。
	Fatal bool
	// Missing 表示 contacts.yaml 文件不存在（区别于语法错误）。
	// 此时前端应提示用户可新增/导出以创建文件，而非禁用全部功能。
	Missing bool
}

var lineRe = regexp.MustCompile(`line (\d+)`)

func extractLine(msg string) int {
	m := lineRe.FindStringSubmatch(msg)
	if m != nil {
		var n int
		fmt.Sscanf(m[1], "%d", &n)
		return n
	}
	return 0
}

// LoadContacts 加载通讯录数据。
// 设计要点：
//   - 每次调用都重新读取文件（管理员替换文件后用户刷新即可看到新数据，无需重启）。
//   - 使用 yaml.Node 解码以获取每个节点的源码行号，实现精确到行号的容错。
//   - 致命错误（YAML 语法错误、缺 contacts 键、文件缺失）→ Fatal=true，Contacts 为空。
//   - 软错误（联系人缺少某字段、字段值超长）→ 仍尽力加载该联系人，并在 Errors 中列出，
//     同时由调用方记录警告日志（遵循“不崩溃”原则）。
func LoadContacts(path string, cfg *Config) LoadResult {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return LoadResult{Errors: []YAMLError{{File: "contacts.yaml", Line: 0, Message: "文件不存在"}}, Fatal: true, Missing: true}
		}
		return LoadResult{Errors: []YAMLError{{File: "contacts.yaml", Line: 0, Message: err.Error()}}, Fatal: true}
	}

	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return LoadResult{
			Errors: []YAMLError{{File: "contacts.yaml", Line: extractLine(err.Error()), Message: err.Error()}},
			Fatal:  true,
		}
	}

	if len(root.Content) == 0 {
		return LoadResult{Errors: []YAMLError{{File: "contacts.yaml", Line: 0, Message: "文件为空"}}, Fatal: true}
	}
	doc := root.Content[0]
	if doc.Kind != yaml.MappingNode {
		return LoadResult{Errors: []YAMLError{{File: "contacts.yaml", Line: doc.Line, Message: "根节点应为映射（mapping）"}}, Fatal: true}
	}

	var seq *yaml.Node
	for i := 0; i+1 < len(doc.Content); i += 2 {
		if doc.Content[i].Value == "contacts" {
			seq = doc.Content[i+1]
			break
		}
	}
	if seq == nil {
		return LoadResult{Errors: []YAMLError{{File: "contacts.yaml", Line: 0, Message: "缺少 contacts 键"}}, Fatal: true}
	}
	if seq.Kind != yaml.SequenceNode {
		return LoadResult{Errors: []YAMLError{{File: "contacts.yaml", Line: seq.Line, Message: "contacts 应为列表（sequence）"}}, Fatal: true}
	}

	var contacts []map[string]string
	var errs []YAMLError

	for _, item := range seq.Content {
		if item.Kind != yaml.MappingNode {
			errs = append(errs, YAMLError{File: "contacts.yaml", Line: item.Line, Message: "联系人应为映射（mapping）"})
			continue
		}
		m := map[string]string{}
		for j := 0; j+1 < len(item.Content); j += 2 {
			k := item.Content[j]
			v := item.Content[j+1]
			key := k.Value
			val := ""
			if v.Kind == yaml.ScalarNode {
				val = v.Value
			}
			m[key] = val
			// 字段超长校验
			for _, f := range cfg.Fields {
				if f.Key == key && len([]rune(val)) > f.MaxLen {
					errs = append(errs, YAMLError{
						File:    "contacts.yaml",
						Line:    v.Line,
						Message: fmt.Sprintf("字段 '%s' 的值超出最大长度（当前 %d 字符，最大 %d 字符）", key, len([]rune(val)), f.MaxLen),
					})
				}
			}
		}
		// 缺少字段校验
		for _, f := range cfg.Fields {
			if _, ok := m[f.Key]; !ok {
				errs = append(errs, YAMLError{
					File:    "contacts.yaml",
					Line:    item.Line,
					Message: fmt.Sprintf("联系人缺少必填字段 '%s'", f.Key),
				})
			}
		}
		contacts = append(contacts, m)
	}

	return LoadResult{Contacts: contacts, Errors: errs, Fatal: false}
}

// isFatalContactsError 判断 LoadResult 是否应被视为致命（用于日志记录归类）
func (l LoadResult) isFatalContactsError() bool { return l.Fatal }

// 给前端用的短错误汇总（避免过长）
func summarizeErrors(errs []YAMLError) string {
	var sb strings.Builder
	for i, e := range errs {
		if i > 0 {
			sb.WriteString("; ")
		}
		if e.Line > 0 {
			sb.WriteString(fmt.Sprintf("第%d行: %s", e.Line, e.Message))
		} else {
			sb.WriteString(e.Message)
		}
	}
	return sb.String()
}
