import React, { useState } from 'react'

export default function Login({ onLogin }) {
  const [srv, setSrv] = useState('http://127.0.0.1:8099')
  const [u, setU] = useState('')
  const [p, setP] = useState('')
  const [err, setErr] = useState('')
  const [busy, setBusy] = useState(false)

  async function submit(e) {
    e?.preventDefault()
    setErr('')
    setBusy(true)
    try {
      const r = await onLogin(srv.replace(/\/+$/, ''), u.trim(), p)
      if (!r.ok) setErr(r.msg || '登录失败')
    } catch (e2) {
      setErr('连接失败: ' + e2)
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="login-bg">
      <form className="login-card" onSubmit={submit}>
        <div className="login-brand">Octop<span className="dot">.</span></div>
        <div className="login-sub">企业 AI 工作台 · 员工端</div>
        <label className="field-label">服务器</label>
        <input className="field" value={srv} onChange={e => setSrv(e.target.value)} placeholder="http://公司服务器:端口" />
        <label className="field-label">账号</label>
        <input className="field" value={u} onChange={e => setU(e.target.value)} autoComplete="off" placeholder="工号或用户名" autoFocus />
        <label className="field-label">密码</label>
        <input className="field" type="password" value={p} onChange={e => setP(e.target.value)} placeholder="密码" />
        <button className="login-btn" disabled={busy}>{busy ? '登录中…' : '进入工作台'}</button>
        <div className="login-err">{err}</div>
      </form>
    </div>
  )
}
