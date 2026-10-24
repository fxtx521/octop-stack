import React, { useMemo, useState } from 'react'

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

export default function Experts({ user, onSummon }) {
  const [q, setQ] = useState('')
  const experts = user?.experts || []
  const list = useMemo(() => {
    if (!q.trim()) return experts
    const k = q.trim().toLowerCase()
    return experts.filter(e =>
      (e.name || '').toLowerCase().includes(k) ||
      (e.description || '').toLowerCase().includes(k) ||
      tagsOf(e).some(t => (t || '').toLowerCase().includes(k))
    )
  }, [experts, q])

  return (
    <div className="market">
      <div className="market-head">
        <button className="market-tab on">◇ 专家</button>
        <input
          className="market-search"
          placeholder="搜索专家"
          value={q}
          onChange={e => setQ(e.target.value)}
        />
      </div>
      <div className="market-body">
        {list.length === 0 ? (
          <div className="tree-empty">{q ? '没有匹配的专家' : '暂无专家数据'}</div>
        ) : (
          <div className="market-grid">
            {list.map((e, i) => (
              <div key={e.expertID || e.id || i} className="exp-card" onClick={() => onSummon(e)}>
                <div className="exp-top">
                  <div className="exp-ava">{(e.name || '?').slice(0, 1)}</div>
                  <div className="exp-name">{e.name || '未命名专家'}</div>
                </div>
                <div className="exp-desc">{descOf(e)}</div>
                <div className="exp-tags">
                  {tagsOf(e).map((t, j) => <span key={j} className="exp-tag">{t}</span>)}
                </div>
                <div className="exp-foot">
                  <span className="exp-use">⚡ {e.usage || e.useCount || '—'}</span>
                  <button className="exp-call" onClick={ev => { ev.stopPropagation(); onSummon(e) }}>召唤</button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
