export type OnlinePresenceState = {
  count: number
  status: 'connecting' | 'online' | 'offline' | 'reconnecting'
}

type OnlineMessage = {
  type: 'online_count'
  count: number
}

type PresenceListener = (state: OnlinePresenceState) => void

const initialState: OnlinePresenceState = { count: 0, status: 'connecting' }
const maximumReconnectDelay = 16_000

class OnlinePresenceManager {
  private state = initialState
  private socket: WebSocket | undefined
  private reconnectTimer: number | undefined
  private reconnectDelay = 1_000
  private paused = false
  private readonly listeners = new Set<PresenceListener>()

  subscribe(listener: PresenceListener) {
    this.listeners.add(listener)
    listener(this.state)
    this.connect()
    return () => {
      this.listeners.delete(listener)
    }
  }

  bindPageLifecycle() {
    window.addEventListener('pagehide', () => {
      this.paused = true
      window.clearTimeout(this.reconnectTimer)
      this.reconnectTimer = undefined
      this.socket?.close(1000, 'page hidden')
      this.socket = undefined
    })
    window.addEventListener('pageshow', () => {
      this.paused = false
      this.connect()
    })
  }

  private connect() {
    if (this.paused || this.socket || this.listeners.size === 0) {
      return
    }

    this.setState({ ...this.state, status: this.reconnectDelay > 1_000 ? 'reconnecting' : 'connecting' })
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const basePath = import.meta.env.BASE_URL.replace(/\/$/, '')
    const socket = new WebSocket(`${protocol}//${window.location.host}${basePath}/ws/online`)
    this.socket = socket

    socket.addEventListener('open', () => {
      if (this.socket !== socket) {
        return
      }
      this.reconnectDelay = 1_000
      this.setState({ ...this.state, status: 'online' })
    })

    socket.addEventListener('message', (event) => {
      if (this.socket !== socket || typeof event.data !== 'string') {
        return
      }
      try {
        const message = JSON.parse(event.data) as Partial<OnlineMessage>
        const count = message.count
        if (message.type === 'online_count' && typeof count === 'number' && Number.isInteger(count) && count >= 0) {
          this.setState({ count, status: 'online' })
        }
      } catch {
        // Ignore malformed messages; the connection remains usable.
      }
    })

    socket.addEventListener('close', () => {
      if (this.socket !== socket) {
        return
      }
      this.socket = undefined
      if (this.paused) {
        return
      }
      this.setState({ ...this.state, status: 'offline' })
      this.scheduleReconnect()
    })

    socket.addEventListener('error', () => {
      // `close` owns state transitions and reconnect scheduling.
    })
  }

  private scheduleReconnect() {
    if (this.reconnectTimer !== undefined || this.listeners.size === 0) {
      return
    }
    const delay = this.reconnectDelay
    this.reconnectDelay = Math.min(maximumReconnectDelay, this.reconnectDelay * 2)
    this.reconnectTimer = window.setTimeout(() => {
      this.reconnectTimer = undefined
      this.connect()
    }, delay)
  }

  private setState(nextState: OnlinePresenceState) {
    this.state = nextState
    for (const listener of this.listeners) {
      listener(nextState)
    }
  }
}

const onlinePresenceManager = new OnlinePresenceManager()

export function subscribeOnlinePresence(listener: PresenceListener) {
  return onlinePresenceManager.subscribe(listener)
}

if (typeof window !== 'undefined') {
  onlinePresenceManager.bindPageLifecycle()
}
