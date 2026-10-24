import React, { useEffect, useState } from 'react'
import { api } from '../lib/octop.js'
import { useConfirm } from './confirm.js'

// A档: 授权目录清单(全盘访问白名单)
export default function AllowedDirs({ onClose }) {
  const [dirs, setDirs] = useState([])
  const [input, setInput] = useState('')
  const [err, setErr] = useState('')
  const [shellOn, setShellOn] = useState(false)
  const [pending, guard] = useConfirm()
  const load = async () => {
    setDirs(await api.allowedList() || [])
    setShellOn(await api.shellState() || false)
  }
  useEffect(() => { load() }, [])
  const toggleShell = () => guard('shell', async () => {
    const on = !shellOn
    await api.shellToggle(on)
    setShellOn(on)
  })()
  const add = async () => {
    const r = await api.allowedAdd(input)
    if (r?.ok) { setInput(''); setErr(''); load() }
    else setErr(r?.error || '添加失败')
  }
  const del = (d) => guard('del-' + d, async () => {
    await api.allowedRemove(d)
    load()
  })()
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
        <div className="ald-shell">
          <div className="skl-main">
            <div className="skl-name">允许执行系统命令
              <span className={'skl-kind' + (shellOn ? ' b' : '')}>{shellOn ? '已开启' : '默认关'}</span>
            </div>
            <div className="skl-desc">每条命令弹卡审批, 高危命令强制拒绝, 全部留痕</div>
          </div>
          <button className={'exp-mini' + (shellOn ? ' exp-danger' : '')} onClick={toggleShell}>
            {shellOn ? (pending === 'shell' ? '确认关闭?' : '关闭') : (pending === 'shell' ? '确认开启?' : '开启')}
          </button>
        </div>
        <div className="skl-list">
          {dirs.length === 0 && <div className="tree-empty">还没有授权目录, 专家只能访问工作区</div>}
          {dirs.map(d => (
            <div key={d} className="skl-row">
              <span className="skl-emoji">📂</span>
              <div className="skl-main"><div className="skl-name">{d}</div></div>
              <button className="exp-mini exp-danger" onClick={del(d)}>
                {pending === 'del-' + d ? '确认取消?' : '取消授权'}
              </button>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
