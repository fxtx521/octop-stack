import React, { useEffect, useMemo, useState } from 'react'
import Login from './components/Login.jsx'
import Sidebar from './components/Sidebar.jsx'
import ChatPane from './components/ChatPane.jsx'
import RightPane from './components/RightPane.jsx'
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
  const [view, setView] = useState('chat')        // chat | files | admin
  const cur = useMemo(() => tasks.find(t => t.id === curId), [tasks, curId])

  function patchTask(id, patch) {
    setTasks(ts => ts.map(t => (t.id === id ? { ...t, ...patch } : t)))
  }

  function newTask() {
    const t = {
      id: 't' + Date.now(),
      seq: taskSeq++,
      title: '新任务',
      expert: user?.experts?.[0]?.id || '',
      workspace: '',
      msgs: [],      // {role:'user|ai|sys', text, ops?, ts}
      pending: [],   // 审批队列
      files: null,   // 文件树缓存
      busy: false,
    }
    setTasks(ts => [t, ...ts])
    setCurId(t.id)
    setView('chat')
  }

  async function doLogin(srv, u, p) {
    try {
      const r = await api.login(srv, u, p)
      if (r.ok) {
        // 恢复上次工作区 + 文件树
        let ws = null
        try {
          const w = await api.startupWorkspace()
          if (w.ok) ws = w.root
        } catch { /* ignore */ }
        setUser({ name: u, role: r.role || 'member', experts: r.experts || [], usage: r.usage || null })
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

  function delTask(id) {
    setTasks(ts => {
      const left = ts.filter(t => t.id !== id)
      if (id === curId) setCurId(left[0]?.id || null)
      return left
    })
  }

  if (!user) return <Login onLogin={doLogin} />

  return (
    <div className="ws">
      {fatal && (
        <div className="fatal-bar" onClick={() => setFatal('')}>
          ⚠ {fatal.slice(0, 160)}（点我关闭）
        </div>
      )}
      <div className="ws-row">
      {view === 'admin' ? (
        <AdminPane user={user} />
      ) : (
        <>
          <Sidebar
            user={user}
            tasks={tasks}
            curId={curId}
            onNew={newTask}
            onOpen={openTask}
            onDelete={delTask}
            onAdmin={() => setView('admin')}
            view={view}
          />
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

function AdminPane({ user }) {
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
