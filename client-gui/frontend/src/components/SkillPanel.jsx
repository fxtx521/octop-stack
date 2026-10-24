import React, { useEffect, useState } from 'react'
import { api } from '../lib/octop.js'
import { useConfirm } from './confirm.js'

// 技能面板: 列出专家全部技能, 支持新建/删除(workspace技能)
export default function SkillPanel({ agent, onClose }) {
  const agentID = agent?.agent_id || agent?.agentID || 'main'
  const [rows, setRows] = useState(null)
  const [err, setErr] = useState('')
  const [creating, setCreating] = useState(false)
  const [f, setF] = useState({ slug: '', description: '', body: '' })
  const [busy, setBusy] = useState(false)
  const [pending, guard] = useConfirm()
  const load = async () => {
    const r = await api.skillList(agentID)
    setRows(Array.isArray(r) ? r : [])
  }
  useEffect(() => { load() }, [agentID])
  const create = async () => {
    if (!f.slug.trim()) { setErr('技能标识必填(小写字母-数字, 如 price-quote)'); return }
    if (!f.body.trim()) { setErr('技能正文必填(告诉专家这个技能怎么用)'); return }
    setBusy(true); setErr('')
    const r = await api.skillCreate(agentID, f.slug.trim(), f.description.trim(), f.body)
    if (r?.ok) { setCreating(false); setF({ slug: '', description: '', body: '' }); load() }
    else setErr(r?.error || '保存失败')
    setBusy(false)
  }
  const del = (slug) => guard('del-' + slug, async () => {
    const r = await api.skillDelete(agentID, slug)
    setErr(r?.ok ? '' : (r?.error || '删除失败'))
    load()
  })()
  return (
    <div className="exf-mask" onClick={onClose}>
      <div className="exf-card exf-wide" onClick={e => e.stopPropagation()}>
        <div className="exf-title">技能库 · {agent?.name || agentID}</div>
        {err && <div className="exf-err">{err}</div>}
        {!creating && (
          <button className="exp-new" onClick={() => setCreating(true)}>＋ 新建技能</button>
        )}
        {creating && (
          <div className="skl-form">
            <input className="exf-in" placeholder="技能标识 *（小写英文-数字, 如 price-quote）"
              value={f.slug} onChange={e => setF(s => ({ ...s, slug: e.target.value }))} />
            <input className="exf-in" placeholder="一句话说明（这个技能干什么）"
              value={f.description} onChange={e => setF(s => ({ ...s, description: e.target.value }))} />
            <textarea className="exf-in exf-ta skl-tall" placeholder="技能正文：告诉专家什么时候用它、按什么步骤干"
              value={f.body} onChange={e => setF(s => ({ ...s, body: e.target.value }))} />
            <div className="exf-btns">
              <button className="exf-btn" onClick={() => setCreating(false)}>取消</button>
              <button className="exf-btn exf-ok" disabled={busy} onClick={create}>
                {busy ? '保存中…' : '保存'}
              </button>
            </div>
          </div>
        )}
        <div className="skl-list">
          {rows === null && <div className="tree-empty">技能列表加载中…</div>}
          {rows && rows.length === 0 && <div className="tree-empty">该专家暂无技能</div>}
          {rows && rows.map(s => (
            <div key={s.slug} className="skl-row">
              <span className="skl-emoji">{s.emoji || '🧩'}</span>
              <div className="skl-main">
                <div className="skl-name">{s.name || s.slug}
                  <span className={'skl-kind' + (s.kind === 'builtin' ? ' b' : '')}>
                    {s.kind === 'builtin' ? '内置' : '自定义'}
                  </span>
                </div>
                <div className="skl-desc">{s.description || s.summary?.zh || ''}</div>
              </div>
              {s.kind !== 'builtin' && (
                <button className="exp-mini exp-danger" onClick={del(s.slug)}>
                  {pending === 'del-' + s.slug ? '确认删除?' : '删除'}
                </button>
              )}
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
