package main

// Version 版本号。遵循项目全局规则：变动日期-V几。
// 项目书示例写 "v1.0" 仅为示意，本实现以本常量为准。
// 注意：升级版本号必须同步修改 winres/winres.json，并重跑
//       go-winres make --arch amd64 --out rsrc
// 重新生成 rsrc_windows_amd64.syso，否则 Windows exe 的
// 「详细信息」仍会显示旧版本号。
const Version = "20260919-V6"

// AppName 应用名称
const AppName = "PhoneBook"
