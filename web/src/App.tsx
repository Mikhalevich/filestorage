import { useCallback, useEffect, useRef, useState } from 'react'

import {
  ApiError,
  deleteFile,
  downloadFile,
  listFiles,
  normalizeDir,
  PUBLIC_USER,
  type StoredFile,
} from './api/client'
import { endSession } from './auth'
import { AuthPanel } from './components/AuthPanel'
import { FileTable } from './components/FileTable'
import { PathBar } from './components/PathBar'
import { CreateDirPanel } from './components/CreateDirPanel'
import { UploadPanel } from './components/UploadPanel'
import { useAuth } from './useAuth'
import { useHashDir } from './useHashDir'

type ListResult = { kind: 'ok'; files: StoredFile[] } | { kind: 'error'; status?: number; message: string }

// results are tagged with the request they answer, a result for another request means loading.
type Tagged<T> = { user: string; dir: string; reload: number; value: T }

// Space picks whose files are shown: the signed-in user's own or the shared public ones.
type Space = 'mine' | 'public'

const noHighlight: ReadonlySet<string> = new Set()

const fileKey = (user: string, dir: string, name: string) => normalizeDir(`${user}/${dir}/${name}`)

export default function App() {
  const { session, notice } = useAuth()
  const [space, setSpace] = useState<Space>('mine')
  const [showAuth, setShowAuth] = useState(false)
  const [dir, navigate] = useHashDir()
  const [reload, setReload] = useState(0)
  const [result, setResult] = useState<Tagged<ListResult>>()
  const [uploaded, setUploaded] = useState<{ user: string; dir: string; names: ReadonlySet<string> }>()
  // files being deleted, keyed by user, dir and name, so a pending delete survives navigating away and back.
  const [deleting, setDeleting] = useState<ReadonlySet<string>>(new Set())
  const [fileError, setFileError] = useState<{ user: string; dir: string; message: string }>()

  // without a session only the public files are reachable.
  const user = session !== undefined && space === 'mine' ? session.user : PUBLIC_USER

  // directories belong to a user, so switching users starts from their root.
  const prevUser = useRef(user)
  useEffect(() => {
    if (prevUser.current !== user) {
      prevUser.current = user
      navigate('')
    }
  }, [user, navigate])

  useEffect(() => {
    const controller = new AbortController()
    const tag = (value: ListResult) => setResult({ user, dir, reload, value })

    listFiles(user, dir, controller.signal)
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
  }, [user, dir, reload])

  const onUploaded = useCallback(
    (stored: StoredFile[]) => {
      setUploaded({ user, dir, names: new Set(stored.map((s) => s.name)) })
      setReload((r) => r + 1)
    },
    [user, dir],
  )

  const onDelete = useCallback(
    async (name: string) => {
      if (!window.confirm(`Delete ${name}?`)) {
        return
      }

      const key = fileKey(user, dir, name)
      setDeleting((prev) => new Set(prev).add(key))
      setFileError(undefined)

      try {
        await deleteFile(user, dir, name)
        setReload((r) => r + 1)
      } catch (err) {
        if (err instanceof ApiError && err.status === 404) {
          // already gone: refreshing shows the actual state.
          setReload((r) => r + 1)
        } else {
          setFileError({
            user,
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
    [user, dir],
  )

  const onDownload = useCallback(
    async (name: string) => {
      setFileError(undefined)

      try {
        await downloadFile(user, dir, name)
      } catch (err) {
        setFileError({
          user,
          dir,
          message: err instanceof ApiError ? err.message : 'Download failed: server unreachable',
        })
      }
    },
    [user, dir],
  )

  const onSignOut = () => {
    setSpace('mine')
    endSession()
  }

  const here = (tagged: { user: string; dir: string } | undefined) => tagged?.user === user && tagged.dir === dir

  const loading = result === undefined || !here(result) || result.reload !== reload
  // while reloading the same directory keep showing the previous list instead of flashing "Loading…".
  const shown = result !== undefined && here(result) ? result.value : undefined
  const highlighted = uploaded !== undefined && here(uploaded) ? uploaded.names : noHighlight
  const deletingHere = new Set(
    shown?.kind === 'ok'
      ? shown.files.map((f) => f.name).filter((name) => deleting.has(fileKey(user, dir, name)))
      : [],
  )

  return (
    <div className="app">
      <header className="header">
        <h1>File storage</h1>
        <div className="header-actions">
          {session === undefined ? (
            !showAuth && (
              <button type="button" className="button" onClick={() => setShowAuth(true)}>
                Sign in
              </button>
            )
          ) : (
            <p className="account">
              <span className="muted">Signed in as</span> <strong>{session.user}</strong>{' '}
              <button type="button" className="link-button" onClick={onSignOut}>
                Sign out
              </button>
            </p>
          )}
          <button type="button" className="button" onClick={() => setReload((r) => r + 1)}>
            Refresh
          </button>
        </div>
      </header>

      {notice !== undefined && session === undefined && (
        <div className="notice notice-error" role="alert">
          <p>{notice}</p>
        </div>
      )}

      {session === undefined && showAuth && <AuthPanel onCancel={() => setShowAuth(false)} />}

      {session !== undefined && (
        <div className="spaces" role="tablist" aria-label="Whose files">
          <button
            type="button"
            role="tab"
            className="auth-tab"
            aria-selected={space === 'mine'}
            onClick={() => setSpace('mine')}
          >
            My files
          </button>
          <button
            type="button"
            role="tab"
            className="auth-tab"
            aria-selected={space === 'public'}
            onClick={() => setSpace('public')}
          >
            Public files
          </button>
        </div>
      )}

      {session === undefined && (
        <p className="muted">
          You're browsing the public files. Sign in to keep files of your own.
        </p>
      )}

      <PathBar dir={dir} onNavigate={navigate} />

      <main>
        <UploadPanel key={`${user}:${dir}`} user={user} dir={dir} onUploaded={onUploaded} />
        <CreateDirPanel
          key={`mkdir:${user}:${dir}`}
          user={user}
          dir={dir}
          onOpen={navigate}
          onCreated={() => setReload((r) => r + 1)}
        />

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
          {fileError !== undefined && here(fileError) && (
            <div className="notice notice-error">
              <p>{fileError.message}</p>
            </div>
          )}
          {shown?.kind === 'ok' && (
            <FileTable
              user={user}
              dir={dir}
              files={shown.files}
              highlighted={highlighted}
              deleting={deletingHere}
              onDelete={(name) => void onDelete(name)}
              onOpenDir={(name) => navigate(normalizeDir(`${dir}/${name}`))}
              // public files need no token, so they keep plain download links.
              onDownload={user === PUBLIC_USER ? undefined : (name) => void onDownload(name)}
            />
          )}
        </section>
      </main>
    </div>
  )
}
