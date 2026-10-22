# Octop 瘦客户端 v0.1

Go 编写, 同一套代码编译 macOS / Windows。只调 Octop 官方 API（登录 + MCP 网关），不碰官方源码。

## 用法

```bash
./octop-client --url http://<云端地址>:<端口>
```

按提示输入账号密码, 自动列出你的专家, 进入对话循环。

## 已验证 (2026-10-22, 隔离实例 8099)

- 登录 → 列专家 → 对话回答, 全链路通
- 员工账号只能看到自己的专家 (官方天然隔离)
- 访问他人专家返回 expert is not accessible
- macOS (arm64) + Windows (amd64) 双构建通过

## 下一步

- [ ] 本地文件夹操作 (沙箱工作区, 走官方 workspace API)
- [ ] token 记账展示
- [ ] 打包成 .app / .exe 安装包
