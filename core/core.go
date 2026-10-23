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
