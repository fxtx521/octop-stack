package main

import (
	"time"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	core "github.com/fxtx521/octop-stack/core"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App GUI后端: 登录/对话/文件树/审批执行, 全部复用core(与CLI同源)
type App struct {
	ctx      context.Context
	client   *core.OctopClient
	WSRoot   string // 已eval过的工作区根
	Expert   string // 当前专家ID
	Thread   string // 兼容保留: AskResult用
	Manifest    string
	AllowedEval []string // A档授权目录(symlink展开后)
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

type LoginResult struct {
	OK      bool          `json:"ok"`
	Role    string        `json:"role"`
	Msg     string        `json:"msg"`
	Server  string        `json:"server"`
	User    string        `json:"user"`
	Experts []core.Expert `json:"experts"`
	Usage   string        `json:"usage"`
}

type AskResult struct {
	OK     bool          `json:"ok"`
	Msg    string        `json:"msg"`
	Answer string        `json:"answer"`
	Thread string        `json:"thread"`
	Ops    []core.FileOp `json:"ops"`
	Shops  []core.ShOp   `json:"shops"`
}

type OpResult2 struct {
	Op     string `json:"op"`
	Path   string `json:"path"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

type ApplyResult struct {
	OK       bool        `json:"ok"`
	Results  []OpResult2 `json:"results"`
	Feedback string      `json:"feedback"` // read内容(喂回专家)
}

// PickDirectory 弹系统目录选择框
func (a *App) PickDirectory() string {
	dir, err := wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "选择项目目录(AI可操作的本地文件夹)",
	})
	if err != nil || dir == "" {
		return ""
	}
	return dir
}

// SetWorkspace 设置本地项目目录并生成清单
func (a *App) SetWorkspace(root string) map[string]any {
	eval, err := core.WorkspaceRoot(root)
	if err != nil {
		return map[string]any{"ok": false, "msg": err.Error()}
	}
	manifest, err := core.BuildManifest(eval)
	if err != nil {
		return map[string]any{"ok": false, "msg": err.Error()}
	}
	a.WSRoot = eval
	a.Manifest = manifest
	a.saveWsPref(eval)
	return map[string]any{"ok": true, "manifest": manifest, "root": root}
}

// wsPrefPath 工作区偏好文件
func wsPrefPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".octop-workbench.json")
}

func (a *App) saveWsPref(root string) {
	pref := a.readPref()
	pref.LastWorkspace = root
	a.writePref(pref)
}

// StartupWorkspace 启动时恢复上次工作区
func (a *App) StartupWorkspace() map[string]any {
	data, err := os.ReadFile(wsPrefPath())
	if err == nil {
		var pref struct {
			LastWorkspace string `json:"last_workspace"`
		}
		var full prefFile
		_ = json.Unmarshal(data, &full)
		a.loadAllowed(full.AllowedDirs)
		if json.Unmarshal(data, &pref) == nil && pref.LastWorkspace != "" {
			eval, err := core.WorkspaceRoot(pref.LastWorkspace)
			if err == nil {
				if manifest, err := core.BuildManifest(eval); err == nil {
					a.WSRoot = eval
					a.Manifest = manifest
					return map[string]any{"ok": true, "root": pref.LastWorkspace, "manifest": manifest}
				}
			}
		}
	}
	return map[string]any{"ok": false}
}

// CurrentWorkspace 返回当前工作区(前端刷新用)
func (a *App) CurrentWorkspace() map[string]any {
	if a.WSRoot == "" {
		return map[string]any{"ok": false}
	}
	return map[string]any{"ok": true, "root": a.WSRoot}
}

// ListFiles 返回工作区文件列表(前端文件树)
func (a *App) ListFiles() []string {
	if a.WSRoot == "" {
		return nil
	}
	var out []string
	root := strings.TrimSuffix(a.WSRoot, "/")
	filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if p != root && strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		if strings.HasPrefix(filepath.Base(rel), ".") {
			return nil
		}
		out = append(out, fmt.Sprintf("%s (%dB)", rel, info.Size()))
		return nil
	})
	sort.Strings(out)
	return out
}

// Login 登录云端实例
func (a *App) Login(server, username, password string) LoginResult {
	c := core.NewClient(server)
	if err := c.Login(username, password); err != nil {
		return LoginResult{Msg: "登录失败: " + err.Error()}
	}
	a.client = c
	a.saveSession(server, username, c.Token)
	experts, err := c.ListExperts()
	if err != nil {
		return LoginResult{Msg: "取专家列表失败: " + err.Error()}
	}
	usage, _ := c.UsageSummary()
	if len(experts) > 0 {
		a.Expert = experts[0].ExpertID
	}
	return LoginResult{OK: true, Experts: experts, Usage: usage, Role: c.Role}
}

// TryResume 启动时用持久化会话自动续登(免重复登录)
func (a *App) TryResume() LoginResult {
	sess := a.loadSession()
	if sess.Server == "" || sess.Token == "" {
		return LoginResult{Msg: "无历史会话"}
	}
	c := core.NewClient(sess.Server)
	if err := c.TryToken(sess.Server, sess.Token); err != nil {
		return LoginResult{Msg: "会话过期"}
	}
	a.client = c
	experts, err := c.ListExperts()
	if err != nil {
		return LoginResult{Msg: "取专家列表失败: " + err.Error()}
	}
	usage, _ := c.UsageSummary()
	if len(experts) > 0 {
		a.Expert = experts[0].ExpertID
	}
	return LoginResult{OK: true, Experts: experts, Usage: usage, Role: c.Role, Server: sess.Server, User: sess.Username}
}

type session struct {
	Server   string `json:"server"`
	Username string `json:"username"`
	Token    string `json:"token"`
}

func (a *App) saveSession(server, username, token string) {
	pref := a.readPref()
	pref.Session = &session{Server: server, Username: username, Token: token}
	a.writePref(pref)
}

func (a *App) loadSession() session {
	pref := a.readPref()
	if pref.Session != nil {
		return *pref.Session
	}
	return session{}
}

// ClearSession 退出登录: 只清会话, 保留工作区记忆
func (a *App) ClearSession() {
	pref := a.readPref()
	pref.Session = nil
	a.writePref(pref)
	a.client = nil
	a.Thread = ""
	a.Expert = ""
}

type prefFile struct {
	LastWorkspace string   `json:"last_workspace"`
	Session       *session `json:"session,omitempty"`
	AllowedDirs   []string `json:"allowed_dirs,omitempty"` // A档: 全盘授权目录清单
	AllowShell    bool     `json:"allow_shell,omitempty"`  // B档: 命令执行开关(默认关)
}

func (a *App) readPref() prefFile {
	data, err := os.ReadFile(wsPrefPath())
	if err != nil {
		return prefFile{}
	}
	var p prefFile
	json.Unmarshal(data, &p)
	return p
}

func (a *App) writePref(p prefFile) {
	data, _ := json.Marshal(p)
	os.WriteFile(wsPrefPath(), data, 0600)
}

// Ask 向专家提问, 自动带工作区清单+fileops协议说明
// expertID/threadID 由前端按任务传入: 无threadID=云端新会话, 有=续聊(同任务隔离)
func (a *App) Ask(expertID, threadID, message string) AskResult {
	if a.client == nil {
		return AskResult{Msg: "未登录"}
	}
	if expertID == "" {
		expertID = a.Expert
	}
	msg := message
	if a.WSRoot != "" {
		msg = message + "\n\n[本地工作区文件清单]\n" + a.Manifest +
			"\n如需操作文件, 请在回答末尾用 ```fileops 代码块输出JSON操作数组(op/path/body/to/reason), 只能用相对路径, 允许op: read/write/append/delete/move。不操作文件就不用输出该代码块。\n" +
			"重要: 清单里的路径就是fileops的path, 以工作区根为基准, 不要在path前再加工作区目录名。新建文件同理。\n" +
			"注意: 你运行在云端, 没有用户电脑的磁盘访问。你记忆里的目录状态不是用户电脑的真实状态, 以这条消息里的清单为唯一事实。需要文件内容时, 输出read的fileops计划并停下等待, 客户端会把内容回传给你。"
		if len(a.AllowedEval) > 0 {
			msg += "\n\n[授权目录]\n以下目录用户已授权访问, 可在fileops的path/to里直接用绝对路径: "
			for i, d := range a.AllowedEval {
				if i > 0 {
					msg += ", "
				}
				msg += d
			}
			msg += "\n授权目录外的绝对路径一律会被拒绝; 工作区内仍用相对路径。"
		}
	}
	if a.WSRoot != "" && a.readPref().AllowShell {
		msg += "\n\n[命令执行]\n用户已开启命令执行授权。如需在本机执行命令, 在回答末尾用 ```shops 代码块输出JSON数组(cmd/reason), 如 {\"cmd\":\"ls -la\",\"reason\":\"查看目录结构\"}。每条命令都要用户批准后才会执行, 高危命令会被强制拒绝。不需要就不输出。"
	}
	var answer, thread string
	var err error
	if threadID != "" {
		// 同任务已有会话: 续聊, 不再开新 thread
		answer, thread, err = a.client.ResumeExpert(expertID, threadID, msg)
		if err != nil {
			return AskResult{Msg: err.Error()}
		}
	} else {
		answer, thread, err = a.client.AskExpert(expertID, msg)
		if err != nil {
			return AskResult{Msg: err.Error()}
		}
	}
	return AskResult{OK: true, Answer: answer, Thread: thread,
		Ops: core.ParseOps(answer), Shops: core.ParseShOps(answer)}
}

// Resume 续指定任务的thread(如喂read结果)
func (a *App) Resume(expertID, threadID, message string) AskResult {
	if a.client == nil {
		return AskResult{Msg: "未登录"}
	}
	if expertID == "" {
		expertID = a.Expert
	}
	if threadID == "" {
		return AskResult{Msg: "无可续会话"}
	}
	answer, thread, err := a.client.ResumeExpert(expertID, threadID, message)
	if err != nil {
		return AskResult{Msg: err.Error()}
	}
	return AskResult{OK: true, Answer: answer, Thread: thread, Ops: core.ParseOps(answer)}
}

// ListExperts 重新拉取云端专家列表(广场同步用)
func (a *App) ListExperts() []core.Expert {
	if a.client == nil {
		return nil
	}
	experts, err := a.client.ListExperts()
	if err != nil {
		return nil
	}
	return experts
}

// ReadFile 预览工作区内文件内容(前端点击文件用)
func (a *App) ReadFile(rel string) map[string]any {
	if a.WSRoot == "" {
		return map[string]any{"ok": false, "msg": "未设置工作区"}
	}
	path, err := core.ResolveInWS(a.WSRoot, rel)
	if err != nil {
		return map[string]any{"ok": false, "msg": err.Error()}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return map[string]any{"ok": false, "msg": err.Error()}
	}
	if len(data) > 200*1024 {
		data = data[:200*1024]
	}
	return map[string]any{"ok": true, "content": string(data), "size": len(data)}
}

// ---------- 管理后台(admin角色) ----------

// AdminOverview 管理后台总览
func (a *App) AdminOverview() map[string]any {
	if a.client == nil {
		return map[string]any{"ok": false, "msg": "未登录"}
	}
	out, err := a.client.AdminGet("/api/admin/overview")
	if err != nil {
		return map[string]any{"ok": false, "msg": err.Error()}
	}
	return map[string]any{"ok": true, "data": out}
}

// AdminMetrics 运行指标
func (a *App) AdminMetrics() map[string]any {
	if a.client == nil {
		return map[string]any{"ok": false, "msg": "未登录"}
	}
	out, err := a.client.AdminGet("/api/admin/metrics")
	if err != nil {
		return map[string]any{"ok": false, "msg": err.Error()}
	}
	return map[string]any{"ok": true, "data": out}
}

// AdminAudit 审计日志
func (a *App) AdminAudit() map[string]any {
	if a.client == nil {
		return map[string]any{"ok": false, "msg": "未登录"}
	}
	out, err := a.client.AdminGet("/api/admin/audit-log")
	if err != nil {
		return map[string]any{"ok": false, "msg": err.Error()}
	}
	return map[string]any{"ok": true, "data": out}
}

// AdminInvites 邀请码列表
func (a *App) AdminInvites() map[string]any {
	if a.client == nil {
		return map[string]any{"ok": false, "msg": "未登录"}
	}
	out, err := a.client.AdminGet("/api/users/invites")
	if err != nil {
		return map[string]any{"ok": false, "msg": err.Error()}
	}
	return map[string]any{"ok": true, "data": out}
}

// AdminCreateInvite 生成开号邀请码(role: staff/admin)
func (a *App) AdminCreateInvite(role, note string) map[string]any {
	if a.client == nil {
		return map[string]any{"ok": false, "msg": "未登录"}
	}
	out, err := a.client.AdminPost("/api/users/invites", map[string]any{
		"role": role, "note": note,
	})
	if err != nil {
		return map[string]any{"ok": false, "msg": err.Error()}
	}
	return map[string]any{"ok": true, "data": out}
}

// Apply 用户批准后执行fileops计划(沙箱+审计都在core)
func (a *App) Apply(opsJSON string, approved bool) ApplyResult {
	var ops []core.FileOp
	if err := json.Unmarshal([]byte(opsJSON), &ops); err != nil {
		return ApplyResult{OK: false}
	}
	results := core.ExecuteOps(a.WSRoot, a.AllowedEval, ops, approved)
	out := make([]OpResult2, 0, len(results))
	var feedback strings.Builder
	for _, r := range results {
		out = append(out, OpResult2{Op: r.Op, Path: r.Path, Status: r.Status, Detail: r.Detail})
		if r.Op == "read" && r.Status == "done" && r.Content != "" {
			feedback.WriteString("\n[文件 " + r.Path + " 内容]\n" + r.Content + "\n")
		}
	}
	return ApplyResult{OK: true, Results: out, Feedback: feedback.String()}
}

// ---- 专家实例管理: 直接透传官方 /api/agents CRUD ----

func (a *App) AgentEnsure() string {
	if a.client == nil {
		return "未登录"
	}
	return ""
}

func (a *App) AgentsManage() []map[string]any {
	if a.client == nil {
		return nil
	}
	rows, err := a.client.AgentsRaw()
	if err != nil {
		return nil
	}
	return rows
}

func (a *App) AgentCreate(name, description, systemPrompt, welcome string, shared bool) map[string]any {
	if msg := a.AgentEnsure(); msg != "" {
		return map[string]any{"ok": false, "error": msg}
	}
	row, err := a.client.CreateAgent(name, description, systemPrompt, welcome, shared)
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	return map[string]any{"ok": true, "agent": row}
}

func (a *App) AgentUpdate(id string, fieldsJSON string) map[string]any {
	if msg := a.AgentEnsure(); msg != "" {
		return map[string]any{"ok": false, "error": msg}
	}
	fields := map[string]any{}
	if err := json.Unmarshal([]byte(fieldsJSON), &fields); err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	if err := a.client.UpdateAgent(id, fields); err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	return map[string]any{"ok": true}
}

func (a *App) AgentToggle(id, action string) map[string]any {
	if msg := a.AgentEnsure(); msg != "" {
		return map[string]any{"ok": false, "error": msg}
	}
	var err error
	if action == "stop" {
		err = a.client.StopAgent(id)
	} else {
		err = a.client.StartAgent(id)
	}
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	return map[string]any{"ok": true}
}

func (a *App) AgentDelete(id string) map[string]any {
	if msg := a.AgentEnsure(); msg != "" {
		return map[string]any{"ok": false, "error": msg}
	}
	if err := a.client.DeleteAgent(id); err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	return map[string]any{"ok": true}
}

// ---- 技能库绑定 ----

func (a *App) SkillList(agentID string) []map[string]any {
	if a.client == nil {
		return nil
	}
	rows, err := a.client.ListSkills(agentID)
	if err != nil {
		return nil
	}
	return rows
}

func (a *App) SkillCreate(agentID, slug, description, body string) map[string]any {
	if msg := a.AgentEnsure(); msg != "" {
		return map[string]any{"ok": false, "error": msg}
	}
	if err := a.client.CreateSkill(agentID, slug, description, body); err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	return map[string]any{"ok": true}
}

func (a *App) SkillDelete(agentID, slug string) map[string]any {
	if msg := a.AgentEnsure(); msg != "" {
		return map[string]any{"ok": false, "error": msg}
	}
	if err := a.client.DeleteSkill(agentID, slug); err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	return map[string]any{"ok": true}
}

// ---- A档: 授权目录清单(全盘访问的白名单制) ----

// loadAllowed 恢复授权目录并做symlink展开
func (a *App) loadAllowed(dirs []string) {
	a.AllowedEval = nil
	for _, d := range dirs {
		if ev, err := filepath.EvalSymlinks(d); err == nil {
			a.AllowedEval = append(a.AllowedEval, ev)
		}
	}
}

func (a *App) AllowedList() []string {
	pref := a.readPref()
	return pref.AllowedDirs
}

func (a *App) AllowedAdd(dir string) map[string]any {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return map[string]any{"ok": false, "error": "路径为空"}
	}
	if !filepath.IsAbs(dir) {
		return map[string]any{"ok": false, "error": "必须是绝对路径"}
	}
	if _, err := os.Stat(dir); err != nil {
		return map[string]any{"ok": false, "error": "目录不存在: " + dir}
	}
	pref := a.readPref()
	for _, d := range pref.AllowedDirs {
		if d == dir {
			return map[string]any{"ok": false, "error": "已在清单中"}
		}
	}
	pref.AllowedDirs = append(pref.AllowedDirs, dir)
	a.writePref(pref)
	a.loadAllowed(pref.AllowedDirs)
	return map[string]any{"ok": true}
}

