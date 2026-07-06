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
import { useCallback, useRef, useState } from 'react'

import { fetchVideoTask, submitVideoTask } from '../api'
import {
  DEFAULT_VIDEO_CONFIG,
  VIDEO_FAILED_STATUSES,
  VIDEO_POLL_INTERVAL_MS,
  VIDEO_POLL_TIMEOUT_MS,
  VIDEO_SIZE_PRESETS,
  VIDEO_STORAGE_KEY,
  VIDEO_SUCCESS_STATUSES,
  videoModelRequiresImage,
  videoModelSupportsImageInput,
} from '../constants'
import type { VideoBatch, VideoConfig, VideoTaskResponse } from '../types'

function loadConfig(): VideoConfig {
  try {
    if (typeof window === 'undefined') return { ...DEFAULT_VIDEO_CONFIG }
    const raw = window.localStorage.getItem(VIDEO_STORAGE_KEY)
    if (!raw) return { ...DEFAULT_VIDEO_CONFIG }
    const parsed = JSON.parse(raw) as Partial<VideoConfig>
    // Never restore a previously chosen image — only the lightweight settings.
    return { ...DEFAULT_VIDEO_CONFIG, ...parsed, image: '' }
  } catch {
    return { ...DEFAULT_VIDEO_CONFIG }
  }
}

function persistConfig(config: VideoConfig): void {
  try {
    if (typeof window === 'undefined') return
    const { image: _image, ...rest } = config
    window.localStorage.setItem(VIDEO_STORAGE_KEY, JSON.stringify(rest))
  } catch {
    /* empty */
  }
}

