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
  coverTone?: string
}

export const FALLBACK_MODELS: MarketModel[] = [
  {
    id: 1,
    model_name: 'gemini-3.5-flash',
    vendor_id: 1,
    vendor_name: 'Google',
    quota_type: 0,
    model_ratio: 0.6,
    completion_ratio: 1,
    enable_groups: ['default'],
    tags: 'Text to Image,Chat,Image,OpenAI',
    marketKind: 'text',
    coverTone: 'from-blue-950 via-blue-700 to-cyan-300',
  },
  {
    id: 2,
    model_name: 'gemini-3.1-pro',
    vendor_id: 1,
    vendor_name: 'Google',
    quota_type: 0,
    model_ratio: 0.8,
    completion_ratio: 1,
    enable_groups: ['default'],
    tags: 'Text to Image,Chat,Image',
    marketKind: 'text',
    coverTone: 'from-black via-slate-950 to-blue-950',
  },
  {
    id: 3,
    model_name: 'gemini-3.5-thinking',
    vendor_id: 1,
    vendor_name: 'Google',
    quota_type: 0,
    model_ratio: 0.9,
    completion_ratio: 1,
    enable_groups: ['default'],
    tags: 'Text to Image,Chat,Reasoning',
    marketKind: 'text',
    coverTone: 'from-emerald-900 via-lime-500 to-yellow-300',
  },
  {
    id: 4,
    model_name: 'gpt-5.4',
    vendor_id: 2,
    vendor_name: 'OpenAI',
    quota_type: 0,
    model_ratio: 1.2,
    completion_ratio: 1,
    enable_groups: ['default'],
    tags: 'Chat,Reasoning,Function calling',
    marketKind: 'text',
    coverTone: 'from-fuchsia-600 via-rose-500 to-orange-300',
  },
  {
    id: 5,
    model_name: 'seedance-2.5',
    vendor_id: 3,
    vendor_name: 'ByteDance',
    quota_type: 1,
    model_ratio: 1,
    completion_ratio: 1,
    model_price: 5,
    enable_groups: ['default'],
    tags: 'Video,Chat,Creative',
    marketKind: 'video',
    coverTone: 'from-orange-500 via-amber-200 to-stone-100',
  },
  {
    id: 6,
    model_name: 'wanx-image-plus',
    vendor_id: 4,
    vendor_name: 'Alibaba',
    quota_type: 1,
    model_ratio: 1,
    completion_ratio: 1,
    model_price: 2,
    enable_groups: ['default'],
    tags: 'Image,Text to Image,Creative',
    marketKind: 'image',
    coverTone: 'from-violet-600 via-sky-500 to-emerald-300',
  },
]

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
