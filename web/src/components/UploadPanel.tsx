import { useRef, useState, type DragEvent } from 'react'

import { ApiError, uploadFiles, type StoredFile } from '../api/client'

type Props = {
  user: string
  dir: string
  onUploaded: (stored: StoredFile[]) => void
}

type Status =
  | { kind: 'idle' }
  | { kind: 'uploading'; count: number }
  | { kind: 'done'; renamed: { from: string; to: string }[]; count: number }
  | { kind: 'error'; message: string }

export function UploadPanel({ user, dir, onUploaded }: Props) {
  const inputRef = useRef<HTMLInputElement>(null)
  const [dragging, setDragging] = useState(false)
  const [status, setStatus] = useState<Status>({ kind: 'idle' })

  const upload = async (files: File[]) => {
    if (files.length === 0 || status.kind === 'uploading') {
      return
    }

    setStatus({ kind: 'uploading', count: files.length })

    try {
      const stored = await uploadFiles(user, dir, files)

      // the server answers in request order and renames files whose name is taken.
      const renamed = stored.flatMap((s, i) =>
        files[i] !== undefined && files[i].name !== s.name ? [{ from: files[i].name, to: s.name }] : [],
      )

      setStatus({ kind: 'done', renamed, count: stored.length })
      onUploaded(stored)
    } catch (err) {
      setStatus({ kind: 'error', message: err instanceof ApiError ? err.message : 'Upload failed: server unreachable' })
    }
  }

  const onDrop = (e: DragEvent) => {
    e.preventDefault()
    setDragging(false)
    void upload(Array.from(e.dataTransfer.files))
  }

  return (
    <section className="upload" aria-label="Upload files">
      <div
        className={`dropzone${dragging ? ' dropzone-active' : ''}`}
        onDragOver={(e) => {
          e.preventDefault()
          setDragging(true)
        }}
        onDragLeave={() => setDragging(false)}
        onDrop={onDrop}
      >
        <p>
          Drop files here or{' '}
          <button type="button" className="link-button" onClick={() => inputRef.current?.click()}>
            choose files
          </button>
        </p>
        <p className="muted">
          They go to <code>/{dir}</code>. A file with a taken name is stored under a new one.
        </p>
        <input
          ref={inputRef}
          type="file"
          multiple
          hidden
          onChange={(e) => {
            void upload(Array.from(e.target.files ?? []))
            e.target.value = ''
          }}
        />
      </div>

      <div role="status" aria-live="polite">
        {status.kind === 'uploading' && (
          <p className="notice">
            Uploading {status.count} file{status.count === 1 ? '' : 's'}…
          </p>
        )}
        {status.kind === 'done' && (
          <div className="notice notice-ok">
            <p>
              Uploaded {status.count} file{status.count === 1 ? '' : 's'}.
            </p>
            {status.renamed.length > 0 && (
              <ul>
                {status.renamed.map((r) => (
                  <li key={r.to}>
                    <code>{r.from}</code> was stored as <code>{r.to}</code>
                  </li>
                ))}
              </ul>
            )}
          </div>
        )}
        {status.kind === 'error' && <p className="notice notice-error">{status.message}</p>}
      </div>
    </section>
  )
}
