package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// makeWS 建一个临时工作区
func makeWS(t *testing.T) (dir string, eval string) {
	t.Helper()
	dir = t.TempDir()
	ev, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	os.MkdirAll(filepath.Join(dir, "sub"), 0o755)
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello"), 0o644)
	os.WriteFile(filepath.Join(dir, "sub", "b.md"), []byte("# B"), 0o644)
	return dir, ev
}

func TestResolveRejectsEscape(t *testing.T) {
	_, eval := makeWS(t)
	cases := []struct{ name, path, wantErr string }{
		{"绝对路径", "/etc/passwd", "拒绝绝对路径"},
		{"点点逃逸", "../outside.txt", "拒绝.."},
		{"嵌套点点", "sub/../../x", "拒绝.."},
		{"点斜杠绝对", "/tmp/x", "拒绝绝对路径"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := resolveInWS(eval, c.path)
			if err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("%s 应被拒(%s), 实际: %v", c.path, c.wantErr, err)
			}
		})
	}
}

func TestResolveRejectsSymlinkEscape(t *testing.T) {
	dir, eval := makeWS(t)
	outside := t.TempDir()
	os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("s"), 0o644)
	// 工作区内放symlink指向外部
	os.Symlink(outside, filepath.Join(dir, "linkdir"))
	if _, err := resolveInWS(eval, "linkdir/secret.txt"); err == nil {
		t.Fatal("symlink逃逸必须被拒绝")
	}
}

func TestResolveAcceptsValid(t *testing.T) {
	_, eval := makeWS(t)
	// 注: 一切含..的路径都被拒(哪怕不出界), 这是刻意的从严设计, 便于审计
	for _, p := range []string{"a.txt", "sub/b.md", "sub/new.txt"} {
		if _, err := resolveInWS(eval, p); err != nil {
			t.Fatalf("合法路径 %s 被拒: %v", p, err)
		}
	}
}

func TestManifest(t *testing.T) {
	_, eval := makeWS(t)
	m, err := buildManifest(eval)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(m, "a.txt") || !strings.Contains(m, "sub/b.md") {
		t.Fatalf("清单缺文件: %s", m)
	}
	if !strings.Contains(m, "文件共2个") {
		t.Fatalf("清单计数错误: %s", m)
	}
}

func TestParseOps(t *testing.T) {
	ans := "我来看一下文件。\n```fileops\n[\n" +
		`  {"op":"read","path":"a.txt","reason":"查看内容"},
  {"op":"write","path":"sub/c.txt","body":"你好","reason":"创建新文件"},
  {"op":"hack","path":"x","reason":"非法操作"}
]` +
		"\n```\n请批准。"
	ops := ParseOps(ans)
	if len(ops) != 2 {
		t.Fatalf("应解析出2个白名单操作, 实际%d个: %+v", len(ops), ops)
	}
	if ops[0].Op != "read" || ops[1].Op != "write" {
		t.Fatalf("操作解析错误: %+v", ops)
	}
	// 无代码块
	if ParseOps("普通回答没有操作") != nil {
		t.Fatal("无fileops块应返回nil")
	}
	// JSON损坏
	bad := "```fileops\n{not json\n```"
	if ParseOps(bad) != nil {
		t.Fatal("损坏JSON应返回nil")
	}
}

func TestExecuteOpsSandbox(t *testing.T) {
	_, eval := makeWS(t)
	ops := []FileOp{
		{Op: "read", Path: "a.txt"},
		{Op: "write", Path: "sub/c.txt", Body: "新增内容"},
		{Op: "write", Path: "../evil.txt", Body: "逃逸"},
		{Op: "delete", Path: "/etc/passwd"},
		{Op: "move", Path: "a.txt", To: "sub/a-moved.txt"},
	}
	results := ExecuteOps(eval, ops, true)
	if len(results) != 5 {
		t.Fatalf("应5个结果: %+v", results)
	}
	if results[0].Status != "done" {
		t.Fatalf("read应done: %+v", results[0])
	}
	if results[1].Status != "done" {
		t.Fatalf("write应done: %+v", results[1])
	}
	if results[2].Status != "rejected" {
		t.Fatalf("逃逸write必须rejected: %+v", results[2])
	}
	if results[3].Status != "rejected" {
		t.Fatalf("绝对路径delete必须rejected: %+v", results[3])
	}
	if results[4].Status != "done" {
		t.Fatalf("move应done: %+v", results[4])
	}
	// 落盘验证
	if _, err := os.Stat(filepath.Join(eval, "sub", "c.txt")); err != nil {
		t.Fatal("c.txt未落盘")
	}
	if _, err := os.Stat(filepath.Join(eval, "sub", "a-moved.txt")); err != nil {
		t.Fatal("a-moved.txt未落盘")
	}
	if _, err := os.Stat(filepath.Join(eval, "a.txt")); !os.IsNotExist(err) {
		t.Fatal("a.txt应已被移走")
	}
	// 逃逸文件必须不存在
	if _, err := os.Stat(filepath.Join(filepath.Dir(eval), "evil.txt")); !os.IsNotExist(err) {
		t.Fatal("逃逸文件竟被写出!")
	}
}

func TestExecuteOpsNeedsApproval(t *testing.T) {
	_, eval := makeWS(t)
	ops := []FileOp{{Op: "write", Path: "d.txt", Body: "x"}}
	results := ExecuteOps(eval, ops, false)
	if results[0].Status != "rejected" || results[0].Detail != "未获用户批准" {
		t.Fatalf("未批准必须全拒: %+v", results[0])
	}
	if _, err := os.Stat(filepath.Join(eval, "d.txt")); !os.IsNotExist(err) {
		t.Fatal("未批准却写盘!")
	}
}

func TestFormatOpsPlan(t *testing.T) {
	ops := []FileOp{
		{Op: "write", Path: "a.txt", Body: "12345", Reason: "补充说明"},
		{Op: "delete", Path: "old.log"},
	}
	plan := FormatOpsPlan(ops)
	for _, want := range []string{"写入 a.txt (5字节)", "删除 old.log", "理由: 补充说明", "y=全部执行"} {
		if !strings.Contains(plan, want) {
			t.Fatalf("审批提示缺: %s\n%s", want, plan)
		}
	}
}

func TestAuditLogInWorkspace(t *testing.T) {
	_, eval := makeWS(t)
	ops := []FileOp{{Op: "write", Path: "x.txt", Body: "1"}}
	results := ExecuteOps(eval, ops, true)
	auditLog(eval, "tester", "E1", ops, results)
	data, err := os.ReadFile(filepath.Join(eval, ".octop-audit.log"))
	if err != nil {
		t.Fatal("审计日志应写在工作区内: ", err)
	}
	if !strings.Contains(string(data), "user=tester expert=E1 op=write") || !strings.Contains(string(data), "status=done") {
		t.Fatalf("审计内容不对: %s", data)
	}
}
