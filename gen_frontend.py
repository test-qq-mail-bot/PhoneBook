#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
将 frontend/index.html 转换为 base64 内嵌的 Go 源文件 frontend_embed.go。

设计说明:
  目标环境的 Go 工具链(go1.26.5)对 //go:embed 指令支持异常(指令被完全忽略),
  因此改用 base64 编码 + 运行时解码的方式内嵌前端页面, 保持"单二进制 + 前端内嵌"
  的原有设计目标, 且不依赖任何第三方库。

用法:
  python gen_frontend.py
产物:
  源代码/frontend_embed.go  (package main, var indexHTML []byte)
注意:
  frontend_embed.go 由本脚本生成, 请勿手改; 修改前端请改 frontend/index.html 后重跑本脚本。
"""
import base64
import os

BASE = os.path.dirname(os.path.abspath(__file__))
SRC = os.path.join(BASE, "frontend", "index.html")
OUT = os.path.join(BASE, "frontend_embed.go")


def main():
    with open(SRC, "rb") as f:
        raw = f.read()
    enc = base64.b64encode(raw).decode("ascii")
    content = (
        "package main\n\n"
        'import "encoding/base64"\n\n'
        "// indexHTML 内嵌的前端页面，由 frontend/index.html 经 gen_frontend.py 自动生成，请勿手改。\n"
        "var indexHTML = func() []byte {\n"
        "\tconst enc = \"%s\"\n"
        "\tb, err := base64.StdEncoding.DecodeString(enc)\n"
        "\tif err != nil {\n"
        "\t\tpanic(\"index.html base64 解码失败: \" + err.Error())\n"
        "\t}\n"
        "\treturn b\n"
        "}()\n" % enc
    )
    with open(OUT, "w", encoding="utf-8") as f:
        f.write(content)
    print("generated %s (%d bytes html -> %d b64 chars)" % (OUT, len(raw), len(enc)))


if __name__ == "__main__":
    main()
