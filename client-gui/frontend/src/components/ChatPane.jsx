import React, { useEffect, useRef, useState } from 'react'
import { api } from '../lib/octop.js'

const SUGGESTIONS = [
  { icon: '📁', label: '整理这个文件夹', hint: '归类散落的文件' },
  { icon: '📊', label: '分析表格数据', hint: '读取并汇总' },
  { icon: '📝', label: '写一份周报', hint: '从工作记录提炼' },
  { icon: '🔍', label: '查资料做研究', hint: '联网搜索整理' },
]

export default function ChatPane({ user, task, patchTask, onOpenFiles }) {
  const [input, setInput] = useState('')
  const scrollRef = useRef(null)

  useEffect(() => {
    scrollRef.current?.scrollTo({ top: 999999, behavior: 'smooth' })
  }, [task?.msgs?.length, task?.pending?.length])

  if (!task) return <div className="chat-empty-state"><div className="chat-empty-hint">选择或新建一个任务</div></div>

  async function send(text) {
    const content = (text ?? input).trim()
    if (!content || task.busy) return
    setInput('')
    const msgs = [...task.msgs, { role: 'user', text: content }]
    patchTask(task.id, { msgs, busy: true, title: task.title === '新任务' ? content.slice(0, 18) : task.title })
    try {
      const r = await api.ask(content)
      if (!r.ok) {
        patchTask(task.id, { busy: false, msgs: [...msgs, { role: 'sys', text: r.msg || '请求失败' }] })
        return
      }
      const add = []
      if (r.answer) add.push({ role: 'ai', text: r.answer })
      patchTask(task.id, {
        busy: false,
        msgs: [...msgs, ...add],
        pending: (r.ops && r.ops.length ? r.ops : []),
      })
    } catch (e) {
      patchTask(task.id, { busy: false, msgs: [...msgs, { role: 'sys', text: '网络错误: ' + e }] })
    }
  }

  async function approve(ops, ok) {
    patchTask(task.id, {
      msgs: [...task.msgs, { role: 'sys', text: (ok ? '已批准' : '已拒绝') + ' ' + ops.length + ' 个文件操作' }],
      pending: [],
      busy: true,
    })
    try {
      const r = await api.apply(ops, ok)
      const add = []
      if (r.results?.length) {
        add.push({ role: 'sys', text: r.results.map(x => x.op + ' ' + x.path + ' → ' + x.status).join('\n') })
      }
      // 执行后刷新文件树(写/删/移都会改变目录内容)
      let files = task.files
      try { files = (await api.listFiles()) || [] } catch { /* ignore */ }
      if (ok && r.feedback) {
        // 把读取到的内容喂回专家, 让它继续任务
        const r2 = await api.resume('[客户端已执行] 结果如下, 请继续你的任务(还需操作文件继续输出fileops):\n' + r.feedback)
        if (r2.answer) add.push({ role: 'ai', text: r2.answer })
        patchTask(task.id, { busy: false, msgs: [...task.msgs, ...add], files, pending: r2.ops?.length ? r2.ops : [] })
        return
      }
      patchTask(task.id, { busy: false, msgs: [...task.msgs, ...add], files })
    } catch (e) {
      patchTask(task.id, { busy: false, msgs: [...task.msgs, { role: 'sys', text: '执行错误: ' + e }] })
    }
  }

  async function pickWorkspace() {
    const dir = await api.pickDirectory()
    if (dir) {
      await api.setWorkspace(dir)
      patchTask(task.id, { workspace: dir })
    }
  }

  const empty = task.msgs.length === 0

  return (
    <main className="chatpane">
      <header className="chat-top">
        <span className="chat-title">{task.title}</span>
        <select className="expert-select" value={task.expert}
          onChange={e => patchTask(task.id, { expert: e.target.value })}>
          {user.experts?.map(x => <option key={x.id} value={x.id}>{x.name}</option>)}
        </select>
        <div className="spacer" />
        <button className="ghost-btn" onClick={pickWorkspace}>
          {task.workspace ? '📁 ' + shortPath(task.workspace) : '📁 选择工作文件夹'}
        </button>
      </header>

      {empty ? (
        <div className="hero">
          <div className="hero-title">Octop<span className="dot">.</span> 我帮你</div>
          <div className="hero-sub">把活交给我，你只管确认结果</div>
          <div className="hero-input">
            <input
              value={input}
              onChange={e => setInput(e.target.value)}
              onKeyDown={e => e.key === 'Enter' && send()}
              placeholder="描述你要做的事，回车发送…"
              autoFocus
            />
            <button onClick={() => send()} disabled={task.busy || !input.trim()}>发送</button>
          </div>
          <div className="hero-tip">💡 建议先点右上角「选择工作文件夹」，告诉我能碰哪个目录</div>
          <div className="hero-sugs">
            {SUGGESTIONS.map(s => (
              <button key={s.label} className="sug" onClick={() => send(s.label)}>
                <span className="sug-icon">{s.icon}</span>
                <span className="sug-label">{s.label}</span>
                <span className="sug-hint">{s.hint}</span>
              </button>
            ))}
          </div>
        </div>
      ) : (
        <>
          <div className="stream" ref={scrollRef}>
            {task.msgs.map((m, i) => <Msg key={i} m={m} />)}
            {task.pending.map((op, idx) => (
              <ApprovalCard key={'ap' + idx} op={op} idx={idx} ops={task.pending} onDecide={approve} />
            ))}
            {task.busy && <div className="typing"><span /><span /><span /></div>}
          </div>
          <div className="inputbar">
            <input
              value={input}
              onChange={e => setInput(e.target.value)}
              onKeyDown={e => e.key === 'Enter' && send()}
              placeholder={task.busy ? '它正在干活…' : '继续对话…'}
              disabled={task.busy}
            />
            <button onClick={() => send()} disabled={task.busy || !input.trim()}>发送</button>
          </div>
        </>
      )}
    </main>
  )
}

function Msg({ m }) {
  if (m.role === 'sys') return <div className="msg-sys">{m.text}</div>
  return (
    <div className={'msg ' + m.role}>
      <div className="msg-bubble">{m.text}</div>
    </div>
  )
}

const OP_META = {
  read: { icon: '👁', label: '读取文件' },
  write: { icon: '✏️', label: '写入文件' },
  append: { icon: '➕', label: '追加内容' },
  delete: { icon: '🗑', label: '删除文件' },
  move: { icon: '📦', label: '移动文件' },
}

function ApprovalCard({ op, idx, ops, onDecide }) {
  const meta = OP_META[op.op] || { icon: '🔧', label: op.op }
  const isLast = idx === ops.length - 1
  return (
    <div className="approve-card">
      <div className="approve-head">
        <span className="approve-icon">{meta.icon}</span>
        <span className="approve-title">{meta.label}{ops.length > 1 ? ` (${idx + 1}/${ops.length})` : ''} — 需要你批准</span>
      </div>
      <div className="approve-path">{op.path}{op.to ? ' → ' + op.to : ''}</div>
      {op.reason && <div className="approve-reason">{op.reason}</div>}
      {isLast && (
        <div className="approve-actions">
          <button className="btn-ok" onClick={() => onDecide(ops, true)}>全部批准</button>
          <button className="btn-no" onClick={() => onDecide(ops, false)}>拒绝</button>
        </div>
      )}
    </div>
  )
}

function shortPath(p) {
  if (!p) return ''
  const parts = p.split('/')
  return parts.length > 2 ? '…/' + parts.slice(-2).join('/') : p
}
