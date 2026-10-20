import { useState, type FormEvent } from 'react'

import { normalizeDir } from '../api/client'

type Props = {
  dir: string
  onNavigate: (dir: string) => void
}

// PathBar shows breadcrumbs of the current directory and lets the user jump to any path.
export function PathBar({ dir, onNavigate }: Props) {
  const [draft, setDraft] = useState(dir)
  const [prevDir, setPrevDir] = useState(dir)

  if (dir !== prevDir) {
    setPrevDir(dir)
    setDraft(dir)
  }

  const segments = dir === '' ? [] : dir.split('/')

  const onSubmit = (e: FormEvent) => {
    e.preventDefault()
    onNavigate(normalizeDir(draft))
  }

  return (
    <div className="pathbar">
      <nav aria-label="Current directory" className="crumbs">
        <button type="button" className="crumb" onClick={() => onNavigate('')} disabled={segments.length === 0}>
          root
        </button>
        {segments.map((segment, i) => {
          const isLast = i === segments.length - 1

          return (
            <span key={i} className="crumb-item">
              <span className="crumb-sep" aria-hidden="true">
                /
              </span>
              <button
                type="button"
                className="crumb"
                onClick={() => onNavigate(segments.slice(0, i + 1).join('/'))}
                disabled={isLast}
                aria-current={isLast ? 'location' : undefined}
              >
                {segment}
              </button>
            </span>
          )
        })}
      </nav>

      <form className="goto" onSubmit={onSubmit}>
        <label htmlFor="goto-path" className="visually-hidden">
          Directory path
        </label>
        <span className="goto-prefix" aria-hidden="true">
          /
        </span>
        <input
          id="goto-path"
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          placeholder="sub/dir"
          spellCheck={false}
          autoComplete="off"
        />
        <button type="submit" className="button">
          Open
        </button>
      </form>
    </div>
  )
}
