package main

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// 沙箱铁律: 所有路径必须落在工作区内, 拒绝绝对路径/..逃逸/symlink逃逸

var (
	errAbsPath    = errors.New("拒绝绝对路径")
	errDotDot     = errors.New("拒绝..路径")
	errEscape     = errors.New("路径逃逸出工作区")
	errSymlinkIn  = errors.New("工作区内含符号链接, 拒绝跨出")
	manifestLimit = 200   // 最多列200个文件
	manifestBytes = 8000  // 清单字符预算
)

// workspaceRootEval 把用户给的工作区路径解析成真实路径(symlink展开后的锚点)
func workspaceRootEval(root string) (string, error) {
	ev, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("工作区不存在: %w", err)
	}
	return ev, nil
}

// resolveInWS 把相对路径安全解析到工作区内
// 检查: 非绝对 / 无.. / 已存在部分的symlink展开后仍在工作区内
func resolveInWS(rootEval, rel string) (string, error) {
	rel = strings.TrimSpace(rel)
	if rel == "" || rel == "." {
		return "", errors.New("空路径")
	}
	if filepath.IsAbs(rel) {
		return "", errAbsPath
	}
	for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
		if part == ".." {
			return "", errDotDot
		}
	}
	clean := filepath.Clean(rel)
	p := filepath.Join(rootEval, clean)

	// 从最深已存在祖先做symlink展开校验
	cur, suffix := p, ""
	for {
		ev, err := filepath.EvalSymlinks(cur)
		if err == nil {
			final := filepath.Join(ev, suffix)
			if final != rootEval && !strings.HasPrefix(final, rootEval+string(filepath.Separator)) {
				return "", errEscape
			}
			return final, nil
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return "", errEscape
		}
		suffix = filepath.Join(filepath.Base(cur), suffix)
		cur = parent
	}
}

// buildManifest 生成工作区文件清单(供专家了解现场), 跳过隐藏文件和符号链接
func buildManifest(rootEval string) (string, error) {
	var lines []string
	var skipped, count int
	var total int
	err := filepath.WalkDir(rootEval, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // 单点读不动就跳过
		}
		rel, _ := filepath.Rel(rootEval, path)
		name := d.Name()
		if strings.HasPrefix(name, ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			skipped++
			return nil
		}
		if d.IsDir() {
			return nil
		}
		count++
		if len(lines) >= manifestLimit {
			return nil
		}
		info, _ := d.Info()
		sz := int64(0)
		if info != nil {
			sz = info.Size()
		}
		line := fmt.Sprintf("%s (%dB)", filepath.ToSlash(rel), sz)
		total += len(line)
		if total > manifestBytes {
			return filepath.SkipAll
		}
		lines = append(lines, line)
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(lines)
	var b strings.Builder
	b.WriteString(fmt.Sprintf("文件共%d个", count))
	if skipped > 0 {
		b.WriteString(fmt.Sprintf("(跳过%d个符号链接)", skipped))
	}
	b.WriteString(":\n")
	for _, l := range lines {
		b.WriteString(l + "\n")
	}
	if count > len(lines) {
		b.WriteString(fmt.Sprintf("...其余%d个未列出\n", count-len(lines)))
	}
	return b.String(), nil
}
