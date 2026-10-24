package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseShOps(t *testing.T) {
	ans := "好的\n```shops\n[{\"cmd\":\"ls -la\",\"reason\":\"看目录\"}]\n```"
	ops := ParseShOps(ans)
	if len(ops) != 1 || ops[0].Cmd != "ls -la" {
		t.Fatalf("parse失败: %+v", ops)
	}
	if ParseShOps("普通回答") != nil {
		t.Fatal("无代码块应返回nil")
	}
}

func TestShSandbox(t *testing.T) {
	ws := t.TempDir()
	// 1. 未批准拒绝
	r := ExecuteShOps(ws, "u", "e", []ShOp{{Cmd: "echo hi"}}, false, 5*time.Second)
	if r[0].Status != "rejected" {
		t.Fatalf("未批准应rejected: %+v", r)
	}
	// 2. 黑名单: 即使批准也拒
	r = ExecuteShOps(ws, "u", "e", []ShOp{{Cmd: "rm -rf /"}}, true, 5*time.Second)
	if r[0].Status != "rejected" || !strings.Contains(r[0].Detail, "黑名单") {
		t.Fatalf("黑名单应拒: %+v", r)
	}
	// 3. 正常执行
	r = ExecuteShOps(ws, "u", "e", []ShOp{{Cmd: "echo hello-b档"}}, true, 5*time.Second)
	if r[0].Status != "done" || !strings.Contains(r[0].Output, "hello-b档") {
		t.Fatalf("echo应done: %+v", r)
	}
	// 4. 超时
	r = ExecuteShOps(ws, "u", "e", []ShOp{{Cmd: "sleep 5"}}, true, 1*time.Second)
	if r[0].Status != "timeout" {
		t.Fatalf("应timeout: %+v", r)
	}
	// 5. 留痕: 含批准与拒绝全落审计
	data, _ := os.ReadFile(filepath.Join(ws, ".octop-audit.log"))
	s := string(data)
	if !strings.Contains(s, "op=sh") || !strings.Contains(s, "rm -rf /") || !strings.Contains(s, "echo hello-b档") {
		t.Fatalf("审计缺: %s", s)
	}
}
