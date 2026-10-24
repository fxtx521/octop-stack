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
  const taRef = useRef(null)
  const scrollRef = useRef(null)

  useEffect(() => {
    scrollRef.current?.scrollTo({ top: 999999, behavior: 'smooth' })
  }, [task?.msgs?.length, task?.pending?.length, task?.busy])

  // textarea 自适应高度
  useEffect(() => {
    const ta = taRef.current
    if (ta) { ta.style.height = 'auto'; ta.style.height = Math.min(ta.scrollHeight, 160) + 'px' }
  }, [input])

  if (!task) return <div className="chat"><div className="stub-body"><span className="big">◇</span><p>选择或新建一个任务</p></div></div>

  async function send(text) {
    const content = (text ?? input).trim()
    if (!content || task.busy) return
    setInput('')
    const msgs = [...task.msgs, { role: 'user', text: content }]
    patchTask(task.id, { msgs, busy: true, title: task.title === '新任务' ? content.slice(0, 18) : task.title })
    try {
      // thread 跟任务走: 有 thread 续聊(同任务同云端会话), 无则云端新建
      const r = await api.ask(task.expert, task.thread, content)
      if (!r.ok) {
        patchTask(task.id, { busy: false, msgs: [...msgs, { role: 'sys', text: r.msg || '请求失败' }] })
        return
      }
      const add = []
      if (r.answer) add.push({ role: 'ai', text: r.answer })
      patchTask(task.id, {
        busy: false,
        thread: r.thread || task.thread || '',
        msgs: [...msgs, ...add],
        pending: (r.ops && r.ops.length ? r.ops : []),
        pendingSh: (r.shops && r.shops.length ? r.shops : []),
      })
    } catch (e) {
      patchTask(task.id, { busy: false, msgs: [...msgs, { role: 'sys', text: '网络错误: ' + e }] })
    }
  }

  async function approve(ops, ok) {
    const isSh = !Array.isArray(ops) || (ops.length && ops[0].cmd !== undefined)
    patchTask(task.id, {
      msgs: [...task.msgs, { role: 'sys', text: (ok ? '已批准' : '已拒绝') + ' ' + ops.length + (isSh ? ' 条命令' : ' 个文件操作') }],
      pending: [],
      pendingSh: [],
      busy: true,
    })
    try {
      if (isSh) {
        const r = await api.applySh(ops, ok)
        const add = []
        if (r.results?.length) {
          add.push({ role: 'sys', text: r.results.map(x => 'sh ' + x.cmd.slice(0, 60) + ' → ' + x.status + (x.detail ? ' (' + x.detail.slice(0, 80) + ')' : '')).join('\n') })
        }
        if (ok && r.feedback) {
          const r2 = await api.resume(task.expert, task.thread, '[客户端已执行命令] 结果如下, 请继续你的任务(还需执行继续输出shops):\n' + r.feedback)
          if (r2.answer) add.push({ role: 'ai', text: r2.answer })
          patchTask(task.id, { busy: false, thread: r2.thread || task.thread || '', msgs: [...task.msgs, ...add], pending: r2.ops?.length ? r2.ops : [], pendingSh: r2.shops?.length ? r2.shops : [] })
          return
        }
        patchTask(task.id, { busy: false, msgs: [...task.msgs, ...add] })
        return
      }
      const r = await api.apply(ops, ok)
      const add = []
      if (r.results?.length) {
        add.push({ role: 'sys', text: r.results.map(x => x.op + ' ' + x.path + ' → ' + x.status).join('\n') })
      }
      // 执行后刷新文件树(写/删/移都会改变目录内容)
      let files = task.files
      try { files = (await api.listFiles()) || [] } catch { /* ignore */ }
      if (ok && r.feedback) {
        // 把读取到的内容喂回专家, 让它继续任务(同任务同 thread)
        const r2 = await api.resume(task.expert, task.thread, '[客户端已执行] 结果如下, 请继续你的任务(还需操作文件继续输出fileops):\n' + r.feedback)
        if (r2.answer) add.push({ role: 'ai', text: r2.answer })
        patchTask(task.id, { busy: false, thread: r2.thread || task.thread || '', msgs: [...task.msgs, ...add], files, pending: r2.ops?.length ? r2.ops : [] })
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

  const curExpert = user?.experts?.find(x => x.expert_id === task.expert)
  const empty = task.msgs.length === 0

  return (
    <main className="chat">
      <header className="chat-head">
        <span className="chat-title">{task.title}</span>
        <button className="chat-head-btn" onClick={pickWorkspace} title="选择工作文件夹">📁</button>
        <button className="chat-head-btn" onClick={onOpenFiles} title="工作区文件">▤</button>
      </header>

      <div className="chat-scroll" ref={scrollRef}>
        <div className="chat-inner">
          {empty && (
            <div className="hero">
              <h1>你好, 我来帮你<em>。</em></h1>
              <p>把活交给我，你只管确认结果。先选好工作文件夹，我就能动手。</p>
              <div className="hero-sugs">
                {SUGGESTIONS.map(s => (
                  <button key={s.label} className="sug-card" onClick={() => send(s.label)}>
                    <b>{s.icon} {s.label}</b>
                    <span>{s.hint}</span>
                  </button>
                ))}
              </div>
            </div>
          )}
          {task.msgs.map((m, i) => <Msg key={i} m={m} />)}
          {task.pending.map((op, idx) => (
            <ApprovalCard key={'ap' + idx} op={op} idx={idx} ops={task.pending} onDecide={approve} />
          ))}
          {task.pendingSh && task.pendingSh.length > 0 && (
            <ShCard ops={task.pendingSh} onDecide={approve} />
          )}
          {task.busy && <div className="msg-busy"><i /><i /><i /> 它正在干活…</div>}
        </div>
      </div>

      <div className="composer-wrap">
        <div className="composer">
          <textarea
            ref={taRef}
            className="composer-input"
            rows={1}
            value={input}
            onChange={e => setInput(e.target.value)}
            onKeyDown={e => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); send() } }}
            placeholder="描述你要做的事, 回车发送…"
            autoFocus
          />
          <div className="composer-bar">
            <span className="cp-model" title="当前专家">◇ {curExpert?.name || user?.experts?.[0]?.name || '默认专家'}</span>
            <button className="cp-btn" onClick={pickWorkspace}>
              {task.workspace ? '📁 ' + shortPath(task.workspace) : '📁 选择工作文件夹'}
            </button>
            <button
              className="cp-send"
              onClick={() => send()}
              disabled={task.busy || !input.trim()}
              title="发送"
            >↑</button>
          </div>
        </div>
        <div className="composer-note">内容由 AI 生成, 文件操作需你批准后才会执行</div>
      </div>
    </main>
  )
}

