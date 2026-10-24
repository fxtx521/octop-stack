# Changelog

Octop 工作台客户端的全部重要变更记录于此。
All notable changes to the Octop Workbench client are documented here.

> 格式：每个版本先完整中文说明，后完整英文说明。
> Format: full Chinese notes first, then full English notes, per release.

## [v0.5.0] - 2026-10-24

### 中文说明

#### 新增

- 专家广场：卡片网格页展示全部可用专家，支持搜索、头像、简介标签，点「召唤」直接以该专家新建任务。
- 免登录续会话：会话 token 本地持久化，启动时自动校验续登（TryResume），重开应用直达工作区，无需重复登录。
- 工作区持久化：记住上次工作文件夹，登录与启动时自动恢复目录与文件树。
- 退出登录：账户菜单新增退出操作，仅清除会话，保留工作区偏好。
- 工作区直填通道：在右栏粘贴文件夹路径即可切换工作区，不依赖原生目录选择器。
- 审批后文件树自动刷新：批准文件操作后文件树自动更新，无需手动刷新。
- 全局错误条：前端未捕获错误以可关闭的横条提示，不再静默失败。

#### 变更

- 界面 v3 重做：按 WorkBuddy 真机实测参数重写为浅色主题——浅灰侧栏、白卡细描边、绿色品牌色、顶部任务栏，圆角输入区带专家/工作区标签与圆形发送按钮。
- 侧栏常驻：所有页面（专家广场、资料库、定时任务、管理后台）均保留左侧导航，随时一键返回。
- fileops 提案剥离：回复正文不再显示原始 fileops 代码块，文件操作只以审批卡呈现。

#### 修复

- 导航孤岛：此前点进资料库/定时任务/专家广场/管理后台会隐藏侧栏，用户被困在页面无法返回；侧栏常驻后修复。
- 专家重复：演示服务端库中存在两条同名专家记录，广场卡片重复；已去重。
- 回车发送与输入法兼容：输入框回车稳定提交，macOS 下粘贴路径可靠。

#### 兼容性

- 服务端：Octop 1.0.2 官方包，零改动（2,277 个文件与 pip RECORD 哈希逐一比对一致）。
- 客户端-服务端协议：纯 HTTP——/api/auth/login、/api/mcp/agents（MCP tools/call）、/api/usage/summary。
- 桌面壳：Wails v2.17（Go 与 WebView 进程内绑定）。

### English

#### Added

- Expert Plaza: card-grid page listing all available experts with search, avatar, description tags and a one-click "Summon" button that starts a new task with the chosen expert.
- Persistent login: session token is stored locally and validated on startup (TryResume), so the app reopens straight into the workspace without signing in again.
- Workspace memory: the last working folder is remembered and its file tree is restored automatically on login and startup.
- Logout: account menu now has a logout action that clears the session while keeping workspace preferences.
- Direct path entry: paste a folder path in the right panel to switch workspace without the native folder picker.
- Post-approval file tree refresh: after approving file operations the tree refreshes by itself; no manual reload needed.
- Global error bar: uncaught frontend errors surface in a dismissible bar instead of failing silently.

#### Changed

- UI v3 redesign: light theme rebuilt against real WorkBuddy measurements — light-gray sidebar, white cards with hairline borders, green brand accent, top task bar, and a rounded composer with agent/workspace chips and circular send button.
- Sidebar is now persistent: the navigation sidebar stays on every page (Experts, Library, Scheduled Tasks, Admin), so you can always jump back from anywhere.
- fileops proposals hidden from replies: raw fileops code blocks are stripped from assistant replies; operations are shown only as approval cards.

#### Fixed

- Navigation dead-end: clicking Library / Scheduled Tasks / Expert Plaza / Admin used to hide the sidebar, trapping the user on the page with no way back. Fixed by keeping the sidebar persistent.
- Duplicate experts: the demo server database contained two identical agent rows, showing duplicated cards in the Plaza. Deduplicated.
- Enter-to-send and IME issues: message input now reliably submits via Enter and accepts pasted paths on macOS.

#### Compatibility

- Server: Octop 1.0.2 official package, unmodified (2,277 files hash-verified against the pip RECORD).
- Client-server protocol: HTTP only — /api/auth/login, /api/mcp/agents (MCP tools/call), /api/usage/summary.
- Desktop shell: Wails v2.17 (Go ↔ WebView in-process binding).
