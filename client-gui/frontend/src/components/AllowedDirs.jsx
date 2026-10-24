import React, { useEffect, useState } from 'react'
import { api } from '../lib/octop.js'

// A档: 授权目录清单(全盘访问白名单)
export default function AllowedDirs({ onClose }) {
  const [dirs, setDirs] = useState([])
  const [input, setInput] = useState('')
  const [err, setErr] = useState('')
  const load = async () => setDirs(await api.allowedList() || [])
  useEffect(() => { load() }, [])
  const add = async () => {
    const r = await api.allowedAdd(input)
    if (r?.ok) { setInput(''); setErr(''); load() }
    else setErr(r?.error || '添加失败')
  }
  const del = async (d) => {
    if (!window.confirm(`取消授权「${d}」? 之后专家将无法访问该目录`)) return
    await api.allowedRemove(d)
    load()
  }
  return (
    <div className="exf-mask" onClick={onClose}>
      <div className="exf-card" onClick={e => e.stopPropagation()}>
        <div className="exf-title">授权目录</div>
        <div className="ald-tip">清单内目录专家可用绝对路径读写
（每个操作仍需你批准）。清单外一律拒绝。</div>
        <div className="ald-add">
          <input className="exf-in" style={{ marginBottom: 0 }}
            placeholder="粘贴绝对路径, 如 /Users/xx/项目资料"
            value={input}
            onChange={e => setInput(e.target.value)}
            onKeyDown={e => e.key === 'Enter' && add()} />
          <button className="exp-new" onClick={add}>添加</button>
        </div>
        {err && <div className="exf-err">{err}</div>}
        <div className="skl-list">
          {dirs.length === 0 && <div className="tree-empty">还没有授权目录, 专家只能访问工作区</div>}
          {dirs.map(d => (
            <div key={d} className="skl-row">
              <span className="skl-emoji">📂</span>
              <div className="skl-main"><div className="skl-name">{d}</div></div>
              <button className="exp-mini exp-danger" onClick={() => del(d)}>取消授权</button>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
