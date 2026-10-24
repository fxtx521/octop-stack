import React, { useEffect, useState } from 'react'
import { api } from '../lib/octop.js'

// key掩码: 只露前6后4, 防明文泄露
function maskKey(k) {
  if (!k) return '未设'
  if (k.length <= 12) return k.slice(0, 3) + '****'
  return k.slice(0, 6) + '****' + k.slice(-4)
}

// 用户管理: 全量列表 + 启停/解锁/改角色
export function AdminUsers() {
  const [rows, setRows] = useState(null)
  const [msg, setMsg] = useState('')
  const load = async () => setRows(await api.usersList() || [])
  useEffect(() => { load() }, [])
  const act = async (fn, okTip) => {
    const r = await fn()
    setMsg(r?.ok ? (okTip || '已生效') : (r?.error || '操作失败'))
    load()
  }
  const toggle = (u) => act(() => api.userPatch(u.id, { disabled: !u.disabled }),
    u.disabled ? '已启用' : '已禁用')
  const role = (u, r) => act(() => api.userPatch(u.id, { role: r }), '角色已改为' + r)
  const unlock = (u) => act(() => api.userUnlock(u.id), '已解锁')
  return (
    <div className="adm-wrap">
      {msg && <div className="tree-empty">{msg}</div>}
      {rows === null && <div className="tree-empty">加载中…</div>}
      {rows && rows.length === 0 && <div className="tree-empty">无用户数据</div>}
      {rows && rows.map(u => (
        <div key={u.id} className="adm-row">
          <div className="adm-main">
            <div className="skl-name">
              {u.display_name || u.username}
              <span className={'skl-kind' + (u.role === 'admin' ? ' b' : '')}>
                {u.role_name || u.role}
              </span>
              {u.disabled && <span className="skl-kind">已禁用</span>}
              {u.login_locked && <span className="skl-kind">登录锁定</span>}
              {u.token_quota && <span className="skl-kind">配额 {u.token_quota}</span>}
            </div>
            <div className="skl-desc">@{u.username} · ID {u.id}</div>
          </div>
          <div className="exp-ops">
            {u.login_locked && (
              <button className="exp-mini" onClick={() => unlock(u)}>解锁</button>
            )}
            {u.role !== 'admin' && (
              <button className="exp-mini" onClick={() => role(u, 'admin')}>设为管理员</button>
            )}
            {u.role === 'admin' && (
              <button className="exp-mini" onClick={() => role(u, 'staff')}>设为成员</button>
            )}
            <button className="exp-mini" onClick={() => toggle(u)}>
              {u.disabled ? '启用' : '禁用'}
            </button>
          </div>
        </div>
      ))}
    </div>
  )
}

// 模型供应商: 列表+新增+启停+删除(key服务端脱敏回显)
export function AdminProviders() {
  const [rows, setRows] = useState(null)
  const [msg, setMsg] = useState('')
  const [creating, setCreating] = useState(false)
  const [f, setF] = useState({ name: '', base_url: '', api_key: '', models: '' })
  const load = async () => setRows(await api.providersList() || [])
  useEffect(() => { load() }, [])
  const create = async () => {
    if (!f.name.trim() || !f.base_url.trim()) { setMsg('名称和 Base URL 必填'); return }
    const models = f.models.trim()
      ? f.models.split(/[,\n]/).map(s => s.trim()).filter(Boolean)
          .map(id => ({ id, name: id }))
      : []
    const r = await api.providerCreate(f.name.trim(), 'openai-compatible',
      f.base_url.trim(), f.api_key.trim(),
      models.length ? JSON.stringify(models) : '')
    if (r?.ok) { setCreating(false); setF({ name: '', base_url: '', api_key: '', models: '' }); setMsg('已添加'); load() }
    else setMsg(r?.error || '添加失败')
  }
  const toggle = (p) => api.providerPatch(p.id, JSON.stringify({ enabled: !p.enabled }))
    .then(r => { setMsg(r?.ok ? '' : (r?.error || '操作失败')); load() })
  const del = async (p) => {
    if (!window.confirm(`删除供应商「${p.name}」? 使用它的专家会被影响`)) return
    const r = await api.providerDelete(p.id)
    setMsg(r?.ok ? '已删除' : (r?.error || '删除失败'))
    load()
  }
  return (
    <div className="adm-wrap">
      {msg && <div className="tree-empty">{msg}</div>}
      {!creating && (
        <button className="exp-new" onClick={() => setCreating(true)}>＋ 接入供应商</button>
      )}
      {creating && (
        <div className="skl-form">
          <input className="exf-in" placeholder="名称（如 qwen / glm）"
            value={f.name} onChange={e => setF(s => ({ ...s, name: e.target.value }))} />
          <input className="exf-in" placeholder="Base URL（如 https://open.bigmodel.cn/api/paas/v4）"
            value={f.base_url} onChange={e => setF(s => ({ ...s, base_url: e.target.value }))} />
          <input className="exf-in" type="password" placeholder="API Key"
            value={f.api_key} onChange={e => setF(s => ({ ...s, api_key: e.target.value }))} />
          <input className="exf-in" placeholder="模型列表（逗号分隔, 如 glm-4.7,glm-4.7-flash）"
            value={f.models} onChange={e => setF(s => ({ ...s, models: e.target.value }))} />
          <div className="exf-btns">
            <button className="exf-btn" onClick={() => setCreating(false)}>取消</button>
            <button className="exf-btn exf-ok" onClick={create}>保存</button>
          </div>
        </div>
      )}
      {rows === null && <div className="tree-empty">加载中…</div>}
      {rows && rows.map(p => (
        <div key={p.id} className="adm-row">
          <div className="adm-main">
            <div className="skl-name">
              {p.name}
              <span className={'skl-kind' + (p.enabled ? ' b' : '')}>
                {p.enabled ? '启用' : '停用'}
              </span>
              <span className="skl-kind">{p.kind}</span>
            </div>
            <div className="skl-desc">{p.base_url} · key {maskKey(p.api_key)}
              {p.models?.length ? ` · ${p.models.map(m => m.id).join(', ')}` : ''}
            </div>
          </div>
          <div className="exp-ops">
            <button className="exp-mini" onClick={() => toggle(p)}>{p.enabled ? '停用' : '启用'}</button>
            <button className="exp-mini exp-danger" onClick={() => del(p)}>删除</button>
          </div>
        </div>
      ))}
    </div>
  )
}
