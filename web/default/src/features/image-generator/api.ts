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
  ModelOption,
  VideoGenerationRequest,
  VideoSubmitResponse,
  VideoTaskResponse,
} from './types'

/**
 * Get user available models.
 */
export async function getUserModels(): Promise<ModelOption[]> {
  const res = await api.get(API_ENDPOINTS.USER_MODELS)
  const { data } = res

  if (!data.success || !Array.isArray(data.data)) {
    return []
  }

  return data.data.map((model: string) => ({
    label: model,
    value: model,
  }))
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

  return {
    task_id: (raw.task_id as string) || taskId,
    status: (raw.status as string) || '',
    url: (raw.result_url as string) || (raw.url as string) || undefined,
    error: raw.fail_reason
      ? { message: raw.fail_reason as string }
      : undefined,
  } as VideoTaskResponse
}
