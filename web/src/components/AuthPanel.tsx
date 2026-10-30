import { useState, type FormEvent } from 'react'

import { ApiError, login, register } from '../api/client'
import { startSession } from '../auth'

type Mode = 'login' | 'register'

type Props = {
  onCancel: () => void
}

type Status = { kind: 'idle' } | { kind: 'sending' } | { kind: 'error'; message: string }

// AuthPanel signs in an existing user or registers a new one; either way the issued token starts the session.
export function AuthPanel({ onCancel }: Props) {
  const [mode, setMode] = useState<Mode>('login')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [status, setStatus] = useState<Status>({ kind: 'idle' })

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault()

    if (status.kind === 'sending') {
      return
    }

    setStatus({ kind: 'sending' })

    try {
      const credentials = { username: username.trim(), password }
      startSession(mode === 'login' ? await login(credentials) : await register(credentials))
    } catch (err) {
      setStatus({
        kind: 'error',
        message:
          err instanceof ApiError ? err.message : err instanceof Error ? err.message : 'Request failed: server unreachable',
      })
    }
  }

  const switchMode = (next: Mode) => {
    setMode(next)
    setStatus({ kind: 'idle' })
  }

  return (
    <section className="auth" aria-label={mode === 'login' ? 'Sign in' : 'Create account'}>
      <div className="auth-tabs" role="tablist">
        <button
          type="button"
          role="tab"
          className="auth-tab"
          aria-selected={mode === 'login'}
          onClick={() => switchMode('login')}
        >
          Sign in
        </button>
        <button
          type="button"
          role="tab"
          className="auth-tab"
          aria-selected={mode === 'register'}
          onClick={() => switchMode('register')}
        >
          Create account
        </button>
      </div>

      <form className="auth-form" onSubmit={(e) => void onSubmit(e)}>
        <label htmlFor="auth-username">Username</label>
        <input
          id="auth-username"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          autoComplete="username"
          autoCapitalize="none"
          spellCheck={false}
          required
          // the server checks the format on register only, login just looks the name up.
          pattern={mode === 'register' ? '[a-z0-9][a-z0-9_\\-]{2,31}' : undefined}
          title={
            mode === 'register'
              ? "3-32 lowercase letters, digits, '_' and '-', starting with a letter or digit"
              : undefined
          }
        />

        <label htmlFor="auth-password">Password</label>
        <input
          id="auth-password"
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
          required
          maxLength={72}
        />

        {mode === 'register' && (
          <p className="muted auth-hint">
            Your files are private to you. The name <code>public</code> is reserved for shared files.
          </p>
        )}

        <div className="auth-actions">
          <button type="submit" className="button button-primary" disabled={status.kind === 'sending'}>
            {status.kind === 'sending' ? 'Sending…' : mode === 'login' ? 'Sign in' : 'Create account'}
          </button>
          <button type="button" className="link-button" onClick={onCancel}>
            Cancel
          </button>
        </div>
      </form>

      <div role="status" aria-live="polite">
        {status.kind === 'error' && <p className="notice notice-error">{status.message}</p>}
      </div>
    </section>
  )
}
