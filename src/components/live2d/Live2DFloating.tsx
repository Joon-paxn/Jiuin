import { useCallback, useState } from 'react'
import { live2dConfig } from '../../config/live2d'
import { classNames } from '../../utils/classNames'
import { Live2DCanvas } from './Live2DCanvas'
import type { Live2DModelConfig, Live2DStatus } from './live2d.types'

type Live2DFloatingProps = {
  config?: Live2DModelConfig
}

export function Live2DFloating({ config = live2dConfig }: Live2DFloatingProps) {
  const [status, setStatus] = useState<Live2DStatus>('idle')
  const updateStatus = useCallback((nextStatus: Live2DStatus) => setStatus(nextStatus), [])

  if (!config.enabled) {
    return null
  }

  return (
    <aside className={classNames('live2d-floating', `is-${status}`)} aria-label={`${config.displayName} Live2D 角色`}>
      <div className="live2d-floating__frame">
        <Live2DCanvas config={config} onStatusChange={updateStatus} />
        {status === 'loading' && <span className="live2d-floating__status">正在唤醒角色…</span>}
        {status === 'error' && <span className="live2d-floating__status">角色暂时无法显示</span>}
      </div>
      <p className="live2d-floating__hint">轻触角色以切换表情</p>
    </aside>
  )
}
