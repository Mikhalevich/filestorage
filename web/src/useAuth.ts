import { useSyncExternalStore } from 'react'

import { getAuthState, subscribeAuth, type AuthState } from './auth'

export function useAuth(): AuthState {
  return useSyncExternalStore(subscribeAuth, getAuthState)
}
