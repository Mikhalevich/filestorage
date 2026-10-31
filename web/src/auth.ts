// The session is the token issued by /auth/register or /auth/login. It's kept in localStorage so it
// survives reloads, and dropped when it expires or the server rejects it.

export type Session = {
  user: string
  token: string
  // expiresAt is the token's exp claim in milliseconds.
  expiresAt: number
}

export type AuthState = {
  session: Session | undefined
  // notice explains why the session ended when it wasn't the user's choice.
  notice: string | undefined
}

const storageKey = 'filestorage.token'

let state: AuthState = { session: undefined, notice: undefined }
let expiryTimer: ReturnType<typeof setTimeout> | undefined
const listeners = new Set<() => void>()

// parseToken reads sub and exp from the JWT payload. The signature is the server's business, the UI only
// needs to know whose token it is and when it stops working.
function parseToken(token: string): Session | undefined {
  try {
    const payload = token.split('.')[1]
    if (payload === undefined) {
      return undefined
    }

    const json: unknown = JSON.parse(atob(payload.replace(/-/g, '+').replace(/_/g, '/')))
    if (typeof json !== 'object' || json === null) {
      return undefined
    }

    const { sub, exp } = json as { sub?: unknown; exp?: unknown }
    if (typeof sub !== 'string' || sub === '' || typeof exp !== 'number') {
      return undefined
    }

    return { user: sub, token, expiresAt: exp * 1000 }
  } catch {
    return undefined
  }
}

function readStorage(): string | null {
  try {
    return window.localStorage.getItem(storageKey)
  } catch {
    return null
  }
}

function writeStorage(token: string | undefined) {
  try {
    if (token === undefined) {
      window.localStorage.removeItem(storageKey)
    } else {
      window.localStorage.setItem(storageKey, token)
    }
  } catch {
    // storage may be unavailable (private mode), the session then lasts until reload.
  }
}

function update(next: AuthState) {
  state = next

  clearTimeout(expiryTimer)
  expiryTimer = undefined

  const session = next.session
  if (session !== undefined) {
    // setTimeout overflows after ~24.8 days, tokens live much shorter.
    expiryTimer = setTimeout(
      () => endSession('Your session expired. Sign in again.'),
      Math.min(session.expiresAt - Date.now(), 2 ** 31 - 1),
    )
  }

  for (const listener of listeners) {
    listener()
  }
}

export function startSession(token: string) {
  const session = parseToken(token)
  if (session === undefined) {
    throw new Error('The server returned an unreadable token')
  }

  writeStorage(token)
  update({ session, notice: undefined })
}

// endSession signs out, notice is shown to explain an involuntary sign-out.
export function endSession(notice?: string) {
  writeStorage(undefined)
  update({ session: undefined, notice })
}

export function getAuthState(): AuthState {
  return state
}

export function subscribeAuth(listener: () => void): () => void {
  listeners.add(listener)

  return () => listeners.delete(listener)
}

// restore the session saved by a previous page load, unless it has expired.
{
  const saved = readStorage()
  const session = saved === null ? undefined : parseToken(saved)

  if (session !== undefined && session.expiresAt > Date.now()) {
    update({ session, notice: undefined })
  } else if (saved !== null) {
    writeStorage(undefined)
  }

  // another tab signed in or out.
  window.addEventListener('storage', (e) => {
    if (e.key !== storageKey) {
      return
    }

    update({ session: e.newValue === null ? undefined : parseToken(e.newValue), notice: undefined })
  })
}
