import { useEffect, useState } from 'react'
import { subscribeOnlinePresence, type OnlinePresenceState } from '../services/onlinePresence'

const initialState: OnlinePresenceState = { count: 0, status: 'connecting' }

export function useOnlinePresence() {
  const [presence, setPresence] = useState(initialState)

  useEffect(() => subscribeOnlinePresence(setPresence), [])

  return presence
}
