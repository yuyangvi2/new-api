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
} from '../constants'
import type { VideoBatch, VideoConfig } from '../types'

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

// Strip a data: URI down to raw base64; leave plain URLs untouched.
function toBackendImage(src: string): string {
  if (src.startsWith('data:')) {
    const comma = src.indexOf(',')
    return comma >= 0 ? src.slice(comma + 1) : src
  }
  return src
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

interface UseVideoGeneratorResult {
  config: VideoConfig
  updateConfig: <K extends keyof VideoConfig>(
    key: K,
    value: VideoConfig[K]
  ) => void
  batches: VideoBatch[]
  isGenerating: boolean
  generate: () => Promise<void>
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
    setBatches((prev) => prev.map((b) => (b.id === id ? { ...b, ...patch } : b)))
  }, [])

  const generate = useCallback(async () => {
    const prompt = config.prompt.trim()
    if (!config.image || isGenerating) return

    const batchId = genId()
    const batch: VideoBatch = {
      id: batchId,
      status: 'submitting',
      prompt,
      model: config.model,
      imagePreview: config.image,
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
          image: toBackendImage(config.image),
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

      // Poll until the task reaches a terminal state or times out.
      const deadline = Date.now() + VIDEO_POLL_TIMEOUT_MS
      while (true) {
        await sleep(VIDEO_POLL_INTERVAL_MS, controller.signal)

        const task = await fetchVideoTask(submit.task_id, controller.signal)
        const status = (task.status || '').toLowerCase()
        if (task.progress) {
          patchBatch(batchId, { progress: task.progress })
        }

        if (VIDEO_SUCCESS_STATUSES.includes(status)) {
          if (task.url) {
            patchBatch(batchId, { status: 'complete', videoUrl: task.url })
          } else {
            patchBatch(batchId, {
              status: 'error',
              errorMessage: 'Task succeeded but no video URL was returned',
            })
          }
          break
        }

        if (VIDEO_FAILED_STATUSES.includes(status)) {
          patchBatch(batchId, {
            status: 'error',
            errorMessage: task.error?.message || 'Video generation failed',
          })
          break
        }

        if (Date.now() > deadline) {
          patchBatch(batchId, {
            status: 'error',
            errorMessage: 'Timed out waiting for the video',
          })
          break
        }
      }
    } catch (error: unknown) {
      const err = error as {
        name?: string
        response?: {
          data?: { message?: string; error?: { message?: string } }
        }
        message?: string
      }
      if (err?.name === 'CanceledError' || err?.name === 'AbortError') {
        patchBatch(batchId, {
          status: 'error',
          errorMessage: 'Generation cancelled',
        })
      } else {
        const message =
          err?.response?.data?.error?.message ||
          err?.response?.data?.message ||
          err?.message ||
          'Video generation failed'
        patchBatch(batchId, { status: 'error', errorMessage: message })
      }
    } finally {
      abortRef.current = null
      setIsGenerating(false)
    }
  }, [config, isGenerating, patchBatch])

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
    cancel,
    clearHistory,
  }
}
