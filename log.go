package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Logger 日志器：按日生成 logs/PhoneBook-YYYY-MM-DD.log，支持 系统/访问/警告/错误/DEBUG 级别。
type Logger struct {
	mu      sync.Mutex
	dir     string
	file    *os.File
	curDate string
}

func NewLogger(dir string) *Logger {
	return &Logger{dir: dir}
}

func (l *Logger) ensure() {
	now := time.Now()
	date := now.Format("2006-01-02")
	if l.file != nil && l.curDate == date {
		return
	}
	if l.file != nil {
		l.file.Close()
		l.file = nil
	}
	if err := os.MkdirAll(l.dir, 0755); err != nil {
		return
	}
	path := filepath.Join(l.dir, "PhoneBook-"+date+".log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	l.file = f
	l.curDate = date
}

func (l *Logger) write(level, msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.ensure()
	ts := time.Now().Format("2006-01-02 15:04:05")
	line := fmt.Sprintf("[%s] [%s] %s\n", ts, level, msg)
	if l.file != nil {
		l.file.WriteString(line)
	}
}

// System 系统级日志（始终输出）
func (l *Logger) System(msg string) { l.write("系统", msg) }

// Access 访问日志（始终输出）
func (l *Logger) Access(msg string) { l.write("访问", msg) }

// Warn 警告日志（始终输出）
func (l *Logger) Warn(msg string) { l.write("警告", msg) }

// Error 错误日志（始终输出）
func (l *Logger) Error(msg string) { l.write("错误", msg) }

// Debug 调试日志（仅 debug=true 时调用）
func (l *Logger) Debug(msg string) { l.write("DEBUG", msg) }

func (l *Logger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		l.file.Close()
		l.file = nil
	}
}
