import React, { useEffect, useMemo, useState } from 'react'
import Login from './components/Login.jsx'
import Sidebar from './components/Sidebar.jsx'
import ChatPane from './components/ChatPane.jsx'
import RightPane from './components/RightPane.jsx'
import Experts from './components/Experts.jsx'
import { api } from './lib/octop.js'
import './app.css'

let taskSeq = 1

// 全局错误兜底: 任何未捕获异常显示红条, 不再静默吞错
let showFatal = () => {}
window.addEventListener('error', e => showFatal(e.message + '\n' + (e.filename || '') + ':' + (e.lineno || '')))
window.addEventListener('unhandledrejection', e => showFatal(String(e.reason)))

export default function App() {
  const [user, setUser] = useState(null)          // {role, experts, usage}
  const [fatal, setFatal] = useState('')
  useEffect(() => { showFatal = setFatal }, [])
  const [tasks, setTasks] = useState([])
  const [curId, setCurId] = useState(null)
  const [view, setView] = useState('chat')        // chat | files | experts | scheduled | library | admin
  const cur = useMemo(() => tasks.find(t => t.id === curId), [tasks, curId])

  function patchTask(id, patch) {
    setTasks(ts => ts.map(t => (t.id === id ? { ...t, ...patch } : t)))
  }

  function newTask() {
    const t = {
      id: 't' + Date.now(),
      seq: taskSeq++,
      title: '新任务',
      expert: user?.experts?.[0]?.expert_id || user?.experts?.[0]?.id || '',
      thread: '',    // 云端会话ID: 空=新会话, 首次Ask后绑定
      workspace: tasks[0]?.workspace || '',
      msgs: [],      // {role:'user|ai|sys', text, ops?, ts}
      pending: [],   // 审批队列
      files: null,   // 文件树缓存
      busy: false,
    }
    setTasks(ts => [t, ...ts])
    setCurId(t.id)
    setView('chat')
  }

  async function enterWorkspace(r, userName) {
    let ws = null
    try {
      const w = await api.startupWorkspace()
      if (w.ok) ws = w.root
    } catch { /* ignore */ }
    setUser({ name: userName, role: r.role || 'member', experts: r.experts || [], usage: r.usage || null })
    const t = {
      id: 't' + (taskSeq++),
      title: '新任务',
      msgs: [],
      pending: [],
      files: [],
      busy: false,
      workspace: ws,
    }
    if (ws) {
      try { t.files = (await api.listFiles()) || [] } catch { /* ignore */ }
    }
    setTasks([t])
    setCurId(t.id)
  }

  // 启动即尝试免登录续会话
  useEffect(() => {
    ;(async () => {
      try {
        const r = await api.tryResume()
        if (r.ok) await enterWorkspace(r, r.user || '已登录用户')
      } catch { /* 首次启动无会话, 走登录页 */ }
    })()
  }, [])

  async function doLogin(srv, u, p) {
    try {
      const r = await api.login(srv, u, p)
      if (r.ok) {
        await enterWorkspace(r, u)
      } else {
        throw new Error(r.msg || '登录失败')
      }
    } catch (e) {
      throw e
    }
  }

  function openTask(id) {
    setCurId(id)
    setView('chat')
  }

  function renameTask(id, title) {
    setTasks(ts => ts.map(t => (t.id === id ? { ...t, title } : t)))
  }

  function delTask(id) {
    setTasks(ts => {
      const left = ts.filter(t => t.id !== id)
      if (id === curId) setCurId(left[0]?.id || null)
      return left
    })
  }

  if (!user) return <Login onLogin={doLogin} />

  function logout() {
    // 只清会话, 保留工作区记忆
    setUser(null)
    setTasks([])
    setCurId(null)
    setView('chat')
    api.clearSession?.()
  }

  function summon(e) {
    // 从专家广场召唤专家 → 新任务并预选该专家
    const t = {
      id: 't' + Date.now(),
      seq: taskSeq++,
      title: '新任务 · ' + (e.name || '专家'),
      expert: e.expert_id || e.id || '',
      thread: '',
      workspace: cur?.workspace || '',
      msgs: [],
      pending: [],
      files: cur?.files || [],
      busy: false,
    }
    setTasks(ts => [t, ...ts])
    setCurId(t.id)
    setView('chat')
  }

  return (
    <div className="ws">
      {fatal && (
        <div className="fatal-bar" onClick={() => setFatal('')}>
          ⚠ {fatal.slice(0, 160)}（点我关闭）
        </div>
      )}
      <div className="ws-row">
      <Sidebar
        user={user}
        tasks={tasks}
        curId={curId}
        onNew={newTask}
        onOpen={openTask}
        onDelete={delTask}
        onRename={renameTask}
        onNav={setView}
        onLogout={logout}
        view={view === 'files' ? 'chat' : view}
      />
      {view === 'admin' ? (
        <AdminPane user={user} onBack={() => setView('chat')} />
      ) : view === 'experts' ? (
        <Experts user={user} onSummon={summon} />
      ) : view === 'scheduled' || view === 'library' ? (
        <StubPage id={view} />
      ) : (
        <>
          <ChatPane
            key={curId}
            user={user}
            task={cur}
            patchTask={patchTask}
            onOpenFiles={() => setView('files')}
          />
          <RightPane
            task={cur}
            patchTask={patchTask}
            view={view}
            setView={setView}
          />
        </>
      )}
      </div>
    </div>
  )
}

function StubPage({ id }) {
  const meta = {
    scheduled: { ic: '◷', label: '定时任务', hint: '让专家按计划自动干活, 敬请期待' },
    library: { ic: '▤', label: '资料库', hint: '团队共享知识库, 敬请期待' },
  }[id] || { ic: '◇', label: '', hint: '' }
  return (
    <div className="stub">
      <div className="stub-head">{meta.label}</div>
      <div className="stub-body">
        <span className="big">{meta.ic}</span>
        <p>{meta.hint}</p>
      </div>
    </div>
  )
}

function AdminPane({ user, onBack }) {
  const [tab, setTab] = useState('overview')
  const [data, setData] = useState(null)
  useEffect(() => {
    (async () => {
      try {
        if (tab === 'overview') setData(await api.adminOverview())
        else if (tab === 'metrics') setData(await api.adminMetrics())
        else if (tab === 'invites') setData(await api.adminInvites())
        else if (tab === 'audit') setData(await api.adminAudit())
      } catch (e) { setData({ error: String(e) }) }
    })()
  }, [tab])
  return (
    <div className="adminpage">
      <div className="admin-head">
        <button className="chat-head-btn" onClick={onBack} title="返回">‹</button>
        <span className="admin-title">管理后台</span>
        <div className="seg">
          {['overview', 'metrics', 'invites', 'audit'].map(t => (
            <button key={t} className={'seg-item' + (tab === t ? ' on' : '')} onClick={() => setTab(t)}>
              {{ overview: '总览', metrics: '用量', invites: '邀请码', audit: '审计' }[t]}
            </button>
          ))}
        </div>
      </div>
      <pre className="admin-body">{data ? JSON.stringify(data, null, 2) : '加载中…'}</pre>
    </div>
  )
}
