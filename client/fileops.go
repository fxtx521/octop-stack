package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileOp 专家返回的结构化文件操作(通过"协议头"约定的JSON)
type FileOp struct {
	Op     string `json:"op"`     // read|write|append|delete|move
	Path   string `json:"path"`   // 工作区相对路径
	To     string `json:"to"`     // move目标(相对路径)
	Body   string `json:"body"`   // write/append内容
	Reason string `json:"reason"` // 一句话理由(给用户看)
}

// ParseOps 从专家回答文本中提取操作计划
// 约定: 专家在回答中用 ```fileops 代码块包裹JSON数组
func ParseOps(answer string) []FileOp {
	marker := "```fileops"
	idx := strings.Index(answer, marker)
	if idx < 0 {
		return nil
	}
	rest := answer[idx+len(marker):]
	end := strings.Index(rest, "```")
	if end < 0 {
		rest = strings.TrimRight(rest, " \n\r")
	} else {
		rest = rest[:end]
	}
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return nil
	}
	var ops []FileOp
	if err := json.Unmarshal([]byte(rest), &ops); err != nil {
		return nil // 解析失败按无操作处理, 不中断对话
	}
	// 只认白名单操作
	var valid []FileOp
	for _, op := range ops {
		switch op.Op {
		case "read", "write", "append", "delete", "move":
			valid = append(valid, op)
		}
	}
	return valid
}

// OpResult 单个操作的执行结果
type OpResult struct {
	Op      string
	Path    string
	Status  string // done|rejected|error
	Detail  string
	Content string // read成功时的文件内容
}

// ExecuteOps 在沙箱内执行操作计划(每个操作独立校验)
func ExecuteOps(rootEval string, ops []FileOp, approved bool) []OpResult {
	var results []OpResult
	for _, op := range ops {
		r := OpResult{Op: op.Op, Path: op.Path}
		full, err := resolveInWS(rootEval, op.Path)
		if err != nil {
			r.Status, r.Detail = "rejected", err.Error()
			results = append(results, r)
			continue
		}
		if !approved {
			r.Status, r.Detail = "rejected", "未获用户批准"
			results = append(results, r)
			continue
		}
		switch op.Op {
		case "read":
			data, err := os.ReadFile(full)
			if err != nil {
				r.Status, r.Detail = "error", err.Error()
				break
			}
			r.Status = "done"
			r.Detail = fmt.Sprintf("已读%d字节", len(data))
			r.Content = string(data)
		case "write":
			if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
				r.Status, r.Detail = "error", err.Error()
				break
			}
			if err := os.WriteFile(full, []byte(op.Body), 0o644); err != nil {
				r.Status, r.Detail = "error", err.Error()
				break
			}
			r.Status, r.Detail = "done", fmt.Sprintf("已写入%d字节", len(op.Body))
		case "append":
			f, err := os.OpenFile(full, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
			if err != nil {
				r.Status, r.Detail = "error", err.Error()
				break
			}
			if _, err := f.WriteString(op.Body); err != nil {
				f.Close()
				r.Status, r.Detail = "error", err.Error()
				break
			}
			f.Close()
			r.Status = "done"
			r.Detail = fmt.Sprintf("已追加%d字节", len(op.Body))
		case "delete":
			if err := os.Remove(full); err != nil {
				r.Status, r.Detail = "error", err.Error()
				break
			}
			r.Status = "done"
			r.Detail = "已删除"
		case "move":
			dst, err := resolveInWS(rootEval, op.To)
			if err != nil {
				r.Status, r.Detail = "rejected", "目标路径: " + err.Error()
				break
			}
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				r.Status, r.Detail = "error", err.Error()
				break
			}
			if err := os.Rename(full, dst); err != nil {
				r.Status, r.Detail = "error", err.Error()
				break
			}
			r.Status = "done"
			r.Detail = "已移动到 " + op.To
		}
		results = append(results, r)
	}
	return results
}

// FormatOpsPlan 给用户看的审批提示
func FormatOpsPlan(ops []FileOp) string {
	var b strings.Builder
	b.WriteString("专家请求以下本地文件操作:\n")
	for i, op := range ops {
		switch op.Op {
		case "read":
			fmt.Fprintf(&b, "  %d) 读取 %s\n", i+1, op.Path)
		case "write":
			fmt.Fprintf(&b, "  %d) 写入 %s (%d字节)\n", i+1, op.Path, len(op.Body))
		case "append":
			fmt.Fprintf(&b, "  %d) 追加 %s (%d字节)\n", i+1, op.Path, len(op.Body))
		case "delete":
			fmt.Fprintf(&b, "  %d) 删除 %s\n", i+1, op.Path)
		case "move":
			fmt.Fprintf(&b, "  %d) 移动 %s -> %s\n", i+1, op.Path, op.To)
		}
		if op.Reason != "" {
			fmt.Fprintf(&b, "     理由: %s\n", op.Reason)
		}
	}
	b.WriteString("批准执行吗? (y=全部执行 / n=全部拒绝)")
	return b.String()
}

// auditLog 审计留痕: 每个操作(含拒绝)落盘到工作区内 .octop-audit.log
func auditLog(wsRoot string, user, expertID string, ops []FileOp, results []OpResult) {
	f, err := os.OpenFile(filepath.Join(wsRoot, ".octop-audit.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	for i, op := range ops {
		status, detail := "rejected", ""
		if i < len(results) {
			status, detail = results[i].Status, results[i].Detail
		}
		fmt.Fprintf(f, "%s user=%s expert=%s op=%s path=%s to=%s status=%s detail=%s\n",
			time.Now().Format(time.RFC3339), user, expertID, op.Op, op.Path, op.To, status, detail)
	}
}

// readUserConfirm 终端读y/n
func readUserConfirm(reader *bufio.Reader) bool {
	line, _ := reader.ReadString('\n')
	line = strings.ToLower(strings.TrimSpace(line))
	return line == "y" || line == "yes"
}

var _ = io.Discard // 保留io引用防止误删import
