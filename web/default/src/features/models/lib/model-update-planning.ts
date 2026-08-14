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
import type { ModelSettings } from '@/features/system-settings/types'

import type { Model } from '../types'

export type ModelPricingMode = 'per-token' | 'per-request'

export type ModelPricingFormValues = {
  price?: string
  ratio?: string
  cacheRatio?: string
  completionRatio?: string
  imageRatio?: string
  audioRatio?: string
  audioCompletionRatio?: string
}

export type ModelPricingSettings = Pick<
  ModelSettings,
  | 'ModelPrice'
  | 'ModelRatio'
  | 'CacheRatio'
  | 'CompletionRatio'
  | 'ImageRatio'
  | 'AudioRatio'
  | 'AudioCompletionRatio'
>

export type ModelPricingUpdate = {
  key: keyof ModelPricingSettings
  value: string
}

export type ModelPayloadForComparison = Pick<
  Model,
  | 'model_name'
  | 'description'
  | 'icon'
  | 'tags'
  | 'vendor_id'
  | 'endpoints'
  | 'name_rule'
  | 'status'
  | 'sync_official'
>

type JsonRecord = Record<string, unknown>

function stableStringify(value: unknown): string {
  if (Array.isArray(value)) {
    return `[${value.map((item) => stableStringify(item)).join(',')}]`
  }
  if (value && typeof value === 'object') {
    const entries = Object.entries(value as JsonRecord).sort(([a], [b]) =>
      a.localeCompare(b)
    )
    const body = entries
      .map(([key, item]) => `${JSON.stringify(key)}:${stableStringify(item)}`)
      .join(',')
    return `{${body}}`
  }
  return JSON.stringify(value)
}

function normalizeJsonString(value: string | undefined): string {
  const trimmed = value?.trim() ?? ''
  if (!trimmed) return ''

  try {
    return stableStringify(JSON.parse(trimmed))
  } catch {
    return trimmed
  }
}

function normalizePricingMapString(value: string | undefined): string {
  const trimmed = value?.trim() ?? ''
  if (!trimmed) return stableStringify({})

  try {
    const parsed = JSON.parse(trimmed)
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
      return stableStringify(parsed)
    }
  } catch {
    return trimmed
  }

  return trimmed
}

function parsePricingMap(value: string | undefined): Record<string, number> {
  const trimmed = value?.trim() ?? ''
  if (!trimmed) return {}

  try {
    const parsed = JSON.parse(trimmed)
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
      return parsed as Record<string, number>
    }
  } catch {
    return {}
  }

  return {}
}

function setNumberIfPresent(
  target: Record<string, number>,
  modelName: string,
  rawValue: string | undefined
) {
  if (!rawValue) return

  const value = Number.parseFloat(rawValue)
  if (!Number.isNaN(value)) {
    target[modelName] = value
  }
}

export function hasModelPricingInput(values: ModelPricingFormValues): boolean {
  return Boolean(
    values.price ||
    values.ratio ||
    values.cacheRatio ||
    values.completionRatio ||
    values.imageRatio ||
    values.audioRatio ||
    values.audioCompletionRatio
  )
}

function normalizeOptionalId(value: number | undefined): number | undefined {
  if (!value) return undefined
  return value
}

function normalizeTags(value: string | undefined): string {
  return (value ?? '')
    .split(',')
    .map((tag) => tag.trim())
    .filter(Boolean)
    .join(',')
}

function appendPricingUpdate(
  updates: ModelPricingUpdate[],
  key: keyof ModelPricingSettings,
  previousValue: string,
  nextMap: Record<string, number>
) {
  const nextValue = stableStringify(nextMap)
  if (nextValue !== normalizePricingMapString(previousValue)) {
    updates.push({ key, value: nextValue })
  }
}

