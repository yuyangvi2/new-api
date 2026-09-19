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
import type { PricingModel } from '@/features/pricing/types'

export type MarketKind = 'text' | 'image' | 'video' | 'audio'

export type MarketModel = PricingModel & {
  marketKind?: MarketKind
}

export function splitTags(tags?: string): string[] {
  if (!tags) return []
  return tags
    .split(',')
    .map((tag) => tag.trim())
    .filter(Boolean)
}

export function inferKind(model: MarketModel): MarketKind {
  if (model.marketKind) return model.marketKind

  const modelType = Number(model.model_type)
  if (modelType === 3) return 'video'
  if (modelType === 2) return 'image'
  if (modelType === 4) return 'audio'

  const endpoints = (model.supported_endpoint_types ?? []).map((item) =>
    item.toLowerCase()
  )
  const inputModalities = (model.input_modalities ?? []).map((item) =>
    item.toLowerCase()
  )
  const outputModalities = (model.output_modalities ?? []).map((item) =>
    item.toLowerCase()
  )
  const values = [
    ...endpoints,
    ...inputModalities,
    ...outputModalities,
    ...splitTags(model.tags),
    model.model_name,
  ]
    .join(' ')
    .toLowerCase()

  if (
    endpoints.includes('openai-video') ||
    outputModalities.includes('video') ||
    [
      'video',
      'openai-video',
      'seedance',
      'kling',
      'sora',
      'veo',
      'vidu',
      'hailuo',
      'wan2',
      'wan-2',
      'wanx2',
      'wanx-2',
      'dreamina',
    ].some((term) => values.includes(term))
  ) {
    return 'video'
  }
  if (
    outputModalities.includes('audio') ||
    ['audio', 'speech', 'whisper', 'tts', 'stt', 'suno', 'music', 'voice'].some(
      (term) => values.includes(term)
    )
  ) {
    return 'audio'
  }
  if (
    endpoints.includes('image-generation') ||
    outputModalities.includes('image') ||
    [
      'image',
      'images',
      'image-generation',
      'vision',
      'dall-e',
      'dalle',
      'gpt-image',
      'flux',
      'midjourney',
      'jimeng',
      'nano-banana',
      'stable-diffusion',
      'sdxl',
      'ideogram',
      'recraft',
      'kolors',
      'imagen',
    ].some((term) => values.includes(term))
  ) {
    return 'image'
  }
  return 'text'
}

export function toModelGuideSlug(modelName: string): string {
  return modelName
    .trim()
    .toLowerCase()
    .replaceAll(/[^a-z0-9]+/g, '-')
    .replaceAll(/^-+|-+$/g, '')
}

export function marketKindLabelKey(kind: MarketKind): string {
  if (kind === 'image') return 'Image'
  if (kind === 'video') return 'Video'
  if (kind === 'audio') return 'Audio'
  return 'Text'
}
