package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// runCmd 执行外部命令，输出转发到当前进程
func runCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// isAdminWindows 通过打开物理磁盘句柄粗判是否管理员权限
func isAdminWindows() bool {
	f, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	if err != nil {
		return false
	}
	f.Close()
	return true
}

// installService 安装为系统服务（Linux systemd / Windows sc）
func installService() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		return err
	}
	exedir := filepath.Dir(exe)

	if runtime.GOOS == "linux" {
		if os.Geteuid() != 0 {
			return fmt.Errorf("请使用 root 权限运行（sudo ./PhoneBook --install）")
		}
		unit := "[Unit]\n" +
			"Description=PhoneBook 通讯录服务\n" +
			"After=network.target\n\n" +
			"[Service]\n" +
			"Type=simple\n" +
			"ExecStart=" + exe + "\n" +
			"WorkingDirectory=" + exedir + "\n" +
			"Restart=on-failure\n\n" +
			"[Install]\n" +
			"WantedBy=multi-user.target\n"
		path := "/etc/systemd/system/PhoneBook.service"
		if err := os.WriteFile(path, []byte(unit), 0644); err != nil {
			return err
		}
		_ = runCmd("systemctl", "daemon-reload")
		_ = runCmd("systemctl", "enable", "PhoneBook")
		_ = runCmd("systemctl", "start", "PhoneBook")
		return nil
	} else if runtime.GOOS == "windows" {
		if !isAdminWindows() {
			return fmt.Errorf("请使用管理员权限运行（PhoneBook.exe --install）")
		}
		_ = runCmd("sc", "create", "PhoneBook", "binPath=", exe, "start=", "auto")
		_ = runCmd("sc", "start", "PhoneBook")
		return nil
	}
	return fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
}

// uninstallService 卸载系统服务
func uninstallService() error {
	if runtime.GOOS == "linux" {
		if os.Geteuid() != 0 {
			return fmt.Errorf("请使用 root 权限运行")
		}
		_ = runCmd("systemctl", "stop", "PhoneBook")
		_ = runCmd("systemctl", "disable", "PhoneBook")
		_ = os.Remove("/etc/systemd/system/PhoneBook.service")
		_ = runCmd("systemctl", "daemon-reload")
		return nil
	} else if runtime.GOOS == "windows" {
		if !isAdminWindows() {
			return fmt.Errorf("请使用管理员权限运行")
		}
		_ = runCmd("sc", "stop", "PhoneBook")
		_ = runCmd("sc", "delete", "PhoneBook")
		return nil
	}
	return fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
}