export function buildModelPricingUpdates(params: {
  modelSettings: ModelPricingSettings
  isEditing: boolean
  oldModelName: string
  finalModelName: string
  pricingMode: ModelPricingMode
  values: ModelPricingFormValues
}): ModelPricingUpdate[] {
  const priceMap = parsePricingMap(params.modelSettings.ModelPrice)
  const ratioMap = parsePricingMap(params.modelSettings.ModelRatio)
  const cacheMap = parsePricingMap(params.modelSettings.CacheRatio)
  const completionMap = parsePricingMap(params.modelSettings.CompletionRatio)
  const imageMap = parsePricingMap(params.modelSettings.ImageRatio)
  const audioMap = parsePricingMap(params.modelSettings.AudioRatio)
  const audioCompletionMap = parsePricingMap(
    params.modelSettings.AudioCompletionRatio
  )

  if (
    params.isEditing &&
    params.oldModelName &&
    params.oldModelName !== params.finalModelName
  ) {
    delete priceMap[params.oldModelName]
    delete ratioMap[params.oldModelName]
    delete cacheMap[params.oldModelName]
    delete completionMap[params.oldModelName]
    delete imageMap[params.oldModelName]
    delete audioMap[params.oldModelName]
    delete audioCompletionMap[params.oldModelName]
  }

  delete priceMap[params.finalModelName]
  delete ratioMap[params.finalModelName]
  delete cacheMap[params.finalModelName]
  delete completionMap[params.finalModelName]
  delete imageMap[params.finalModelName]
  delete audioMap[params.finalModelName]
  delete audioCompletionMap[params.finalModelName]

  if (params.pricingMode === 'per-request') {
    setNumberIfPresent(priceMap, params.finalModelName, params.values.price)
  } else {
    setNumberIfPresent(ratioMap, params.finalModelName, params.values.ratio)
    setNumberIfPresent(
      cacheMap,
      params.finalModelName,
      params.values.cacheRatio
    )
    setNumberIfPresent(
      completionMap,
      params.finalModelName,
      params.values.completionRatio
    )
    setNumberIfPresent(
      imageMap,
      params.finalModelName,
      params.values.imageRatio
    )
    setNumberIfPresent(
      audioMap,
      params.finalModelName,
      params.values.audioRatio
    )
    setNumberIfPresent(
      audioCompletionMap,
      params.finalModelName,
      params.values.audioCompletionRatio
    )
  }

  const updates: ModelPricingUpdate[] = []
  appendPricingUpdate(
    updates,
    'ModelPrice',
    params.modelSettings.ModelPrice,
    priceMap
  )
  appendPricingUpdate(
    updates,
    'ModelRatio',
    params.modelSettings.ModelRatio,
    ratioMap
  )
  appendPricingUpdate(
    updates,
    'CacheRatio',
    params.modelSettings.CacheRatio,
    cacheMap
  )
  appendPricingUpdate(
    updates,
    'CompletionRatio',
    params.modelSettings.CompletionRatio,
    completionMap
  )
  appendPricingUpdate(
    updates,
    'ImageRatio',
    params.modelSettings.ImageRatio,
    imageMap
  )
  appendPricingUpdate(
    updates,
    'AudioRatio',
    params.modelSettings.AudioRatio,
    audioMap
  )
  appendPricingUpdate(
    updates,
    'AudioCompletionRatio',
    params.modelSettings.AudioCompletionRatio,
    audioCompletionMap
  )
  return updates
}

export function hasModelPayloadChanged(
  currentModel: Model,
  nextPayload: ModelPayloadForComparison
): boolean {
  return (
    currentModel.model_name !== nextPayload.model_name ||
    (currentModel.description ?? '') !== (nextPayload.description ?? '') ||
    (currentModel.icon ?? '') !== (nextPayload.icon ?? '') ||
    normalizeTags(currentModel.tags) !== normalizeTags(nextPayload.tags) ||
    normalizeOptionalId(currentModel.vendor_id) !==
      normalizeOptionalId(nextPayload.vendor_id) ||
    normalizeJsonString(currentModel.endpoints) !==
      normalizeJsonString(nextPayload.endpoints) ||
    currentModel.name_rule !== nextPayload.name_rule ||
    currentModel.status !== nextPayload.status ||
    currentModel.sync_official !== nextPayload.sync_official
  )
}