function Msg({ m }) {
  if (m.role === 'sys') return <div className="msg-sys">{m.text}</div>
  if (m.role === 'user') {
    return (
      <div className="msg msg-user">
        <div className="bubble">{m.text}</div>
      </div>
    )
  }
  return (
    <div className="msg msg-ai">
      <div className="ava">O</div>
      <div className="body"><div className="text">{stripFileops(m.text)}</div></div>
    </div>
  )
}

// fileops 提案已由审批卡呈现, 正文里剥离原始代码块
function stripFileops(t) {
  if (!t) return t
  const out = t.replace(/```fileops[\s\S]*?```/g, '').replace(/```\s*[\s\S]*?"op"[\s\S]*?```/g, '')
  return out.replace(/\n{3,}/g, '\n\n').trim()
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
    <div className="approval">
      <div className="ap-head">
        <b>{meta.label}{ops.length > 1 ? ` (${idx + 1}/${ops.length})` : ''}</b>
        <span>需要你批准</span>
      </div>
      <div className="ap-op">
        <span className="ap-icon">{meta.icon}</span>
        <span className="ap-path">{op.path}{op.to ? ' → ' + op.to : ''}</span>
        <span className="ap-idx">{op.path?.split('/').pop()?.split('.').pop()?.toUpperCase()}</span>
      </div>
      {op.reason && <div className="ap-reason">{op.reason}</div>}
      {isLast && (
        <div className="ap-actions">
          <button className="ap-btn ap-yes" onClick={() => onDecide(ops, true)}>全部批准</button>
          <button className="ap-btn ap-no" onClick={() => onDecide(ops, false)}>拒绝</button>
        </div>
      )}
    </div>
  )
}

function shortPath(p) {
  if (!p) return ''
  const parts = p.replace(/\/+$/, '').split('/')
  return parts.length > 2 ? '…/' + parts.slice(-2).join('/') : p
}

// 命令执行审批卡(B档): 展示完整命令+理由, 全批/全拒
function ShCard({ ops, onDecide }) {
  return (
    <div className="ap-card">
      <div className="ap-title">⚡ 专家想执行 {ops.length} 条系统命令(每条已留痕, 高危强制拒绝)</div>
      {ops.map((o, i) => (
        <div key={i} className="ap-item">
          <code className="ap-path">{'$ ' + o.cmd}</code>
          <div className="ap-desc">{o.reason || ''}</div>
        </div>
      ))}
      <div className="ap-btns">
        <button className="ap-btn" onClick={() => onDecide(ops, false)}>全部拒绝</button>
        <button className="ap-btn ap-yes" onClick={() => onDecide(ops, true)}>全部批准</button>
      </div>
    </div>
  )
}
