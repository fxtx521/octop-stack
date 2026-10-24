# Changelog / 更新日志

All notable changes to the Octop Workbench client are documented here.
Octop 工作台客户端的全部重要变更记录于此。

## [v0.5.0] - 2026-10-24

### Added / 新增

- **Expert Plaza (专家广场)**: card-grid page listing all available experts
  with search, avatar, description tags and a one-click "Summon" button that
  starts a new task with the chosen expert.
  **专家广场**：卡片网格页展示全部可用专家，
  支持搜索、头像、简介标签，点「召唤」直接以该专家新建任务。
- **Persistent login (免登录续会话)**: session token is stored locally and
  validated on startup (`TryResume`), so the app reopens straight into the
  workspace without signing in again.
  **免登录续会话**：会话 token 本地持久化，启动时自动校验续登
  （`TryResume`），重开应用直达工作区，无需重复登录。
- **Workspace memory (工作区持久化)**: the last working folder is remembered
  and its file tree is restored automatically on login and startup.
  **工作区持久化**：记住上次工作文件夹，登录与启动时自动恢复目录与文件树。
- **Logout (退出登录)**: account menu now has a logout action that clears the
  session while keeping workspace preferences.
  **退出登录**：账户菜单新增退出操作，仅清除会话，保留工作区偏好。
- **Direct path entry (工作区直填通道)**: paste a folder path in the right
  panel to switch workspace without the native folder picker.
  **工作区直填通道**：在右栏粘贴文件夹路径即可切换工作区，不依赖原生目录选择器。
- **Post-approval file tree refresh (审批后文件树自动刷新)**: after approving
  file operations the tree refreshes by itself; no manual reload needed.
  **审批后文件树自动刷新**：批准文件操作后文件树自动更新，无需手动刷新。
- **Global error bar (全局错误条)**: uncaught frontend errors surface in a
  dismissible bar instead of failing silently.
  **全局错误条**：前端未捕获错误以可关闭的横条提示，不再静默失败。

### Changed / 变更

- **UI v3 redesign (界面 v3 重做)**: light theme rebuilt against real
  WorkBuddy measurements — light-gray sidebar, white cards with hairline
  borders, green brand accent, top task bar, and a rounded composer with
  agent/workspace chips and circular send button.
  **界面 v3 重做**：按 WorkBuddy 真机实测参数重写为浅色主题——
  浅灰侧栏、白卡细描边、绿色品牌色、顶部任务栏，
  圆角输入区带专家/工作区标签与圆形发送按钮。
- **Sidebar is now persistent (侧栏常驻)**: the navigation sidebar stays on
  every page (Experts, Library, Scheduled Tasks, Admin), so you can always
  jump back from anywhere.
  **侧栏常驻**：所有页面（专家广场、资料库、定时任务、管理后台）
  均保留左侧导航，随时一键返回。
- **fileops proposals hidden from replies (fileops 提案剥离)**: raw
  `fileops` code blocks are stripped from assistant replies; operations are
  shown only as approval cards.
  **fileops 提案剥离**：回复正文不再显示原始 `fileops` 代码块，
  文件操作只以审批卡呈现。

### Fixed / 修复

- **Navigation dead-end (页面孤岛)**: clicking Library / Scheduled Tasks /
  Expert Plaza / Admin used to hide the sidebar, trapping the user on the
  page with no way back. Fixed by keeping the sidebar persistent.
  **导航孤岛**：此前点进资料库/定时任务/专家广场/管理后台会隐藏侧栏，
  用户被困在页面无法返回；侧栏常驻后修复。
- **Duplicate experts (专家重复)**: the demo server database contained two
  identical agent rows, showing duplicated cards in the Plaza. Deduplicated.
  **专家重复**：演示服务端库中存在两条同名专家记录，广场卡片重复；已去重。
- **Enter-to-send and IME issues (回车发送与输入法兼容)**: message input now
  reliably submits via Enter and accepts pasted paths on macOS.
  **回车发送与输入法兼容**：输入框回车稳定提交，macOS 下粘贴路径可靠。

### Compatibility / 兼容性

- Server: Octop 1.0.2 official package, **unmodified** (2,277 files
  hash-verified against the pip RECORD).
  服务端：Octop 1.0.2 官方包，**零改动**（2,277 个文件与 pip RECORD
  哈希逐一比对一致）。
- Client-server protocol: HTTP only — `/api/auth/login`,
  `/api/mcp/agents` (MCP `tools/call`), `/api/usage/summary`.
  客户端-服务端协议：纯 HTTP——`/api/auth/login`、
  `/api/mcp/agents`（MCP `tools/call`）、`/api/usage/summary`。
- Desktop shell: Wails v2.17 (Go ↔ WebView in-process binding).
  桌面壳：Wails v2.17（Go 与 WebView 进程内绑定）。
