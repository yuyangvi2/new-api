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
import { fetchImageTask, generateImages, submitImageTask } from '../api'
import {
  DEFAULT_CONFIG,
  detectImageModelFamily,
  IMAGE_TASK_FAILED_STATUSES,
  IMAGE_TASK_POLL_INTERVAL_MS,
  IMAGE_TASK_POLL_TIMEOUT_MS,
  IMAGE_TASK_SUCCESS_STATUSES,
  isTaskBasedImageModel,
  STORAGE_KEY,
} from '../constants'
import type {
  GeneratedImage,
  GenerationBatch,
  GeneratorConfig,
  ImageDataItem,
} from '../types'

// Persist prompt/model/group/size choices across sessions.
// Never restore large blobs like reference images.
function loadConfig(): GeneratorConfig {
  try {
    if (typeof window === 'undefined') return { ...DEFAULT_CONFIG }
    const raw = window.localStorage.getItem(STORAGE_KEY)
    if (!raw) return { ...DEFAULT_CONFIG }
    const parsed = JSON.parse(raw) as Partial<GeneratorConfig>
    return { ...DEFAULT_CONFIG, ...parsed, images: [] }
  } catch {
    return { ...DEFAULT_CONFIG }
  }
}

function persistConfig(config: GeneratorConfig): void {
  try {
    if (typeof window === 'undefined') return
    // Strip images from persisted config (too large for localStorage)
    const { images: _images, ...rest } = config
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(rest))
  } catch {
    /* empty */
  }
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

function genId(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

function toSrc(item: ImageDataItem): string | null {
  if (item.url) return item.url
  if (item.b64_json) return `data:image/png;base64,${item.b64_json}`
  return null
}

interface UseImageGeneratorResult {
  config: GeneratorConfig
  updateConfig: <K extends keyof GeneratorConfig>(
    key: K,
    value: GeneratorConfig[K]
  ) => void
  batches: GenerationBatch[]
  isGenerating: boolean
  generate: () => Promise<void>
  cancel: () => void
  clearHistory: () => void
}

export function useImageGenerator(): UseImageGeneratorResult {
  const [config, setConfig] = useState<GeneratorConfig>(loadConfig)
  const [batches, setBatches] = useState<GenerationBatch[]>([])
  const [isGenerating, setIsGenerating] = useState(false)
  const abortRef = useRef<AbortController | null>(null)

  const updateConfig = useCallback(
    <K extends keyof GeneratorConfig>(key: K, value: GeneratorConfig[K]) => {
      setConfig((prev) => {
        const next = { ...prev, [key]: value }
        persistConfig(next)
        return next
      })
    },
    []
  )

  const patchBatch = useCallback(
    (id: string, patch: Partial<GenerationBatch>) => {
      setBatches((prev) =>
        prev.map((b) => (b.id === id ? { ...b, ...patch } : b))
      )
    },
    []
  )

  const generate = useCallback(async () => {
    const prompt = config.prompt.trim()
    if (!prompt || isGenerating) return

    const family = detectImageModelFamily(config.model)
    const isTaskBased = isTaskBasedImageModel(family)

    const batchId = genId()
    const batch: GenerationBatch = {
      id: batchId,
      status: 'loading',
      prompt,
      model: config.model,
      size: config.size,
      count: isTaskBased ? 1 : config.n,
      images: [],
      createdAt: Date.now(),
    }
    setBatches((prev) => [batch, ...prev])
    setIsGenerating(true)

    const controller = new AbortController()
    abortRef.current = controller

    try {
      if (isTaskBased) {
        // ---- Task-based flow (image-gi / image-gi2) ----
        const meta: Record<string, unknown> = {}
        if (config.metadata) {
          for (const [k, v] of Object.entries(config.metadata)) {
            if (v !== '' && v !== undefined && v !== null) {
              meta[k] = v
            }
          }
        }

        const submit = await submitImageTask(
          {
            model: config.model,
            group: config.group,
            prompt,
            images: config.images.length > 0
              ? config.images.map(toBackendImage)
              : undefined,
            size: config.size,
            ...(Object.keys(meta).length > 0 ? { metadata: meta } : {}),
          },
          controller.signal
        )

        if (!submit.task_id) {
          throw new Error('No task id returned')
        }

        patchBatch(batchId, { status: 'loading' })

        // Poll until terminal state
        const deadline = Date.now() + IMAGE_TASK_POLL_TIMEOUT_MS
        // eslint-disable-next-line no-constant-condition
        while (true) {
          await sleep(IMAGE_TASK_POLL_INTERVAL_MS, controller.signal)

          const task = await fetchImageTask(submit.task_id, controller.signal)
          const status = (task.status || '').toLowerCase()

          if (IMAGE_TASK_SUCCESS_STATUSES.includes(status)) {
            if (task.url) {
              const img: GeneratedImage = {
                id: genId(),
                src: task.url,
                prompt,
                model: config.model,
                size: config.size,
                createdAt: Date.now(),
              }
              patchBatch(batchId, { status: 'complete', images: [img] })
            } else {
              patchBatch(batchId, {
                status: 'error',
                errorMessage: 'Task succeeded but no image URL was returned',
              })
            }
            break
          }

          if (IMAGE_TASK_FAILED_STATUSES.includes(status)) {
            patchBatch(batchId, {
              status: 'error',
              errorMessage: task.error?.message || 'Image generation failed',
            })
            break
          }

          if (Date.now() > deadline) {
            patchBatch(batchId, {
              status: 'error',
              errorMessage: 'Timed out waiting for image generation',
            })
            break
          }
        }
      } else {
        // ---- Synchronous flow (dall-e, gpt-image, etc.) ----
        const isDallE3 = family === 'dall-e'
        const isGptImage = family === 'gpt-image'

        const payload: Record<string, unknown> = {
          model: config.model,
          group: config.group,
          prompt,
          n: isDallE3 ? 1 : config.n,
          size: config.size,
        }

        if (isDallE3) {
          payload.quality = config.quality
        }

        if (isGptImage) {
          payload.quality = config.quality
          // Pass gpt-image specific metadata (background)
          const bg = config.metadata?.background
          if (bg && bg !== 'auto') {
            payload.background = bg
          }
        }

        const response = await generateImages(
          payload as unknown as Parameters<typeof generateImages>[0],
          controller.signal
        )

        const images: GeneratedImage[] = (response.data ?? [])
          .map((item) => {
            const src = toSrc(item)
            if (!src) return null
            return {
              id: genId(),
              src,
              prompt: item.revised_prompt || prompt,
              model: config.model,
              size: config.size,
              createdAt: Date.now(),
            }
          })
          .filter((x): x is GeneratedImage => x !== null)

        if (images.length === 0) {
          patchBatch(batchId, {
            status: 'error',
            errorMessage: 'No image was returned',
          })
        } else {
          patchBatch(batchId, { status: 'complete', images })
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
          'Image generation failed'
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
