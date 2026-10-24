import React, { useEffect, useMemo, useState } from 'react'
import { api } from '../lib/octop.js'

function tagsOf(e) {
  // 专家数据里 skills/tags 形态不一, 兜底提取展示标签
  if (Array.isArray(e.tags) && e.tags.length) return e.tags.slice(0, 3)
  if (Array.isArray(e.skills) && e.skills.length) return e.skills.slice(0, 3)
  if (typeof e.description === 'string' && e.description.includes('、')) {
    return e.description.split('、').slice(0, 3)
  }
  return []
}

function descOf(e) {
  return e.description || e.intro || e.summary || '暂无简介'
}

// 新建/编辑专家表单
function ExpertForm({ initial, onDone, onCancel }) {
  const editing = !!initial?.agentID
  const [f, setF] = useState({
    name: initial?.name || '',
    description: initial?.description || '',
    system_prompt: initial?.system_prompt || '',
    welcome_message: initial?.welcome_message || '',
    is_shared: !!initial?.is_shared,
  })
  const [err, setErr] = useState('')
  const [busy, setBusy] = useState(false)
  const set = (k, v) => setF(s => ({ ...s, [k]: v }))
  const submit = async () => {
    if (!f.name.trim()) { setErr('专家名称必填'); return }
    setBusy(true); setErr('')
    try {
      const r = editing
        ? await api.agentUpdate(initial.agentID, {
            name: f.name, description: f.description,
            system_prompt: f.system_prompt, welcome_message: f.welcome_message,
          })
        : await api.agentCreate(f.name, f.description, f.system_prompt, f.welcome_message, f.is_shared)
      if (r?.ok) onDone()
      else setErr(r?.error || '保存失败')
    } catch (e) { setErr(String(e)) }
    setBusy(false)
  }
  return (
    <div className="exf-mask" onClick={onCancel}>
      <div className="exf-card" onClick={e => e.stopPropagation()}>
        <div className="exf-title">{editing ? '编辑专家' : '新建专家'}</div>
        <input className="exf-in" placeholder="专家名称 *（如：报价助手）"
          value={f.name} onChange={e => set('name', e.target.value)} />
        <input className="exf-in" placeholder="一句话简介（如：帮销售按客户情况出报价单）"
          value={f.description} onChange={e => set('description', e.target.value)} />
        <textarea className="exf-in exf-ta" placeholder="人设提示词：告诉这位专家他是谁、按什么规矩干活"
          value={f.system_prompt} onChange={e => set('system_prompt', e.target.value)} />
        <input className="exf-in" placeholder="开场欢迎语（可空）"
          value={f.welcome_message} onChange={e => set('welcome_message', e.target.value)} />
        {!editing && (
          <label className="exf-share">
            <input type="checkbox" checked={f.is_shared}
              onChange={e => set('is_shared', e.target.checked)} />
            共享给全部成员
          </label>
        )}
        {err && <div className="exf-err">{err}</div>}
        <div className="exf-btns">
          <button className="exf-btn" onClick={onCancel}>取消</button>
          <button className="exf-btn exf-ok" disabled={busy} onClick={submit}>
            {busy ? '保存中…' : '保存'}
          </button>
        </div>
      </div>
    </div>
  )
}

