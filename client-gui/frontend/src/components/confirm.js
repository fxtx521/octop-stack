import { useState } from 'react'

// Wails WebView 的 window.confirm 静默返回false, 不能用。
// 两段式确认: 第一次点亮成「确认?」态, 3秒内再点才执行。
export function useConfirm() {
  const [pending, setPending] = useState(null)
  const guard = (key, fn) => () => {
    if (pending === key) {
      setPending(null)
      fn()
    } else {
      setPending(key)
      setTimeout(() => setPending(p => (p === key ? null : p)), 3000)
    }
  }
  return [pending, guard]
}
