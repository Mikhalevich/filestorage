import { useCallback, useEffect, useState } from 'react'

import { ApiError, deleteFile, listFiles, normalizeDir, type StoredFile } from './api/client'
import { FileTable } from './components/FileTable'
import { PathBar } from './components/PathBar'
import { CreateDirPanel } from './components/CreateDirPanel'
import { UploadPanel } from './components/UploadPanel'
import { useHashDir } from './useHashDir'

type ListResult = { kind: 'ok'; files: StoredFile[] } | { kind: 'error'; status?: number; message: string }

// results are tagged with the request they answer, a result for another request means loading.
type Tagged<T> = { dir: string; reload: number; value: T }

const noHighlight: ReadonlySet<string> = new Set()

const deleteKey = (dir: string, name: string) => normalizeDir(`${dir}/${name}`)

export default function App() {
  const [dir, navigate] = useHashDir()
  const [reload, setReload] = useState(0)
  const [result, setResult] = useState<Tagged<ListResult>>()
  const [uploaded, setUploaded] = useState<{ dir: string; names: ReadonlySet<string> }>()
  // files being deleted, keyed by dir and name, so a pending delete survives navigating away and back.
  const [deleting, setDeleting] = useState<ReadonlySet<string>>(new Set())
  const [deleteError, setDeleteError] = useState<{ dir: string; message: string }>()

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

  const onDelete = useCallback(
    async (name: string) => {
      if (!window.confirm(`Delete ${name}?`)) {
        return
      }

      const key = deleteKey(dir, name)
      setDeleting((prev) => new Set(prev).add(key))
      setDeleteError(undefined)

      try {
        await deleteFile(dir, name)
        setReload((r) => r + 1)
      } catch (err) {
        if (err instanceof ApiError && err.status === 404) {
          // already gone: refreshing shows the actual state.
          setReload((r) => r + 1)
        } else {
          setDeleteError({
            dir,
            message: err instanceof ApiError ? err.message : 'Delete failed: server unreachable',
          })
        }
      } finally {
        setDeleting((prev) => {
          const next = new Set(prev)
          next.delete(key)

          return next
        })
      }
    },
    [dir],
  )

  const loading = result === undefined || result.dir !== dir || result.reload !== reload
  // while reloading the same directory keep showing the previous list instead of flashing "Loading…".
  const shown = result !== undefined && result.dir === dir ? result.value : undefined
  const highlighted = uploaded?.dir === dir ? uploaded.names : noHighlight
  const deletingHere = new Set(
    shown?.kind === 'ok' ? shown.files.map((f) => f.name).filter((name) => deleting.has(deleteKey(dir, name))) : [],
  )

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
        <CreateDirPanel key={`mkdir:${dir}`} dir={dir} onOpen={navigate} onCreated={() => setReload((r) => r + 1)} />

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
          {deleteError?.dir === dir && (
            <div className="notice notice-error">
              <p>{deleteError.message}</p>
            </div>
          )}
          {shown?.kind === 'ok' && (
            <FileTable
              dir={dir}
              files={shown.files}
              highlighted={highlighted}
              deleting={deletingHere}
              onDelete={(name) => void onDelete(name)}
              onOpenDir={(name) => navigate(normalizeDir(`${dir}/${name}`))}
            />
          )}
        </section>
      </main>
    </div>
  )
}