func (a *App) AllowedRemove(dir string) map[string]any {
	pref := a.readPref()
	out := pref.AllowedDirs[:0]
	for _, d := range pref.AllowedDirs {
		if d != dir {
			out = append(out, d)
		}
	}
	pref.AllowedDirs = out
	a.writePref(pref)
	a.loadAllowed(out)
	return map[string]any{"ok": true}
}

// ---- 用户管理 + 模型供应商绑定 ----

func (a *App) UsersList() []map[string]any {
	if a.client == nil {
		return nil
	}
	rows, err := a.client.UsersRaw()
	if err != nil {
		return nil
	}
	return rows
}

func (a *App) UserPatch(id int, fieldsJSON string) map[string]any {
	if msg := a.AgentEnsure(); msg != "" {
		return map[string]any{"ok": false, "error": msg}
	}
	fields := map[string]any{}
	if err := json.Unmarshal([]byte(fieldsJSON), &fields); err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	if err := a.client.UserPatch(id, fields); err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	return map[string]any{"ok": true}
}

func (a *App) UserUnlock(id int) map[string]any {
	if msg := a.AgentEnsure(); msg != "" {
		return map[string]any{"ok": false, "error": msg}
	}
	if err := a.client.UserUnlock(id); err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	return map[string]any{"ok": true}
}

