import React, { useState } from 'react'

// 通用文件树: 递归目录, 可展开/收起, 点击文件回调
export default function FileTree({ files, onOpen, defaultDepth = 1 }) {
  const [expanded, setExpanded] = useState({})
  const nodes = normalize(files)

  if (!nodes.length) return <div className="tree-empty">点 ⟳ 加载文件列表</div>
  return (
    <div className="tree">
      {nodes.map(n => (
        <Node key={n.path} node={n} depth={0}
          expanded={expanded} setExpanded={setExpanded}
          onOpen={onOpen} />
      ))}
    </div>
  )
}

function Node({ node, depth, expanded, setExpanded, onOpen }) {
  const isOpen = !!expanded[node.path]
  const isDir = node.type === 'dir'
  return (
    <>
      <div
        className={'tree-row' + (isDir ? ' dir' : '')}
        style={{ paddingLeft: 8 + depth * 14 }}
        onClick={() => isDir
          ? setExpanded(e => ({ ...e, [node.path]: !isOpen }))
          : onOpen(node.path)}
      >
        <span className="tree-caret">{isDir ? (isOpen ? '▾' : '▸') : ''}</span>
        <span className="tree-icon">{isDir ? (isOpen ? '📂' : '📁') : iconByExt(node.name)}</span>
        <span className="tree-name">{node.name}</span>
      </div>
      {isDir && isOpen && (node.children || []).map(c => (
        <Node key={c.path} node={c} depth={depth + 1}
          expanded={expanded} setExpanded={setExpanded} onOpen={onOpen} />
      ))}
    </>
  )
}

function iconByExt(name) {
  const ext = (name.split('.').pop() || '').toLowerCase()
  if (['md', 'txt', 'doc', 'docx'].includes(ext)) return '📄'
  if (['xlsx', 'xls', 'csv'].includes(ext)) return '📊'
  if (['png', 'jpg', 'jpeg', 'gif', 'webp'].includes(ext)) return '🖼'
  if (['pdf'].includes(ext)) return '📕'
  if (['js', 'ts', 'py', 'go', 'json', 'sh'].includes(ext)) return '⚙️'
  return '📄'
}

// 兼容: 后端可能返回 {tree:[...]} 或扁平路径数组
function normalize(files) {
  if (!files || !files.length) return []
  if (typeof files[0] === 'string') {
    // 扁平路径 → 树
    const root = []
    const dirs = {}
    for (const p of files) {
      const parts = p.split('/').filter(Boolean)
      let cur = root, acc = ''
      for (let i = 0; i < parts.length; i++) {
        acc = acc ? acc + '/' + parts[i] : parts[i]
        const last = i === parts.length - 1
        let hit = cur.find(x => x.name === parts[i])
        if (!hit) {
          hit = { name: parts[i], path: acc, type: last ? 'file' : 'dir', children: [] }
          cur.push(hit)
        }
        cur = hit.children
      }
    }
    return root
  }
  return files.map(f => ({
    name: f.name,
    path: f.path || f.name,
    type: f.type || (f.isDir ? 'dir' : 'file'),
    children: f.children ? normalize(f.children) : [],
  }))
}
