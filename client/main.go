package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// OctopClient 只调官方 API: /api/auth/login, /api/mcp/agents (MCP网关)
type OctopClient struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

func NewClient(baseURL string) *OctopClient {
	return &OctopClient{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTP:    &http.Client{Timeout: 120 * time.Second},
	}
}

// Login 调官方登录接口换 JWT
func (c *OctopClient) Login(username, password string) error {
	body, _ := json.Marshal(map[string]string{"username": username, "password": password})
	resp, err := c.HTTP.Post(c.BaseURL+"/api/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("登录请求失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("登录失败 HTTP %d: %s", resp.StatusCode, string(b))
	}
	var out struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return err
	}
	if out.AccessToken == "" {
		return fmt.Errorf("登录返回无 token")
	}
	c.Token = out.AccessToken
	return nil
}

// mcpCall 调 MCP 网关 (官方 POST /api/mcp/agents)
func (c *OctopClient) mcpCall(id int, method string, params map[string]any) (map[string]any, error) {
	payload := map[string]any{"jsonrpc": "2.0", "id": id, "method": method}
	if params != nil {
		payload["params"] = params
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", c.BaseURL+"/api/mcp/agents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Authorization", "Bearer "+c.Token)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(raw))
	}
	// SSE 场景: 取最后一个 data: 行; JSON 场景: 直接解析
	text := string(raw)
	if idx := strings.LastIndex(text, "data:"); idx >= 0 {
		text = strings.TrimSpace(text[idx+len("data:"):])
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return nil, fmt.Errorf("响应解析失败: %v 原文: %.200s", err, text)
	}
	return out, nil
}

// Expert MCP 返回里的专家
type Expert struct {
	ExpertID string `json:"expert_id"`
	Name     string `json:"name"`
}

// ListExperts 列出当前用户可用的专家
func (c *OctopClient) ListExperts() ([]Expert, error) {
	out, err := c.mcpCall(1, "tools/call", map[string]any{"name": "list_octop_experts", "arguments": map[string]any{}})
	if err != nil {
		return nil, err
	}
	// isError 判断
	if res, ok := out["result"].(map[string]any); ok {
		if sc, ok := res["structuredContent"].(map[string]any); ok {
			if experts, ok := sc["experts"].([]any); ok {
				var list []Expert
				for _, e := range experts {
					if m, ok := e.(map[string]any); ok {
						list = append(list, Expert{ExpertID: str(m["expert_id"]), Name: str(m["name"])})
					}
				}
				return list, nil
			}
			if errMsg, ok := sc["error"].(string); ok {
				return nil, fmt.Errorf("官方返回错误: %s", errMsg)
			}
		}
	}
	return nil, fmt.Errorf("未识别的响应结构")
}

// AskExpert 问专家, 返回回答与thread_id
func (c *OctopClient) AskExpert(expertID, message string) (answer, threadID string, err error) {
	out, err := c.mcpCall(2, "tools/call", map[string]any{
		"name": "call_octop_expert",
		"arguments": map[string]any{"expert_id": expertID, "message": message},
	})
	if err != nil {
		return "", "", err
	}
	if res, ok := out["result"].(map[string]any); ok {
		if sc, ok := res["structuredContent"].(map[string]any); ok {
			if e, ok := sc["error"].(string); ok && e != "" {
				return "", "", fmt.Errorf("专家调用失败: %s", e)
			}
			return str(sc["response"]), str(sc["thread_id"]), nil
		}
	}
	return "", "", fmt.Errorf("未识别的响应结构")
}

func str(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// version 由构建时 -ldflags 注入
var version = "dev"

func main() {
	ver := flag.Bool("version", false, "打印版本")
	base := flag.String("url", "http://127.0.0.1:8099", "Octop 服务地址")
	ws := flag.String("ws", "", "本地工作区目录(不设则纯对话, 不碰本地文件)")
	flag.Parse()
	if *ver {
		fmt.Println("octop-client", version)
		return
	}

	reader := bufio.NewReader(os.Stdin)
	prompt := func(label string) string {
		fmt.Print(label)
		line, _ := reader.ReadString('\n')
		return strings.TrimSpace(line)
	}
	username := prompt("用户名: ")
	fmt.Print("密码: ")
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)

	// 工作区初始化(沙箱锚点)
	var wsEval string
	var manifest string
	if *ws != "" {
		ev, err := workspaceRootEval(*ws)
		if err != nil {
			fmt.Println("❌", err)
			os.Exit(1)
		}
		wsEval = ev
		m, err := buildManifest(ev)
		if err != nil {
			fmt.Println("❌ 清单生成失败:", err)
			os.Exit(1)
		}
		manifest = m
		fmt.Printf("📁 工作区: %s\n%s", ev, manifest)
	}

	client := NewClient(*base)
	if err := client.Login(username, password); err != nil {
		fmt.Println("❌", err)
		os.Exit(1)
	}
	fmt.Println("✅ 登录成功")

	experts, err := client.ListExperts()
	if err != nil {
		fmt.Println("❌ 列专家失败:", err)
		os.Exit(1)
	}
	if len(experts) == 0 {
		fmt.Println("当前账号没有可用专家")
		os.Exit(0)
	}
	fmt.Printf("可用专家 %d 个:\n", len(experts))
	for i, e := range experts {
		fmt.Printf("  %d) %s (%s)\n", i+1, e.Name, e.ExpertID)
	}

	// 选第一个专家, 进入对话循环
	chosen := experts[0]
	fmt.Printf("\n已连接专家: %s\n", chosen.Name)
	if wsEval != "" {
		fmt.Println("文件操作模式: 专家可在工作区内读写, 每次需你批准")
	}
	fmt.Println("输入问题开始对话, 输入 exit 退出")

	var lastThread string
	for {
		q := prompt("你> ")
		if q == "" {
			continue
		}
		if q == "exit" || q == "退出" {
			break
		}
		// 带工作区上下文提问
		msg := q
		if wsEval != "" {
			msg = q + "\n\n[本地工作区文件清单]\n" + manifest +
				"\n如需操作文件, 请在回答末尾用 ```fileops 代码块输出JSON操作数组(op/path/body/to/reason), 只能用相对路径, 允许op: read/write/append/delete/move。不操作文件就不用输出该代码块。"
		}
		ans, thread, err := client.AskExpert(chosen.ExpertID, msg)
		if err != nil {
			fmt.Println("❌", err)
			continue
		}
		if thread != "" {
			lastThread = thread
		}
		_ = lastThread
		fmt.Printf("AI> %s\n\n", ans)

		// 解析操作计划
		ops := ParseOps(ans)
		if len(ops) == 0 {
			continue
		}
		fmt.Println()
		fmt.Println(FormatOpsPlan(ops))
		approved := readUserConfirm(reader)
		results := ExecuteOps(wsEval, ops, approved)
		for _, r := range results {
			switch r.Status {
			case "done":
				fmt.Printf("  ✅ %-6s %-20s %s\n", r.Op, r.Path, r.Detail)
			case "rejected":
				fmt.Printf("  ⛔ %-6s %-20s %s\n", r.Op, r.Path, r.Detail)
			default:
				fmt.Printf("  ❌ %-6s %-20s %s\n", r.Op, r.Path, r.Detail)
			}
		}
		auditLog(wsEval, username, chosen.ExpertID, ops, results)
		fmt.Println()
	}
}

// readPassword 保留给未来终端模式(交互式不回显), 管道模式统一走ReadString
