# Changelog

Octop 工作台客户端的全部重要变更记录于此。
All notable changes to the Octop Workbench client are documented here.

> 格式：每个版本先完整中文说明，后完整英文说明。
> Format: full Chinese notes first, then full English notes, per release.

## [v0.6.0] - 2026-10-25

### 中文说明

#### 新增

- 专家实例管理：专家广场新增管理视图，可直接新建、编辑、启动/停止、删除专家实例（直连云端 /api/agents），新建专家即时可用，无需登云端后台。
- 技能库面板：每张专家卡片新增「技能」入口，列出内置与自定义技能，支持新建（技能标识/一句话说明/技能正文）与删除，自定义技能即时生效（/api/agents/{id}/skills）。
- 本地能力 · 授权目录（A 档）：侧栏新增「授权目录」面板，加入清单的目录允许专家用绝对路径读写（每个操作仍需批准）；清单外一律拒绝；符号链接展开校验防绕过；已授权目录自动注入对话提示词，专家清楚自己的边界。
- 本地能力 · 命令执行（B 档）：授权目录面板内新增开关，默认关闭。开启后专家可提请执行系统命令，每条弹出审批卡展示完整命令与理由；高危命令（删除系统目录/关机重启/磁盘写入/管道执行远程脚本等）即使已批准也强制拒绝；命令执行 30 秒超时保护；所有命令（含被拒的）写入本地审计日志。
- 管理后台 · 用户管理：新增「用户」页，全量用户列表支持设为管理员/成员、启用/禁用、清除登录锁定，展示角色与配额。
- 管理后台 · 模型供应商：新增「供应商」页，支持接入 OpenAI 兼容供应商（名称/Base URL/API Key/模型列表）、启用/停用、删除；API Key 以掩码显示（仅露前 6 后 4 位），不泄露明文。

#### 修复

- 修复确认对话框在 Wails WebView 内静默失效的问题：window.confirm 在 WKWebView 中不弹窗且直接返回取消，导致删除任务/专家/技能/供应商与各项开关点击无响应。全部改为两段式确认按钮（第一次点变「确认?」态，3 秒内再点才执行）。
- 修复专家卡片字段名不一致（云端 agent_id 与前端 agentID）导致的按钮点穿、误触发召唤。
- 修复技能面板按钮误触发专家召唤（补事件阻止冒泡）。

### English Notes

#### Added

- Expert instance management: new management view in Expert Plaza to create, edit, start/stop and delete expert instances directly (via /api/agents); newly created experts are usable immediately without opening the cloud console.
- Skill panel: each expert card now has a "Skills" entry listing builtin and custom skills, with create (slug/one-line description/body) and delete; custom skills take effect instantly (/api/agents/{id}/skills).
- Local capability · Allowed directories (Tier A): new sidebar "Allowed Directories" panel. Experts may read/write files by absolute path inside listed directories (every operation still requires approval); anything outside is rejected; symlinks are resolved to prevent escape; the allowed list is injected into the conversation so experts know their boundary.
- Local capability · Shell execution (Tier B): a toggle in the Allowed Directories panel, off by default. When enabled, experts can propose shell commands; each command shows an approval card with the full command and reason; high-risk commands (system-wide deletion, shutdown/reboot, raw disk writes, piping remote scripts, etc.) are rejected even if approved; 30s timeout per command; every command (including rejected ones) is written to a local audit log.
- Admin console · Users: new "Users" tab with the full user list — promote/demote admin, enable/disable, clear login lockout, with role and quota display.
- Admin console · Model providers: new "Providers" tab to add OpenAI-compatible providers (name / base URL / API key / model list), enable/disable and delete; API keys are masked (first 6 and last 4 characters only).

#### Fixed

- Confirmation dialogs silently failing inside the Wails WebView: window.confirm does not render in WKWebView and always returns false, making task/expert/skill/provider deletion and toggles unresponsive. Replaced with a two-step confirm button (first click arms it as "Confirm?", second click within 3 seconds executes).
- Expert cards mixing agent_id (server) and agentID (frontend), which caused click-through and accidental summoning.
- Skill panel buttons bubbling up and accidentally summoning the expert.

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
