import createClient from 'openapi-fetch'

import type { components, paths } from './schema'

export type StoredFile = components['schemas']['File']

type UploadBody = paths['/files/']['post']['requestBody']['content']['multipart/form-data']

const client = createClient<paths>()

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

export async function listFiles(dir: string, signal?: AbortSignal): Promise<StoredFile[]> {
  const path = normalizeDir(dir)

  // the root has its own route: path params can't be empty.
  const { data, error, response } =
    path === ''
      ? await client.GET('/files/', { signal })
      : await client.GET('/files/{path}', { params: { path: { path } }, signal })

  if (data === undefined) {
    throw new ApiError(response.status, error)
  }

  return data
}

export async function uploadFiles(dir: string, files: File[]): Promise<StoredFile[]> {
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
      ? await client.POST('/files/', { body, bodySerializer })
      : await client.POST('/files/{path}', { params: { path: { path } }, body, bodySerializer })

  if (data === undefined) {
    throw new ApiError(response.status, error)
  }

  return data
}

export async function deleteFile(dir: string, name: string): Promise<void> {
  const path = normalizeDir(`${dir}/${name}`)

  const { error, response } = await client.DELETE('/files/{path}', { params: { path: { path } } })

  // 204 carries no body, so success is told by the status rather than by data.
  if (!response.ok) {
    throw new ApiError(response.status, error)
  }
}

// downloadURL builds the link for GET /download/{path}; the file is streamed by the browser, not fetched.
export function downloadURL(dir: string, name: string): string {
  const path = normalizeDir(`${dir}/${name}`)

  return `/download/${path.split('/').map(encodeURIComponent).join('/')}`
}
