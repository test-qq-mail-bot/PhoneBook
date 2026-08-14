package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeContacts(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "contacts.yaml")
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

const validContacts = `contacts:
  - phone_id: "0001"
    name: "张三"
    desk_location: "A栋3楼301"
    department: "技术部"
    position: "工程师"
    remark: ""
    device_id: "DEV-001"
  - phone_id: "0002"
    name: "李四"
    desk_location: "A栋3楼302"
    department: "技术部"
    position: "高级工程师"
    remark: "负责网络维护"
    device_id: "DEV-002"
`

func TestLoadContactsValid(t *testing.T) {
	p := writeContacts(t, validContacts)
	res := LoadContacts(p, defaultConfig())
	if res.Fatal {
		t.Fatalf("Fatal = true, errors: %v", res.Errors)
	}
	if len(res.Contacts) != 2 {
		t.Errorf("Contacts len = %d, want 2", len(res.Contacts))
	}
	if len(res.Errors) != 0 {
		t.Errorf("unexpected errors: %v", res.Errors)
	}
	if res.Contacts[0]["name"] != "张三" {
		t.Errorf("first contact name = %q", res.Contacts[0]["name"])
	}
}

func TestLoadContactsMissingFile(t *testing.T) {
	res := LoadContacts(filepath.Join(t.TempDir(), "nope.yaml"), defaultConfig())
	if !res.Fatal {
		t.Error("expected Fatal = true for missing file")
	}
	if len(res.Contacts) != 0 {
		t.Error("expected empty contacts on fatal")
	}
	if len(res.Errors) == 0 || !strings.Contains(res.Errors[0].Message, "文件不存在") {
		t.Errorf("expected 文件不存在 error, got %v", res.Errors)
	}
}

func TestLoadContactsMissingField(t *testing.T) {
	content := `contacts:
  - phone_id: "0001"
    name: "张三"
    desk_location: "A栋3楼301"
    department: "技术部"
    position: "工程师"
    remark: ""
`
	res := LoadContacts(writeContacts(t, content), defaultConfig())
	if res.Fatal {
		t.Fatalf("Fatal should be false for missing optional field, errors: %v", res.Errors)
	}
	if len(res.Contacts) != 1 {
		t.Errorf("Contacts len = %d, want 1", len(res.Contacts))
	}
	found := false
	for _, e := range res.Errors {
		if strings.Contains(e.Message, "缺少必填字段 'device_id'") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected missing device_id error, got %v", res.Errors)
	}
}

func TestLoadContactsOverlong(t *testing.T) {
	long := "这是一个明显超过三十个字符长度的姓名用于触发超长字段校验错误的测试字符串xxxx"
	content := `contacts:
  - phone_id: "0001"
    name: "` + long + `"
    desk_location: "A栋3楼301"
    department: "技术部"
    position: "工程师"
    remark: ""
    device_id: "DEV-001"
`
	res := LoadContacts(writeContacts(t, content), defaultConfig())
	if res.Fatal {
		t.Fatalf("Fatal should be false for overlong field")
	}
	found := false
	for _, e := range res.Errors {
		if strings.Contains(e.Message, "超出最大长度") && strings.Contains(e.Message, "name") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected overlong name error, got %v", res.Errors)
	}
}

func TestLoadContactsSyntaxError(t *testing.T) {
	content := "contacts: [a, b\n"
	res := LoadContacts(writeContacts(t, content), defaultConfig())
	if !res.Fatal {
		t.Error("expected Fatal = true for syntax error")
	}
}

func TestLoadContactsMissingKey(t *testing.T) {
	res := LoadContacts(writeContacts(t, "foo: bar\n"), defaultConfig())
	if !res.Fatal {
		t.Error("expected Fatal = true for missing contacts key")
	}
}

func TestSummarizeErrors(t *testing.T) {
	errs := []YAMLError{
		{File: "contacts.yaml", Line: 5, Message: "测试错误A"},
		{File: "contacts.yaml", Line: 0, Message: "测试错误B"},
	}
	s := summarizeErrors(errs)
	if !strings.Contains(s, "第5行") || !strings.Contains(s, "测试错误A") || !strings.Contains(s, "测试错误B") {
		t.Errorf("summarizeErrors output wrong: %q", s)
	}
}
