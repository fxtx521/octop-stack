package core

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// B档: 命令执行(审批制+黑名单+留痕)。默认关闭, 用户显式开启。
// 铁律: 即使已批准, 命中黑名单也拒绝; 每条命令(含拒绝)落审计日志。

type ShOp struct {
	Cmd    string `json:"cmd"`
	Reason string `json:"reason"`
}

type ShResult struct {
	Cmd    string `json:"cmd"`
	Status string `json:"status"` // done|rejected|error|timeout
	Detail string `json:"detail"`
	Output string `json:"output"`
}

// shBlacklist 高危模式: 命中即拒(双重保险, 不依赖用户审批)
var shBlacklist = []string{
	"rm -rf /", "rm -fr /", "mkfs", ":(){ :|:& };:",
	"shutdown", "reboot", "halt", "dd if=", "dd of=/dev/",
	"diskutil erase", "> /dev/sda", "/dev/disk0",
	"chmod -R 777 /", "curl | sh", "curl | bash", "wget | sh", "wget | bash",
	"history -c", "launchctl unload", "nvram ", "csrutil",
}

// ParseShOps 从回答里抽 ```shops 代码块(与fileops同构)
func ParseShOps(answer string) []ShOp {
	marker := "```shops"
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
	var ops []ShOp
	if err := json.Unmarshal([]byte(rest), &ops); err != nil {
		return nil
	}
	return ops
}

func shBlocked(cmd string) string {
	low := strings.ToLower(cmd)
	for _, b := range shBlacklist {
		if strings.Contains(low, strings.ToLower(b)) {
			return b
		}
	}
	return ""
}

// ExecuteShOps 执行命令计划: 未批准/黑名单/超时都有明确状态, 全部留痕
func ExecuteShOps(wsRoot, user, expertID string, ops []ShOp, approved bool, timeout time.Duration) []ShResult {
	results := make([]ShResult, 0, len(ops))
	for _, op := range ops {
		r := ShResult{Cmd: op.Cmd, Status: "rejected"}
		switch {
		case strings.TrimSpace(op.Cmd) == "":
			r.Detail = "空命令"
		case !approved:
			r.Detail = "未获用户批准"
		case shBlocked(op.Cmd) != "":
			r.Detail = "命中高危黑名单: " + shBlocked(op.Cmd)
		default:
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			cmd := exec.CommandContext(ctx, "sh", "-c", op.Cmd)
			out, err := cmd.CombinedOutput()
			cancel()
			if ctx.Err() == context.DeadlineExceeded {
				r.Status = "timeout"
				r.Detail = fmt.Sprintf("超时(%s)", timeout)
				r.Output = truncateStr(string(out), 4096)
			} else if err != nil {
				r.Status = "error"
				r.Detail = err.Error()
				r.Output = truncateStr(string(out), 4096)
			} else {
				r.Status = "done"
				r.Output = truncateStr(string(out), 4096)
			}
		}
		results = append(results, r)
	}
	shAuditLog(wsRoot, user, expertID, ops, results)
	return results
}

// shAuditLog 命令审计: 落工作区 .octop-audit.log(op=sh cmd=全文)
func shAuditLog(wsRoot, user, expertID string, ops []ShOp, results []ShResult) {
	if wsRoot == "" {
		return
	}
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
		fmt.Fprintf(f, "%s user=%s expert=%s op=sh cmd=%q status=%s detail=%s\n",
			time.Now().Format(time.RFC3339), user, expertID, op.Cmd, status, detail)
	}
}

func truncateStr(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
