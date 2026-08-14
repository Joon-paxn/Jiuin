import type { Live2DModel } from 'pixi-live2d-display'
import {
  resolveInteractionPart,
  selectInteractionDialogue,
  type Live2DInteractionEvent,
  type Live2DInteractionPart,
} from './interactionConfig'

type RendererViewport = {
  screen: {
    width: number
    height: number
  }
}

type Live2DInteractionControllerOptions = {
  model: Live2DModel
  renderer: RendererViewport
  view: HTMLCanvasElement
  hitAreas: readonly string[]
  onInteraction: (interaction: Live2DInteractionEvent) => void
}

export type Live2DInteractionController = {
  dispose: () => void
}

const trackingDamping = 0.16
const trackingSettledDelta = 0.002
const interactionCooldownMs = 550

function clamp(value: number) {
  return Math.min(1, Math.max(-1, value))
}

function normalizedViewportPoint(event: PointerEvent) {
  if (!window.innerWidth || !window.innerHeight) {
    return { x: 0, y: 0 }
  }

  return {
    x: clamp((event.clientX / window.innerWidth) * 2 - 1),
    y: clamp((event.clientY / window.innerHeight) * 2 - 1),
  }
}

function canvasPoint(event: PointerEvent, view: HTMLCanvasElement, renderer: RendererViewport) {
  const bounds = view.getBoundingClientRect()
  if (!bounds.width || !bounds.height) {
    return undefined
  }

  return {
    x: (event.clientX - bounds.left) * (renderer.screen.width / bounds.width),
    y: (event.clientY - bounds.top) * (renderer.screen.height / bounds.height),
  }
}

/**
 * Keeps high-frequency pointer work outside React. It is recreated whenever a
 * model is replaced, so each model only receives its own hit-area metadata.
 */
export function createLive2DInteractionController({
  model,
  renderer,
  view,
  hitAreas,
  onInteraction,
}: Live2DInteractionControllerOptions): Live2DInteractionController {
  const enabledHitAreas = new Set(hitAreas)
  const target = { x: 0, y: 0 }
  const current = { x: 0, y: 0 }
  const previousDialogue = new Map<Live2DInteractionPart, string>()
  let frameHandle = 0
  let lastInteractionAt = 0
  let disposed = false

  const updateFocus = () => {
    frameHandle = 0
    if (disposed) {
      return
    }

    current.x += (target.x - current.x) * trackingDamping
    current.y += (target.y - current.y) * trackingDamping

    // Live2DModel.focus accepts Pixi world coordinates. The normalized
    // viewport vector is mapped to this renderer's world without page scroll.
    model.focus(
      ((current.x + 1) * 0.5) * renderer.screen.width,
      ((current.y + 1) * 0.5) * renderer.screen.height,
    )

    if (
      Math.abs(target.x - current.x) > trackingSettledDelta
      || Math.abs(target.y - current.y) > trackingSettledDelta
    ) {
      frameHandle = window.requestAnimationFrame(updateFocus)
    }
  }

  const scheduleFocus = () => {
    if (!frameHandle && !disposed) {
      frameHandle = window.requestAnimationFrame(updateFocus)
    }
  }

  const trackPointer = (event: PointerEvent) => {
    // Touch tracking is intentionally omitted; touch still supports real
    // hit-area clicks, while mouse users receive full-viewport gaze tracking.
    if (event.pointerType && event.pointerType !== 'mouse') {
      return
    }
    Object.assign(target, normalizedViewportPoint(event))
    scheduleFocus()
  }

  const interact = (event: PointerEvent) => {
    const now = performance.now()
    if (disposed || now - lastInteractionAt < interactionCooldownMs || enabledHitAreas.size === 0) {
      return
    }

    const point = canvasPoint(event, view, renderer)
    if (!point) {
      return
    }

    const hitAreaName = model.hitTest(point.x, point.y).find((name) => enabledHitAreas.has(name))
    if (!hitAreaName) {
      return
    }

    lastInteractionAt = now
    const part = resolveInteractionPart(hitAreaName)
    const message = selectInteractionDialogue(part, previousDialogue.get(part))
    previousDialogue.set(part, message)
    onInteraction({ hitAreaName, part, message })
  }

  document.addEventListener('pointermove', trackPointer, { passive: true })
  view.addEventListener('pointerup', interact)

  return {
    dispose: () => {
      disposed = true
      document.removeEventListener('pointermove', trackPointer)
      view.removeEventListener('pointerup', interact)
      if (frameHandle) {
        window.cancelAnimationFrame(frameHandle)
      }
    },
  }
}
