/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { api } from '@/lib/api'

import { API_ENDPOINTS } from './constants'
import type {
  GroupOption,
  ImageGenerationRequest,
  ImageGenerationResponse,
  ImageTaskRequest,
  ModelOption,
  VideoGenerationRequest,
  VideoSubmitResponse,
  VideoTaskResponse,
} from './types'

type MediaUploadKind = 'image' | 'video' | 'audio'

type RawUserModel =
  | string
  | {
      label?: string
      value?: string
      groups?: string[]
      model_ratio?: number
      modelRatio?: number
    }

function normalizeTaskProgress(progress: unknown): string | undefined {
  if (progress === undefined || progress === null) return undefined
  const normalized = String(progress).trim()
  return normalized || undefined
}

function recordFrom(value: unknown): Record<string, unknown> | undefined {
  return value && typeof value === 'object' && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : undefined
}

function stringFrom(value: unknown): string | undefined {
  if (typeof value !== 'string') return undefined
  const trimmed = value.trim()
  return trimmed || undefined
}

function firstMessage(...values: unknown[]): string | undefined {
  for (const value of values) {
    const message = stringFrom(value)
    if (message) return message
  }
  return undefined
}

function extractTaskErrorMessage(
  raw: Record<string, unknown>
): string | undefined {
  const rawError = recordFrom(raw.error)

  return firstMessage(raw.fail_reason, rawError?.message, raw.message)
}

/**
 * Get user available models.
 */
export async function getUserModels(): Promise<ModelOption[]> {
  const res = await api.get(API_ENDPOINTS.USER_MODELS, {
    params: { with_groups: true },
  })
  const { data } = res

  if (!data.success || !Array.isArray(data.data)) {
    return []
  }

  return (data.data as RawUserModel[])
    .map((model): ModelOption | null => {
      if (typeof model === 'string') {
        return {
          label: model,
          value: model,
        }
      }
      const value = model.value || model.label
      if (!value) return null
      return {
        label: model.label || value,
        value,
        groups: Array.isArray(model.groups) ? model.groups : undefined,
        modelRatio:
          typeof model.model_ratio === 'number'
            ? model.model_ratio
            : model.modelRatio,
      }
    })
    .filter((model: ModelOption | null): model is ModelOption => model !== null)
}

/**
 * Get user groups.
 */
export async function getUserGroups(): Promise<GroupOption[]> {
  const res = await api.get(API_ENDPOINTS.USER_GROUPS)
  const { data } = res

  if (!data.success || !data.data) {
    return []
  }

  const groupData = data.data as Record<string, { desc: string; ratio: number }>

  return Object.entries(groupData).map(([group, info]) => ({
    label: group,
    value: group,
    ratio: info.ratio,
    desc: info.desc,
  }))
}

/**
 * Generate images via the playground proxy. Authenticated by the login
 * session (cookie) — no API key needed; a temp token is created server-side.
 */
export async function generateImages(
  payload: ImageGenerationRequest,
  signal?: AbortSignal
): Promise<ImageGenerationResponse> {
  const res = await api.post(API_ENDPOINTS.IMAGE_GENERATIONS, payload, {
    skipErrorHandler: true,
    skipBusinessError: true,
    signal,
  } as Record<string, unknown>)
  return res.data as ImageGenerationResponse
}

/**
 * Submit a task-based image generation (for async models like image-gi).
 * Uses the same task API as video generation.
 */
export async function submitImageTask(
  payload: ImageTaskRequest,
  signal?: AbortSignal
): Promise<VideoSubmitResponse> {
  const res = await api.post(API_ENDPOINTS.IMAGE_TASK_SUBMIT, payload, {
    skipErrorHandler: true,
    skipBusinessError: true,
    signal,
  } as Record<string, unknown>)
  return res.data as VideoSubmitResponse
}

/**
 * Poll a task-based image generation task's status by id.
 */
export async function fetchImageTask(
  taskId: string,
  signal?: AbortSignal
): Promise<VideoTaskResponse> {
  const res = await api.get(API_ENDPOINTS.IMAGE_TASK(taskId), {
    skipErrorHandler: true,
    skipBusinessError: true,
    disableDuplicate: true,
    signal,
  } as Record<string, unknown>)

  const raw = (res.data?.data ?? res.data) as Record<string, unknown>
  const errorMessage = extractTaskErrorMessage(raw)

  return {
    task_id: (raw.task_id as string) || taskId,
    status: (raw.status as string) || '',
    progress: normalizeTaskProgress(raw.progress),
    url: (raw.result_url as string) || (raw.url as string) || undefined,
    error: errorMessage ? { message: errorMessage } : undefined,
  } as VideoTaskResponse
}

/**
 * Submit an image-to-video task via the playground proxy (session auth).
 * Returns a task id to poll.
 */
export async function submitVideoTask(
  payload: VideoGenerationRequest,
  signal?: AbortSignal
): Promise<VideoSubmitResponse> {
  const res = await api.post(API_ENDPOINTS.VIDEO_SUBMIT, payload, {
    skipErrorHandler: true,
    skipBusinessError: true,
    signal,
  } as Record<string, unknown>)
  return res.data as VideoSubmitResponse
}

/**
 * Poll a video task's status by id.
 *
 * The backend wraps the task payload inside `{ code, message, data }`.
 * We unwrap `.data` and normalise `result_url` → `url` so the polling
 * loop can consume a flat {@link VideoTaskResponse}.
 */
export async function fetchVideoTask(
  taskId: string,
  signal?: AbortSignal
): Promise<VideoTaskResponse> {
  const res = await api.get(API_ENDPOINTS.VIDEO_TASK(taskId), {
    skipErrorHandler: true,
    skipBusinessError: true,
    disableDuplicate: true,
    signal,
  } as Record<string, unknown>)

  // Unwrap the standard { code, message, data } envelope.
  const raw = (res.data?.data ?? res.data) as Record<string, unknown>
  const errorMessage = extractTaskErrorMessage(raw)

  return {
    task_id: (raw.task_id as string) || taskId,
    status: (raw.status as string) || '',
    progress: normalizeTaskProgress(raw.progress),
    url: (raw.result_url as string) || (raw.url as string) || undefined,
    error: errorMessage ? { message: errorMessage } : undefined,
  } as VideoTaskResponse
}

export async function uploadReferenceMedia(
  file: File,
  kind: MediaUploadKind,
  signal?: AbortSignal
): Promise<string> {
  const formData = new FormData()
  formData.append('file', file)

  const res = await api.post(API_ENDPOINTS.MEDIA_UPLOAD, formData, {
    params: { kind },
    skipBusinessError: true,
    signal,
  } as Record<string, unknown>)

  const data = res.data as {
    success?: boolean
    message?: string
    data?: { url?: string }
  }
  if (!data.success || !data.data?.url) {
    throw new Error(data.message || 'Upload failed')
  }
  return data.data.url
}
