import React from 'react'

const NAV = [
  { id: 'chat', ic: '✚', label: '新建任务' },
  { id: 'experts', ic: '◇', label: '专家广场' },
  { id: 'scheduled', ic: '◷', label: '定时任务' },
  { id: 'library', ic: '▤', label: '资料库' },
  { id: 'admin', ic: '⚙', label: '管理后台' },
]

const STATUS_COLOR = { doing: 'var(--brand)', done: '#bfc2c7', error: 'var(--danger)' }

export default function Sidebar({ user, tasks, curId, view, onNav, onNew, onOpen, onDelete, onLogout }) {
  const u = user?.usage || ''
  return (
    <aside className="sb">
      <div className="sb-top">
        <span className="sb-logo">Octop<i /></span>
        <span className="sb-ver">V1.0</span>
        <span className="spacer" />
      </div>

      <nav className="sb-nav">
        {NAV.map(n => (
          <button
            key={n.id}
            className={'sb-item' + (view === n.id ? ' on' : '')}
            onClick={() => (n.id === 'chat' ? onNew() : onNav(n.id))}
          >
            <span className="ic">{n.ic}</span>
            {n.label}
            {n.id === 'experts' && user?.experts?.length > 0 && (
              <span className="sb-badge">{user.experts.length}</span>
            )}
          </button>
        ))}
      </nav>

      <div className="sb-group">任务 ({tasks.length})</div>
      <div className="sb-tasks">
        {tasks.length === 0 && <div className="sb-empty">还没有任务, 点上方「新建任务」</div>}
        {tasks.map(t => (
          <div
            key={t.id}
            className={'sb-task' + (t.id === curId && (view === 'chat' || view === 'files') ? ' on' : '')}
            onClick={() => onOpen(t.id)}
            title={t.title}
          >
            <span
              className="sb-task-dot"
              style={{ background: t.busy ? STATUS_COLOR.doing : t.pending?.length ? 'var(--warn)' : STATUS_COLOR.done }}
            />
            <span className="sb-task-title">{t.title}</span>
            <button
              className="sb-task-del"
              onClick={e => { e.stopPropagation(); onDelete(t.id) }}
              title="删除"
            >×</button>
          </div>
        ))}
      </div>

      <div className="sb-user">
        <div className="sb-avatar">{(user?.name || '?').slice(0, 1).toUpperCase()}</div>
        <div className="sb-user-main">
          <div className="sb-user-name">{user?.name || '-'}</div>
          <div className="sb-user-role">{user?.role === 'admin' ? '管理员' : '成员'}</div>
          {u && <div className="sb-usage" title="用量">{u}</div>}
        </div>
        <button className="sb-logout" onClick={onLogout} title="退出登录">⏻</button>
      </div>
    </aside>
  )
}
