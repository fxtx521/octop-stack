package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type OctopClient struct {
	BaseURL string
	Token   string
	Role    string // admin / staff
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
		User        struct {
			Role string `json:"role"`
		} `json:"user"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return err
	}
	if out.AccessToken == "" {
		return fmt.Errorf("登录返回无 token")
	}
	c.Token = out.AccessToken
	c.Role = out.User.Role
	return nil
}

// TryToken 用已存token恢复会话, 验证有效才返回nil
func (c *OctopClient) TryToken(server, token string) error {
	c.BaseURL = strings.TrimRight(server, "/")
	c.Token = token
	if _, err := c.ListExperts(); err != nil {
		return err
	}
	return nil
}

// AdminGet 管理端GET(需admin token)
func (c *OctopClient) AdminGet(path string) (map[string]any, error) {
	req, err := http.NewRequest("GET", c.BaseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(b))
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		// audit-log等返回数组
		return map[string]any{"_list": json.RawMessage(b)}, nil
	}
	return out, nil
}

// AdminPost 管理端POST
func (c *OctopClient) AdminPost(path string, body map[string]any) (map[string]any, error) {
	j, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", c.BaseURL+path, bytes.NewReader(j))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(b))
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return map[string]any{"_raw": string(b)}, nil
	}
	return out, nil
}

// mcpCall 调 MCP 网关 (官方 POST /api/mcp/agents)
func (c *OctopClient) MCPCall(id int, method string, params map[string]any) (map[string]any, error) {
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
	out, err := c.MCPCall(1, "tools/call", map[string]any{"name": "list_octop_experts", "arguments": map[string]any{}})
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
						list = append(list, Expert{ExpertID: Str(m["expert_id"]), Name: Str(m["name"])})
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
	out, err := c.MCPCall(2, "tools/call", map[string]any{
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
			return Str(sc["response"]), Str(sc["thread_id"]), nil
		}
	}
	return "", "", fmt.Errorf("未识别的响应结构")
}

// ResumeExpert 续thread: 官方协议是 call_octop_expert带thread_id
// (resume_octop_expert只管HITL挂起态, fileops是文本协议thread已完结, 不走它)
func (c *OctopClient) ResumeExpert(expertID, threadID, message string) (answer, threadID2 string, err error) {
	out, err := c.MCPCall(3, "tools/call", map[string]any{
		"name": "call_octop_expert",
		"arguments": map[string]any{
			"expert_id": expertID,
			"thread_id": threadID,
			"message":   message,
		},
	})
	if err != nil {
		return "", "", err
	}
	if res, ok := out["result"].(map[string]any); ok {
		if sc, ok := res["structuredContent"].(map[string]any); ok {
			if e, ok := sc["error"].(string); ok && e != "" {
				return "", "", fmt.Errorf("续答失败: %s", e)
			}
			return Str(sc["response"]), Str(sc["thread_id"]), nil
		}
	}
	return "", "", fmt.Errorf("未识别的响应结构")
}

func Str(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// version 由构建时 -ldflags 注入
var version = "dev"

// fetchUsageSummary 查当前用户token用量(官方 /api/usage/summary)
func (c *OctopClient) UsageSummary() (string, error) {
	req, _ := http.NewRequest("GET", c.BaseURL+"/api/usage/summary?window=all&granularity=total", nil)
	req.Header.Set("Authorization", "Bearer "+c.Token)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("usage HTTP %d: %s", resp.StatusCode, string(body))
	}
	var u struct {
		Window           string `json:"window"`
		InputTokens      int64  `json:"input_tokens"`
		UncachedInput    int64  `json:"uncached_input_tokens"`
		CacheRead        int64  `json:"cache_read_tokens"`
		OutputTokens     int64  `json:"output_tokens"`
		TotalTokens      int64  `json:"total_tokens"`
		ModelCalls       int    `json:"model_calls"`
		Turns            int    `json:"turns"`
		AvgPerTurn       int64  `json:"avg_per_turn"`
		CacheHitPercent  int    `json:"cache_hit_percent"`
	}
	if err := json.Unmarshal(body, &u); err != nil {
		return "", err
	}
	cacheNote := ""
	if u.CacheRead > 0 {
		cacheNote = fmt.Sprintf(" (缓存命中%d%%, 未重复计费的输入%d)", u.CacheHitPercent, u.UncachedInput)
	}
	return fmt.Sprintf("📊 我的token账单(累计)\n  总量: %s\n  输入: %s / 输出: %s%s\n  调用: %d次 · 对话轮: %d轮 · 平均每轮: %s",
		humanNum(u.TotalTokens), humanNum(u.InputTokens), humanNum(u.OutputTokens), cacheNote,
		u.ModelCalls, u.Turns, humanNum(u.AvgPerTurn)), nil
}

// humanNum 大数加逗号
func humanNum(n int64) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var out []byte
	for i, b := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, b)
	}
	return string(out)
}

// ---- 专家实例管理 (官方 /api/agents CRUD, Octop 1.0.2 自带) ----

// AgentsRaw 返回 /api/agents 原始列表(含 state/config, 供管理页渲染)
func (c *OctopClient) AgentsRaw() ([]map[string]any, error) {
	req, _ := http.NewRequest("GET", c.BaseURL+"/api/agents", nil)
	req.Header.Set("Authorization", "Bearer "+c.Token)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GET /api/agents -> %d: %s", resp.StatusCode, truncate(data))
	}
	var out []map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// doAgentAction 调 agents 的无返回体动作端点(start/stop/reload/delete)
func (c *OctopClient) doAgentAction(method, path string) error {
	req, err := http.NewRequest(method, c.BaseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s %s -> %d: %s", method, path, resp.StatusCode, truncate(data))
	}
	return nil
}

func (c *OctopClient) StartAgent(id string) error  { return c.doAgentAction("POST", "/api/agents/"+id+"/start") }
func (c *OctopClient) StopAgent(id string) error   { return c.doAgentAction("POST", "/api/agents/"+id+"/stop") }
func (c *OctopClient) DeleteAgent(id string) error { return c.doAgentAction("DELETE", "/api/agents/"+id) }

// CreateAgent 新建专家(name/描述/人设提示词/欢迎语/是否共享)
func (c *OctopClient) CreateAgent(name, description, systemPrompt, welcome string, shared bool) (map[string]any, error) {
	body := map[string]any{
		"name": name, "description": description,
		"system_prompt": systemPrompt, "welcome_message": welcome,
		"is_shared": shared,
	}
	j, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", c.BaseURL+"/api/agents", bytes.NewReader(j))
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("POST /api/agents -> %d: %s", resp.StatusCode, truncate(data))
	}
	var out map[string]any
	json.Unmarshal(data, &out)
	return out, nil
}

// UpdateAgent 改专家(人设提示词/简介/欢迎语)
func (c *OctopClient) UpdateAgent(id string, fields map[string]any) error {
	j, _ := json.Marshal(fields)
	req, _ := http.NewRequest("PATCH", c.BaseURL+"/api/agents/"+id, bytes.NewReader(j))
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("PATCH /api/agents/%s -> %d: %s", id, resp.StatusCode, truncate(data))
	}
	return nil
}

// truncate 截断错误响应体, 防错误日志过长
func truncate(b []byte) string {
	s := string(b)
	if len(s) > 300 {
		return s[:300]
	}
	return s
}

// ---- 技能库 (官方 /api/agents/{id}/skills, Octop 1.0.2 自带) ----

// ListSkills 拉某专家的技能列表(内置+workspace)
func (c *OctopClient) ListSkills(agentID string) ([]map[string]any, error) {
	req, _ := http.NewRequest("GET", c.BaseURL+"/api/agents/"+agentID+"/skills", nil)
	req.Header.Set("Authorization", "Bearer "+c.Token)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GET skills -> %d: %s", resp.StatusCode, truncate(data))
	}
	var out []map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// skillMD 组装 SKILL.md (frontmatter: name/description + 正文)
func skillMD(slug, description, body string) string {
	return "---\nname: " + slug + "\ndescription: " + description + "\n---\n\n" + body + "\n"
}

// CreateSkill 给专家新建一个技能(SKILL.md 形态, 与官方ZIP导入同构)
func (c *OctopClient) CreateSkill(agentID, slug, description, body string) error {
	payload := map[string]any{
		"name":    slug,
		"content": skillMD(slug, description, body),
	}
	j, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", c.BaseURL+"/api/agents/"+agentID+"/skills", bytes.NewReader(j))
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("POST skill -> %d: %s", resp.StatusCode, truncate(data))
	}
	return nil
}

// DeleteSkill 删除专家技能(软删, 官方204)
func (c *OctopClient) DeleteSkill(agentID, slug string) error {
	req, _ := http.NewRequest("DELETE", c.BaseURL+"/api/agents/"+agentID+"/skills/"+slug, nil)
	req.Header.Set("Authorization", "Bearer "+c.Token)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 && resp.StatusCode != 204 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("DELETE skill -> %d: %s", resp.StatusCode, truncate(data))
	}
	return nil
}

// ---- 用户管理 + 模型供应商(官方 /api/users, /api/admin/providers) ----

// UsersRaw 用户全量列表(角色/锁定/配额)
func (c *OctopClient) UsersRaw() ([]map[string]any, error) {
	req, _ := http.NewRequest("GET", c.BaseURL+"/api/users", nil)
	req.Header.Set("Authorization", "Bearer "+c.Token)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GET /api/users -> %d: %s", resp.StatusCode, truncate(data))
	}
	var out []map[string]any
	json.Unmarshal(data, &out)
	return out, nil
}

// doJSON 通用JSON请求(带body或nil)
func (c *OctopClient) doJSON(method, path string, body any) (int, []byte, error) {
	var rd io.Reader
	if body != nil {
		j, _ := json.Marshal(body)
		rd = bytes.NewReader(j)
	}
	req, err := http.NewRequest(method, c.BaseURL+path, rd)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, data, nil
}

// UserPatch 改用户(role/disabled/permissions/display_name)
func (c *OctopClient) UserPatch(id int, fields map[string]any) error {
	code, data, err := c.doJSON("PATCH", fmt.Sprintf("/api/users/%d", id), fields)
	if err != nil {
		return err
	}
	if code >= 300 {
		return fmt.Errorf("PATCH user -> %d: %s", code, truncate(data))
	}
	return nil
}

// UserUnlock 清除登录锁定
func (c *OctopClient) UserUnlock(id int) error {
	code, data, err := c.doJSON("POST", fmt.Sprintf("/api/users/%d/unlock-login", id), nil)
	if err != nil {
		return err
	}
	if code >= 300 {
		return fmt.Errorf("unlock -> %d: %s", code, truncate(data))
	}
	return nil
}

// ProvidersRaw 供应商列表(key服务端已脱敏)
func (c *OctopClient) ProvidersRaw() ([]map[string]any, error) {
	code, data, err := c.doJSON("GET", "/api/admin/providers", nil)
	if err != nil {
		return nil, err
	}
	if code != 200 {
		return nil, fmt.Errorf("GET providers -> %d: %s", code, truncate(data))
	}
	var out []map[string]any
	json.Unmarshal(data, &out)
	return out, nil
}

// ProviderCreate 新增供应商(name/kind/base_url/api_key/models)
func (c *OctopClient) ProviderCreate(name, kind, baseURL, apiKey string, models []map[string]string) (map[string]any, error) {
	body := map[string]any{"name": name, "kind": kind, "base_url": baseURL, "api_key": apiKey, "enabled": true}
	if len(models) > 0 {
		body["models"] = models
	}
	code, data, err := c.doJSON("POST", "/api/admin/providers", body)
	if err != nil {
		return nil, err
	}
	if code >= 300 {
		return nil, fmt.Errorf("POST provider -> %d: %s", code, truncate(data))
	}
	var out map[string]any
	json.Unmarshal(data, &out)
	return out, nil
}

// ProviderPatch 改供应商(api_key/base_url/enabled/models)
func (c *OctopClient) ProviderPatch(id int, fields map[string]any) error {
	code, data, err := c.doJSON("PATCH", fmt.Sprintf("/api/admin/providers/%d", id), fields)
	if err != nil {
		return err
	}
	if code >= 300 {
		return fmt.Errorf("PATCH provider -> %d: %s", code, truncate(data))
	}
	return nil
}

// ProviderDelete 删供应商(本地runtime供应商会被服务端拒)
func (c *OctopClient) ProviderDelete(id int) error {
	code, data, err := c.doJSON("DELETE", fmt.Sprintf("/api/admin/providers/%d", id), nil)
	if err != nil {
		return err
	}
	if code >= 300 {
		return fmt.Errorf("DELETE provider -> %d: %s", code, truncate(data))
	}
	return nil
}
