import React, { useEffect, useState } from 'react'
import { api } from '../lib/octop.js'
import FileTree from './FileTree.jsx'

export default function RightPane({ task, patchTask, view, setView }) {
  const [tab, setTab] = useState('files')
  const [preview, setPreview] = useState(null) // {path, text}

  useEffect(() => { setTab(view === 'files' ? 'files' : tab) }, [view])

  async function openFile(path) {
    setPreview({ path, text: '读取中…' })
    try {
      const r = await api.readFile(path)
      setPreview({ path, text: r.ok ? (r.content || '(空文件)') : (r.msg || '读取失败') })
    } catch (e) {
      setPreview({ path, text: '读取错误: ' + e })
    }
  }

  async function refresh() {
    try {
      const files = await api.listFiles()
      patchTask(task.id, { files: files || [] })
    } catch { /* ignore */ }
  }

  return (
    <aside className="rightpane">
      <div className="rp-tabs">
        <button className={'rp-tab' + (tab === 'files' ? ' on' : '')} onClick={() => setTab('files')}>文件</button>
        <button className={'rp-tab' + (tab === 'log' ? ' on' : '')} onClick={() => setTab('log')}>动态</button>
      </div>

      {tab === 'files' && (
        <div className="rp-files">
          <div className="rp-files-head">
            <span className="rp-label">{task?.workspace ? '工作区' : '未选择工作区'}</span>
            <button className="icon-btn" title="刷新" onClick={refresh}>⟳</button>
          </div>
          <div className="rp-tree">
            {task?.workspace
              ? <FileTree task={task} files={task.files} onOpen={openFile} />
              : <div className="rp-empty">先在对话区选择工作文件夹</div>}
          </div>
          {preview && (
            <div className="rp-preview">
              <div className="rp-preview-head">
                <span className="rp-preview-path">{preview.path}</span>
                <button className="icon-btn" onClick={() => setPreview(null)}>×</button>
              </div>
              <pre className="rp-preview-body">{preview.text}</pre>
            </div>
          )}
        </div>
      )}

      {tab === 'log' && (
        <div className="rp-log">
          {(task?.msgs || []).slice().reverse().map((m, i) => (
            <div key={i} className={'log-item log-' + m.role}>
              {m.text.length > 120 ? m.text.slice(0, 120) + '…' : m.text}
            </div>
          ))}
          {!task?.msgs?.length && <div className="rp-empty">暂无动态</div>}
        </div>
      )}
    </aside>
  )
}
