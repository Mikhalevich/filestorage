import createClient from 'openapi-fetch'

import { endSession, getAuthState } from '../auth'
import type { components, paths } from './schema'

export type StoredFile = components['schemas']['File']

type UploadBody = paths['/files/{user}/']['post']['requestBody']['content']['multipart/form-data']

type Credentials = components['schemas']['Credentials']

const client = createClient<paths>()

// authHeader returns the Authorization header of the current session, if any.
function authHeader(): Record<string, string> {
  const session = getAuthState().session

  return session === undefined ? {} : { Authorization: `Bearer ${session.token}` }
}

// every request but the auth ones carries the session token. A 401 on a request that sent a token means the token
// is no longer valid, so the session ends; a 401 from /auth/login is just wrong credentials.
client.use({
  onRequest({ request }) {
    const session = getAuthState().session
    if (session !== undefined && !new URL(request.url).pathname.startsWith('/auth/')) {
      request.headers.set('Authorization', `Bearer ${session.token}`)
    }

    return request
  },
  onResponse({ request, response }) {
    if (response.status === 401 && request.headers.has('Authorization')) {
      endSession('Your session is no longer valid. Sign in again.')
    }

    return response
  },
})

// PUBLIC_USER owns the files available without a token, every other user's files need that user's token.
export const PUBLIC_USER = 'public'

// ApiError carries the plain-text message the server sends along with an error status.
export class ApiError extends Error {
  readonly status: number

  constructor(status: number, message: unknown) {
    super(typeof message === 'string' && message.trim() !== '' ? message.trim() : `Request failed (${status})`)
    this.name = 'ApiError'
    this.status = status
  }
}

// normalizeDir trims surrounding slashes: '' is the storage root, 'a/b' is a subdirectory.
export function normalizeDir(dir: string): string {
  return dir
    .split('/')
    .filter((segment) => segment !== '')
    .join('/')
}

export async function listFiles(user: string, dir: string, signal?: AbortSignal): Promise<StoredFile[]> {
  const path = normalizeDir(dir)

  // the user's root has its own route: path params can't be empty.
  const { data, error, response } =
    path === ''
      ? await client.GET('/files/{user}/', { params: { path: { user } }, signal })
      : await client.GET('/files/{user}/{path}', { params: { path: { user, path } }, signal })

  if (data === undefined) {
    throw new ApiError(response.status, error)
  }

  return data
}

export async function uploadFiles(user: string, dir: string, files: File[]): Promise<StoredFile[]> {
  const path = normalizeDir(dir)
  const body: UploadBody = { file: files }

  // fetch sets the multipart Content-Type with boundary itself, so it must not be passed here.
  const bodySerializer = (content: UploadBody): FormData => {
    const form = new FormData()
    for (const file of content.file ?? []) {
      // File keeps its own name, it's used as the stored file name.
      form.append('file', file)
    }

    return form
  }

  const { data, error, response } =
    path === ''
      ? await client.POST('/files/{user}/', { params: { path: { user } }, body, bodySerializer })
      : await client.POST('/files/{user}/{path}', { params: { path: { user, path } }, body, bodySerializer })

  if (data === undefined) {
    throw new ApiError(response.status, error)
  }

  return data
}

export async function deleteFile(user: string, dir: string, name: string): Promise<void> {
  const path = normalizeDir(`${dir}/${name}`)

  const { error, response } = await client.DELETE('/files/{user}/{path}', { params: { path: { user, path } } })

  // 204 carries no body, so success is told by the status rather than by data.
  if (!response.ok) {
    throw new ApiError(response.status, error)
  }
}

export async function createDirectory(user: string, dir: string, name: string): Promise<void> {
  const path = normalizeDir(`${dir}/${name}`)

  const { error, response } = await client.POST('/dirs/{user}/{path}', { params: { path: { user, path } } })

  // 201 carries no body, so success is told by the status rather than by data.
  if (!response.ok) {
    throw new ApiError(response.status, error)
  }
}

// register creates the user and returns the token issued for it.
export async function register(credentials: Credentials): Promise<string> {
  const { data, error, response } = await client.POST('/auth/register', { body: credentials })

  if (data === undefined) {
    throw new ApiError(response.status, error)
  }

  return data.token
}

// login returns the token issued for the user.
export async function login(credentials: Credentials): Promise<string> {
  const { data, error, response } = await client.POST('/auth/login', { body: credentials })

  if (data === undefined) {
    throw new ApiError(response.status, error)
  }

  return data.token
}

// downloadFile fetches a file with the session token and hands it to the browser to save. Links can't send the
// token, so this is how a signed-in user's own files are downloaded. The whole file is held in memory meanwhile.
export async function downloadFile(user: string, dir: string, name: string): Promise<void> {
  const response = await fetch(downloadURL(user, dir, name), { headers: authHeader() })

  if (!response.ok) {
    if (response.status === 401) {
      endSession('Your session is no longer valid. Sign in again.')
    }

    throw new ApiError(response.status, await response.text())
  }

  const url = URL.createObjectURL(await response.blob())

  try {
    const link = document.createElement('a')
    link.href = url
    link.download = name
    link.click()
  } finally {
    // the click starts the download synchronously, the blob URL isn't needed afterwards.
    setTimeout(() => URL.revokeObjectURL(url), 0)
  }
}

// downloadURL builds the link for GET /download/{user}/{path}; the file is streamed by the browser, not fetched.
export function downloadURL(user: string, dir: string, name: string): string {
  const path = normalizeDir(`${user}/${dir}/${name}`)

  return `/download/${path.split('/').map(encodeURIComponent).join('/')}`
}
