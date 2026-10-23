package main

import (
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
	Thread   string // 当前会话thread
	Manifest string
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
	Experts []core.Expert `json:"experts"`
	Usage   string        `json:"usage"`
}

type AskResult struct {
	OK     bool          `json:"ok"`
	Msg    string        `json:"msg"`
	Answer string        `json:"answer"`
	Thread string        `json:"thread"`
	Ops    []core.FileOp `json:"ops"`
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
	return map[string]any{"ok": true, "manifest": manifest, "root": root}
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

// Ask 向专家提问, 自动带工作区清单+fileops协议说明
func (a *App) Ask(message string) AskResult {
	if a.client == nil {
		return AskResult{Msg: "未登录"}
	}
	msg := message
	if a.WSRoot != "" {
		msg = message + "\n\n[本地工作区文件清单]\n" + a.Manifest +
			"\n如需操作文件, 请在回答末尾用 ```fileops 代码块输出JSON操作数组(op/path/body/to/reason), 只能用相对路径, 允许op: read/write/append/delete/move。不操作文件就不用输出该代码块。\n" +
			"重要: 清单里的路径就是fileops的path, 以工作区根为基准, 不要在path前再加工作区目录名。新建文件同理。\n" +
			"注意: 你运行在云端, 没有用户电脑的磁盘访问。你记忆里的目录状态不是用户电脑的真实状态, 以这条消息里的清单为唯一事实。需要文件内容时, 输出read的fileops计划并停下等待, 客户端会把内容回传给你。"
	}
	answer, thread, err := a.client.AskExpert(a.Expert, msg)
	if err != nil {
		return AskResult{Msg: err.Error()}
	}
	a.Thread = thread
	return AskResult{OK: true, Answer: answer, Thread: thread, Ops: core.ParseOps(answer)}
}

// Resume 续thread(如喂read结果)
func (a *App) Resume(message string) AskResult {
	if a.client == nil || a.Thread == "" {
		return AskResult{Msg: "无可续会话"}
	}
	answer, thread, err := a.client.ResumeExpert(a.Expert, a.Thread, message)
	if err != nil {
		return AskResult{Msg: err.Error()}
	}
	a.Thread = thread
	return AskResult{OK: true, Answer: answer, Thread: thread, Ops: core.ParseOps(answer)}
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
	results := core.ExecuteOps(a.WSRoot, ops, approved)
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