func (a *App) ProvidersList() []map[string]any {
	if a.client == nil {
		return nil
	}
	rows, err := a.client.ProvidersRaw()
	if err != nil {
		return nil
	}
	return rows
}

func (a *App) ProviderCreate(name, kind, baseURL, apiKey, modelsJSON string) map[string]any {
	if msg := a.AgentEnsure(); msg != "" {
		return map[string]any{"ok": false, "error": msg}
	}
	var models []map[string]string
	if modelsJSON != "" {
		json.Unmarshal([]byte(modelsJSON), &models)
	}
	row, err := a.client.ProviderCreate(name, kind, baseURL, apiKey, models)
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	return map[string]any{"ok": true, "provider": row}
}

func (a *App) ProviderPatch(id int, fieldsJSON string) map[string]any {
	if msg := a.AgentEnsure(); msg != "" {
		return map[string]any{"ok": false, "error": msg}
	}
	fields := map[string]any{}
	if err := json.Unmarshal([]byte(fieldsJSON), &fields); err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	if err := a.client.ProviderPatch(id, fields); err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	return map[string]any{"ok": true}
}

func (a *App) ProviderDelete(id int) map[string]any {
	if msg := a.AgentEnsure(); msg != "" {
		return map[string]any{"ok": false, "error": msg}
	}
	if err := a.client.ProviderDelete(id); err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	return map[string]any{"ok": true}
}

// ---- B档: 命令执行(默认关, 开关+审批+黑名单+留痕) ----

func (a *App) ShellState() bool {
	return a.readPref().AllowShell
}

func (a *App) ShellToggle(on bool) map[string]any {
	pref := a.readPref()
	pref.AllowShell = on
	a.writePref(pref)
	return map[string]any{"ok": true, "on": on}
}

// ApplySh 用户批准后执行命令计划(黑名单/超时/留痕都在core)
func (a *App) ApplySh(opsJSON string, approved bool) map[string]any {
	var ops []core.ShOp
	if err := json.Unmarshal([]byte(opsJSON), &ops); err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	results := core.ExecuteShOps(a.WSRoot, a.loadSession().Username, a.Expert, ops, approved, 30*time.Second)
	feedback := ""
	for _, r := range results {
		if r.Status == "done" && r.Output != "" {
			feedback += "\n[命令 " + r.Cmd + " 输出]\n" + r.Output + "\n"
		}
	}
	return map[string]any{"ok": true, "results": results, "feedback": feedback}
}