function genId(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

function sleep(ms: number, signal: AbortSignal): Promise<void> {
  return new Promise((resolve, reject) => {
    const timer = setTimeout(resolve, ms)
    signal.addEventListener(
      'abort',
      () => {
        clearTimeout(timer)
        reject(new DOMException('Aborted', 'AbortError'))
      },
      { once: true }
    )
  })
}

type PatchVideoBatch = (id: string, patch: Partial<VideoBatch>) => void

function videoContentUrl(taskId: string): string {
  return `/v1/videos/${taskId}/content`
}

function applyVideoTaskResult(
  batchId: string,
  task: VideoTaskResponse,
  patchBatch: PatchVideoBatch
): boolean {
  const status = (task.status || '').toLowerCase()
  const progressPatch = {
    ...(task.progress ? { progress: task.progress } : {}),
    ...(task.debugResult ? { debugResult: task.debugResult } : {}),
  }

  if (VIDEO_SUCCESS_STATUSES.includes(status)) {
    patchBatch(batchId, {
      ...progressPatch,
      status: 'complete',
      videoUrl: videoContentUrl(task.task_id),
    })
    return true
  }

  if (VIDEO_FAILED_STATUSES.includes(status)) {
    patchBatch(batchId, {
      ...progressPatch,
      status: 'error',
      errorMessage: task.error?.message || 'Video generation failed',
    })
    return true
  }

  if (Object.keys(progressPatch).length > 0) {
    patchBatch(batchId, progressPatch)
  }
  return false
}

async function pollVideoTask(
  taskId: string,
  batchId: string,
  signal: AbortSignal,
  patchBatch: PatchVideoBatch,
  fetchImmediately: boolean
): Promise<void> {
  const deadline = Date.now() + VIDEO_POLL_TIMEOUT_MS

  if (fetchImmediately) {
    const task = await fetchVideoTask(taskId, signal)
    if (applyVideoTaskResult(batchId, task, patchBatch)) {
      return
    }
  }

  while (true) {
    await sleep(VIDEO_POLL_INTERVAL_MS, signal)

    const task = await fetchVideoTask(taskId, signal)
    if (applyVideoTaskResult(batchId, task, patchBatch)) {
      return
    }

    if (Date.now() > deadline) {
      patchBatch(batchId, {
        status: 'error',
        errorMessage: 'Timed out waiting for the video',
      })
      return
    }
  }
}

function extractErrorMessage(error: unknown): string {
  const err = error as {
    response?: {
      data?: { message?: string; error?: { message?: string } }
    }
    message?: string
  }

  return (
    err?.response?.data?.error?.message ||
    err?.response?.data?.message ||
    err?.message ||
    'Video generation failed'
  )
}

function patchVideoTaskError(
  batchId: string,
  error: unknown,
  patchBatch: PatchVideoBatch
): void {
  const err = error as { name?: string }
  if (err?.name === 'CanceledError' || err?.name === 'AbortError') {
    patchBatch(batchId, {
      status: 'error',
      errorMessage: 'Generation cancelled',
    })
    return
  }

  patchBatch(batchId, {
    status: 'error',
    errorMessage: extractErrorMessage(error),
  })
}

interface UseVideoGeneratorResult {
  config: VideoConfig
  updateConfig: <K extends keyof VideoConfig>(
    key: K,
    value: VideoConfig[K]
  ) => void
  batches: VideoBatch[]
  isGenerating: boolean
  generate: () => Promise<void>
  recoverTask: (taskId: string) => void
  cancel: () => void
  clearHistory: () => void
}

export function useVideoGenerator(): UseVideoGeneratorResult {
  const [config, setConfig] = useState<VideoConfig>(loadConfig)
  const [batches, setBatches] = useState<VideoBatch[]>([])
  const [isGenerating, setIsGenerating] = useState(false)
  const abortRef = useRef<AbortController | null>(null)

  const updateConfig = useCallback(
    <K extends keyof VideoConfig>(key: K, value: VideoConfig[K]) => {
      setConfig((prev) => {
        const next = { ...prev, [key]: value }
        persistConfig(next)
        return next
      })
    },
    []
  )

  const patchBatch = useCallback((id: string, patch: Partial<VideoBatch>) => {
    setBatches((prev) =>
      prev.map((b) => (b.id === id ? { ...b, ...patch } : b))
    )
  }, [])

  const generate = useCallback(async () => {
    const prompt = config.prompt.trim()
    const requiresImage = videoModelRequiresImage(config.model)
    const inputImage = videoModelSupportsImageInput(config.model)
      ? config.image
      : ''
    if ((requiresImage && !inputImage) || isGenerating) return

    const batchId = genId()
    const batch: VideoBatch = {
      id: batchId,
      status: 'submitting',
      prompt,
      model: config.model,
      imagePreview: inputImage,
      createdAt: Date.now(),
    }
    setBatches((prev) => [batch, ...prev])
    setIsGenerating(true)

    const controller = new AbortController()
    abortRef.current = controller

    try {
      const preset = VIDEO_SIZE_PRESETS.find(
        (p) => `${p.width}x${p.height}` === config.size
      )

      // Collect non-empty metadata entries for model-family-specific params.
      const meta: Record<string, unknown> = {}
      if (config.metadata) {
        for (const [k, v] of Object.entries(config.metadata)) {
          if (v !== '' && v !== undefined && v !== null) {
            meta[k] = v
          }
        }
      }

      const submit = await submitVideoTask(
        {
          model: config.model,
          group: config.group,
          prompt: prompt || undefined,
          image: inputImage || undefined,
          duration: config.duration,
          width: preset?.width,
          height: preset?.height,
          ...(Object.keys(meta).length > 0 ? { metadata: meta } : {}),
        },
        controller.signal
      )

      if (!submit.task_id) {
        throw new Error('No task id returned')
      }

      patchBatch(batchId, { status: 'polling', taskId: submit.task_id })

      await pollVideoTask(
        submit.task_id,
        batchId,
        controller.signal,
        patchBatch,
        false
      )
    } catch (error: unknown) {
      patchVideoTaskError(batchId, error, patchBatch)
    } finally {
      abortRef.current = null
      setIsGenerating(false)
    }
  }, [config, isGenerating, patchBatch])

  const recoverTask = useCallback(
    (taskIdInput: string) => {
      const taskId = taskIdInput.trim()
      if (!taskId || isGenerating) return

      const batchId = genId()
      const batch: VideoBatch = {
        id: batchId,
        status: 'polling',
        prompt: '',
        model: '',
        taskId,
        createdAt: Date.now(),
      }
      setBatches((prev) => [batch, ...prev.filter((b) => b.taskId !== taskId)])
      setIsGenerating(true)

      const controller = new AbortController()
      abortRef.current = controller

      void pollVideoTask(taskId, batchId, controller.signal, patchBatch, true)
        .catch((error: unknown) => {
          patchVideoTaskError(batchId, error, patchBatch)
        })
        .finally(() => {
          abortRef.current = null
          setIsGenerating(false)
        })
    },
    [isGenerating, patchBatch]
  )

  const cancel = useCallback(() => {
    abortRef.current?.abort()
  }, [])

  const clearHistory = useCallback(() => {
    setBatches([])
  }, [])

  return {
    config,
    updateConfig,
    batches,
    isGenerating,
    generate,
    recoverTask,
    cancel,
    clearHistory,
  }
}
