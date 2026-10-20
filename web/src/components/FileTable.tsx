import { downloadURL, type StoredFile } from '../api/client'
import { formatSize, formatTime } from '../format'

type Props = {
  dir: string
  files: StoredFile[]
  highlighted: ReadonlySet<string>
  deleting: ReadonlySet<string>
  onDelete: (name: string) => void
  onOpenDir: (name: string) => void
}

// FileTable keeps the server order: directories by name, then files oldest first, ties by name.
export function FileTable({ dir, files, highlighted, deleting, onDelete, onOpenDir }: Props) {
  if (files.length === 0) {
    return <p className="empty">This directory is empty.</p>
  }

  return (
    <div className="table-wrap">
      <table className="files">
        <thead>
          <tr>
            <th scope="col">Name</th>
            <th scope="col" className="num">
              Size
            </th>
            <th scope="col">Modified</th>
            <th scope="col">
              <span className="visually-hidden">Actions</span>
            </th>
          </tr>
        </thead>
        <tbody>
          {files.map((f) =>
            f.type === 'directory' ? (
              <tr key={`dir:${f.name}`}>
                <td className="name">
                  <button type="button" className="dir-link" onClick={() => onOpenDir(f.name)}>
                    {f.name}/
                  </button>
                </td>
                <td className="num muted">—</td>
                <td className="time">
                  <time dateTime={f.mod_time}>{formatTime(f.mod_time)}</time>
                </td>
                <td className="actions" />
              </tr>
            ) : (
              <tr key={f.name} className={highlighted.has(f.name) ? 'row-new' : undefined}>
                <td className="name">
                  <a href={downloadURL(dir, f.name)} download={f.name} title={`Download ${f.name}`}>
                    {f.name}
                  </a>
                </td>
                <td className="num">{formatSize(f.size)}</td>
                <td className="time">
                  <time dateTime={f.mod_time}>{formatTime(f.mod_time)}</time>
                </td>
                <td className="actions">
                  <button
                    type="button"
                    className="button button-danger"
                    disabled={deleting.has(f.name)}
                    aria-label={`Delete ${f.name}`}
                    onClick={() => onDelete(f.name)}
                  >
                    {deleting.has(f.name) ? 'Deleting…' : 'Delete'}
                  </button>
                </td>
              </tr>
            ),
          )}
        </tbody>
      </table>
    </div>
  )
}
