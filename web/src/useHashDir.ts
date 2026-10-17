import { useCallback, useEffect, useState } from 'react'

import { normalizeDir } from './api/client'

// The current directory lives in the URL hash (#/sub/dir) so it survives reloads and works with back/forward.
function readHash(): string {
  return normalizeDir(decodeURIComponent(window.location.hash.replace(/^#/, '')))
}

export function useHashDir(): [string, (dir: string) => void] {
  const [dir, setDir] = useState(readHash)

  useEffect(() => {
    const onHashChange = () => setDir(readHash())
    window.addEventListener('hashchange', onHashChange)

    return () => window.removeEventListener('hashchange', onHashChange)
  }, [])

  const navigate = useCallback((next: string) => {
    window.location.hash = `/${normalizeDir(next).split('/').map(encodeURIComponent).join('/')}`
  }, [])

  return [dir, navigate]
}
