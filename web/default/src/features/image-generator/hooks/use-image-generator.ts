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
import { generateImages } from '../api'
import { DEFAULT_CONFIG, STORAGE_KEY } from '../constants'
import type {
  GeneratedImage,
  GenerationBatch,
  GeneratorConfig,
  ImageDataItem,
} from '../types'

// Persist prompt/model/group/size choices across sessions.
function loadConfig(): GeneratorConfig {
  try {
    if (typeof window === 'undefined') return { ...DEFAULT_CONFIG }
    const raw = window.localStorage.getItem(STORAGE_KEY)
    if (!raw) return { ...DEFAULT_CONFIG }
    const parsed = JSON.parse(raw) as Partial<GeneratorConfig>
    return { ...DEFAULT_CONFIG, ...parsed }
  } catch {
    return { ...DEFAULT_CONFIG }
  }
}

function persistConfig(config: GeneratorConfig): void {
  try {
    if (typeof window === 'undefined') return
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(config))
  } catch {
    /* empty */
  }
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

    const batchId = genId()
    const batch: GenerationBatch = {
      id: batchId,
      status: 'loading',
      prompt,
      model: config.model,
      size: config.size,
      count: config.n,
      images: [],
      createdAt: Date.now(),
    }
    // Newest batch first.
    setBatches((prev) => [batch, ...prev])
    setIsGenerating(true)

    const controller = new AbortController()
    abortRef.current = controller

    try {
      const isDallE3 = config.model.startsWith('dall-e-3')
      const response = await generateImages(
        {
          model: config.model,
          group: config.group,
          prompt,
          // dall-e-3 only supports n=1; clamp to be safe.
          n: isDallE3 ? 1 : config.n,
          size: config.size,
          ...(isDallE3 ? { quality: config.quality } : {}),
        },
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
