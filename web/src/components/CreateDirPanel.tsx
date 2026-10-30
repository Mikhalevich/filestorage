import { useState, type FormEvent } from 'react'

import { ApiError, createDirectory, normalizeDir } from '../api/client'

type Props = {
  user: string
  dir: string
  onOpen: (dir: string) => void
  onCreated: () => void
}

type Status =
  | { kind: 'idle' }
  | { kind: 'creating' }
  | { kind: 'done'; path: string }
  | { kind: 'error'; message: string }

// CreateDirPanel creates a subdirectory of dir and offers a link to open it.
export function CreateDirPanel({ user, dir, onOpen, onCreated }: Props) {
  const [name, setName] = useState('')
  const [status, setStatus] = useState<Status>({ kind: 'idle' })

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault()

    const trimmed = normalizeDir(name)
    if (trimmed === '' || status.kind === 'creating') {
      return
    }

    setStatus({ kind: 'creating' })

    try {
      await createDirectory(user, dir, trimmed)
      setStatus({ kind: 'done', path: normalizeDir(`${dir}/${trimmed}`) })
      setName('')
      onCreated()
    } catch (err) {
      setStatus({
        kind: 'error',
        message: err instanceof ApiError ? err.message : 'Create failed: server unreachable',
      })
    }
  }

  return (
    <section aria-label="Create directory">
      <form className="mkdir" onSubmit={(e) => void onSubmit(e)}>
        <label htmlFor="mkdir-name" className="visually-hidden">
          New directory name
        </label>
        <input
          id="mkdir-name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="new directory"
          spellCheck={false}
          autoComplete="off"
        />
        <button type="submit" className="button" disabled={normalizeDir(name) === '' || status.kind === 'creating'}>
          Create directory
        </button>
      </form>

      <div role="status" aria-live="polite">
        {status.kind === 'creating' && <p className="notice">Creating…</p>}
        {status.kind === 'done' && (
          <p className="notice notice-ok">
            Created <code>/{status.path}</code>.{' '}
            <button type="button" className="link-button" onClick={() => onOpen(status.path)}>
              Open
            </button>
          </p>
        )}
        {status.kind === 'error' && <p className="notice notice-error">{status.message}</p>}
      </div>
    </section>
  )
}