export default function Experts({ user, onSummon }) {
  const [q, setQ] = useState('')
  const [live, setLive] = useState(null) // 进页时从云端刷新的专家列表
  const [stale, setStale] = useState(false)
  const [manage, setManage] = useState(false) // 管理视图(含启停/编辑/删除)
  const [rows, setRows] = useState([]) // /api/agents 原始行(state/agentID)
  const [form, setForm] = useState(null) // null | {} 新建 | agent 编辑
  const [msg, setMsg] = useState('')
  const refresh = async () => {
    try {
      const fresh = await api.listExperts()
      if (fresh && fresh.length) { setLive(fresh); setStale(false) }
      else setStale(true)
      const m = await api.agentsManage()
      if (Array.isArray(m)) setRows(m)
    } catch { setStale(true) }
  }
  useEffect(() => { refresh() }, [])
  const experts = live || user?.experts || []
  // 管理视图: 合并 /api/agents 行(拿 agentID/state/归属)
  const merged = useMemo(() => {
    if (!manage) return experts
    const byName = {}
    rows.forEach(r => { byName[r.name] = r })
    return experts.map(e => ({ ...e, ...(byName[e.name] || {}) }))
  }, [experts, rows, manage])
  const list = useMemo(() => {
    if (!q.trim()) return merged
    const k = q.trim().toLowerCase()
    return merged.filter(e =>
      (e.name || '').toLowerCase().includes(k) ||
      (e.description || '').toLowerCase().includes(k) ||
      tagsOf(e).some(t => (t || '').toLowerCase().includes(k))
    )
  }, [merged, q])
  const doToggle = async (e, ev) => {
    ev.stopPropagation()
    const id = e.agentID; if (!id) { setMsg('该专家无实例ID(可能是云端预设), 无法启停'); return }
    const r = await api.agentToggle(id, e.state === 'running' ? 'stop' : 'start')
    setMsg(r?.ok ? '' : (r?.error || '操作失败'))
    refresh()
  }
  const doDelete = async (e, ev) => {
    ev.stopPropagation()
    const id = e.agentID
    if (!id) { setMsg('该专家无实例ID, 无法删除'); return }
    if (!window.confirm(`删除专家「${e.name}」? 其会话历史保留在云端`)) return
    const r = await api.agentDelete(id)
    setMsg(r?.ok ? '' : (r?.error || '删除失败'))
    refresh()
  }
  const doEdit = (e, ev) => {
    ev.stopPropagation()
    setForm(e)
  }

  return (
    <div className="market">
      <div className="market-head">
        <button className="market-tab on">◇ 专家</button>
        <button className={'exp-mgmt' + (manage ? ' on' : '')}
          onClick={() => setManage(m => !m)}>管理</button>
        <input
          className="market-search"
          placeholder="搜索专家"
          value={q}
          onChange={e => setQ(e.target.value)}
        />
      </div>
      <div className="market-body">
        {msg && <div className="tree-empty">{msg}</div>}
        {stale && <div className="tree-empty">云端专家列表拉取失败, 显示登录时缓存</div>}
        {manage && (
          <div className="exp-mgmt-bar">
            <button className="exp-new" onClick={() => setForm({})}>＋ 新建专家</button>
            <span className="exp-mgmt-tip">启停/编辑/删除直连云端 /api/agents</span>
          </div>
        )}
        {list.length === 0 ? (
          <div className="tree-empty">{q ? '没有匹配的专家' : '暂无专家数据'}</div>
        ) : (
          <div className="market-grid">
            {list.map((e, i) => (
              <div key={e.expertID || e.id || e.agentID || i} className="exp-card" onClick={() => onSummon(e)}>
                <div className="exp-top">
                  <div className="exp-ava">{(e.name || '?').slice(0, 1)}</div>
                  <div className="exp-name">{e.name || '未命名专家'}</div>
                  {manage && e.state && (
                    <span className={'exp-state ' + (e.state === 'running' ? 'run' : 'stop')}>
                      {e.state === 'running' ? '运行中' : '已停止'}
                    </span>
                  )}
                </div>
                <div className="exp-desc">{descOf(e)}</div>
                <div className="exp-tags">
                  {tagsOf(e).map((t, j) => <span key={j} className="exp-tag">{t}</span>)}
                </div>
                <div className="exp-foot">
                  <span className="exp-use">⚡ {e.usage || e.useCount || '—'}</span>
                  {manage ? (
                    <span className="exp-ops">
                      <button className="exp-mini" onClick={ev => doToggle(e, ev)}>
                        {e.state === 'running' ? '停止' : '启动'}
                      </button>
                      <button className="exp-mini" onClick={ev => doEdit(e, ev)}>编辑</button>
                      <button className="exp-mini exp-danger" onClick={ev => doDelete(e, ev)}>删除</button>
                    </span>
                  ) : (
                    <button className="exp-call" onClick={ev => { ev.stopPropagation(); onSummon(e) }}>召唤</button>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
      {form && (
        <ExpertForm
          initial={form.agentID ? form : null}
          onDone={() => { setForm(null); refresh() }}
          onCancel={() => setForm(null)}
        />
      )}
    </div>
  )
}
