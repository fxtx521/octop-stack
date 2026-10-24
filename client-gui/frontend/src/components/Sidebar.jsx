import React from 'react'

const STATUS_COLOR = { doing: 'var(--warn)', done: 'var(--accent)', error: 'var(--danger)' }

export default function Sidebar({ user, tasks, curId, onNew, onOpen, onDelete, onAdmin, view }) {
  return (
    <aside className="sb">
      <div className="sb-brand">Octop<span className="dot">.</span></div>
      <button className="sb-new" onClick={onNew}>＋ 新任务</button>
      <div className="sb-sec">任务</div>
      <div className="sb-tasks">
        {tasks.length === 0 && <div className="sb-empty">还没有任务</div>}
        {tasks.map(t => (
          <div
            key={t.id}
            className={'sb-task' + (t.id === curId && view !== 'admin' ? ' on' : '')}
            onClick={() => onOpen(t.id)}
            title={t.title}
          >
            <span className="sb-task-dot" style={{ background: t.busy ? STATUS_COLOR.doing : 'var(--faint)' }} />
            <span className="sb-task-name">{t.title}</span>
            <button
              className="sb-task-x"
              title="删除"
              onClick={e => { e.stopPropagation(); onDelete(t.id) }}
            >×</button>
          </div>
        ))}
      </div>
      <div className="sb-bottom">
        <button className="sb-admin" onClick={onAdmin}>🛡 管理后台</button>
        <div className="sb-user">
          <span className="sb-avatar">{(user.role || 'U')[0].toUpperCase()}</span>
          <span className="sb-role">{user.role === 'admin' ? '管理员' : '员工'}</span>
        </div>
        {user.usage && <div className="sb-usage" title="用量">用量 {user.usage}</div>}
      </div>
    </aside>
  )
}
