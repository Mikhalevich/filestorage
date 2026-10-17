import { useCallback, useEffect, useState } from 'react'

import { ApiError, listFiles, type StoredFile } from './api/client'
import { FileTable } from './components/FileTable'
import { PathBar } from './components/PathBar'
import { UploadPanel } from './components/UploadPanel'
import { useHashDir } from './useHashDir'

type ListResult = { kind: 'ok'; files: StoredFile[] } | { kind: 'error'; status?: number; message: string }

// results are tagged with the request they answer, a result for another request means loading.
type Tagged<T> = { dir: string; reload: number; value: T }

const noHighlight: ReadonlySet<string> = new Set()

export default function App() {
  const [dir, navigate] = useHashDir()
  const [reload, setReload] = useState(0)
  const [result, setResult] = useState<Tagged<ListResult>>()
  const [uploaded, setUploaded] = useState<{ dir: string; names: ReadonlySet<string> }>()

  useEffect(() => {
    const controller = new AbortController()
    const tag = (value: ListResult) => setResult({ dir, reload, value })

    listFiles(dir, controller.signal)
      .then((files) => tag({ kind: 'ok', files }))
      .catch((err: unknown) => {
        if (controller.signal.aborted) {
          return
        }

        tag(
          err instanceof ApiError
            ? { kind: 'error', status: err.status, message: err.message }
            : { kind: 'error', message: 'Server unreachable' },
        )
      })

    return () => controller.abort()
  }, [dir, reload])

  const onUploaded = useCallback(
    (stored: StoredFile[]) => {
      setUploaded({ dir, names: new Set(stored.map((s) => s.name)) })
      setReload((r) => r + 1)
    },
    [dir],
  )

  const loading = result === undefined || result.dir !== dir || result.reload !== reload
  // while reloading the same directory keep showing the previous list instead of flashing "Loading…".
  const shown = result !== undefined && result.dir === dir ? result.value : undefined
  const highlighted = uploaded?.dir === dir ? uploaded.names : noHighlight

  return (
    <div className="app">
      <header className="header">
        <h1>File storage</h1>
        <button type="button" className="button" onClick={() => setReload((r) => r + 1)}>
          Refresh
        </button>
      </header>

      <PathBar dir={dir} onNavigate={navigate} />

      <main>
        <UploadPanel key={dir} dir={dir} onUploaded={onUploaded} />

        <section aria-label="Files" aria-busy={loading}>
          {shown === undefined && <p className="muted">Loading…</p>}
          {shown?.kind === 'error' && (
            <div className="notice notice-error">
              <p>{shown.message}</p>
              {shown.status === 404 && dir !== '' && (
                <button type="button" className="link-button" onClick={() => navigate('')}>
                  Go to root
                </button>
              )}
            </div>
          )}
          {shown?.kind === 'ok' && <FileTable dir={dir} files={shown.files} highlighted={highlighted} />}
        </section>
      </main>
    </div>
  )
}
