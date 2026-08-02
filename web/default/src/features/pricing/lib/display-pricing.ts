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
import { formatBillingCurrencyFromUSD } from '@/lib/currency'

import type { DisplayPricingUnit, PricingModel } from '../types'

export type DisplayPricingEntry = {
  key: string
  label: string
  labelKey?: string
  formatted: string
  numericPrice: number
  unit: DisplayPricingUnit
}

export type TieredSecondPricingEntry = {
  key: string
  label: string
  steps: TieredSecondPricingStep[]
  primaryStep?: TieredSecondPricingStep
  unit: 'second'
}

export type TieredSecondPricingStep = {
  key: string
  label: string
  formatted: string
  numericPrice: number
  fromSecond?: number
  toSecond?: number
}

const DISPLAY_PRICING_LABEL_ALIASES: ReadonlyArray<
  readonly [source: string, key: string]
> = [
  ['Reference to Video', 'Reference to Video'],
  ['Text to Video', 'Text to Video'],
  ['Voice selection', 'Voice selection'],
  ['Not specified', 'Not specified'],
  ['High quality', 'High quality'],
  ['With sound', 'With sound'],
  ['No sound', 'No sound'],
  ['Resolution', 'Resolution'],
  ['Specified', 'Specified'],
  ['Standard', 'Standard'],
  ['Sound', 'Sound'],
  ['Mode', 'Mode'],
  ['参考生视频', 'Reference to Video'],
  ['文生视频', 'Text to Video'],
  ['指定音色', 'Voice selection'],
  ['未指定', 'Not specified'],
  ['高品质', 'High quality'],
  ['有声', 'With sound'],
  ['无声', 'No sound'],
  ['分辨率', 'Resolution'],
  ['指定', 'Specified'],
  ['标准', 'Standard'],
  ['声音', 'Sound'],
  ['模式', 'Mode'],
]

export function localizeDisplayPricingLabel(
  label: string,
  translate: (key: string) => string
): string {
  const directTranslation = translate(label)
  if (directTranslation !== label) return directTranslation

  return DISPLAY_PRICING_LABEL_ALIASES.reduce(
    (localizedLabel, [source, key]) =>
      localizedLabel.replaceAll(source, translate(key)),
    label
  )
}

function normalizedModelName(model: PricingModel): string {
  return model.model_name.trim().toLowerCase()
}

function isGrokImagineQualityImage(modelName: string): boolean {
  return (
    modelName === 'grok-imagine-image-quality' ||
    modelName === 'grok-imagine-image-pro'
  )
}

function getBuiltinDisplayPricingEntries(
  model: PricingModel,
  groupRatio: number
): DisplayPricingEntry[] {
  const modelName = normalizedModelName(model)
  const entry = (
    key: string,
    label: string,
    price: number,
    unit: DisplayPricingUnit
  ): DisplayPricingEntry => {
    const numericPrice = price * groupRatio
    return {
      key,
      label,
      labelKey: label,
      formatted: formatDisplayPricingValue(numericPrice),
      numericPrice,
      unit,
    }
  }

  if (
    modelName === 'grok-imagine' ||
    modelName === 'grok-imagine-image' ||
    isGrokImagineQualityImage(modelName)
  ) {
    if (isGrokImagineQualityImage(modelName)) {
      return [
        entry('output-1k', '1K output', 0.05, 'image'),
        entry('output-2k', '2K output', 0.07, 'image'),
        entry('image-input', 'Image input', 0.01, 'image'),
      ]
    }
    return [
      entry('output', 'Output image', 0.02, 'image'),
      entry('image-input', 'Image input', 0.002, 'image'),
    ]
  }

  if (modelName === 'grok-imagine-video') {
    return [
      entry('image-input', 'Image input', 0.002, 'image'),
      entry('video-input', 'Video input', 0.01, 'second'),
      entry('output-480p', '480p output', 0.05, 'second'),
      entry('output-720p', '720p output', 0.07, 'second'),
    ]
  }

  if (modelName === 'grok-imagine-video-1.5') {
    return [
      entry('image-input', 'Image input', 0.01, 'image'),
      entry('output-480p', '480p output', 0.08, 'second'),
      entry('output-720p', '720p output', 0.14, 'second'),
      entry('output-1080p', '1080p output', 0.25, 'second'),
    ]
  }

  return []
}

function isBuiltinDisplayPricingModel(model: PricingModel): boolean {
  return getBuiltinDisplayPricingEntries(model, 1).length > 0
}

function formatDisplayPricingValue(price: number): string {
  return formatBillingCurrencyFromUSD(price, {
    digitsLarge: 4,
    digitsSmall: 6,
    abbreviate: false,
  })
}

function formatSecondStepLabel(
  step: { label?: string; from_second?: number; to_second?: number },
  index: number
): string {
  if (typeof step.label === 'string' && step.label.trim()) {
    return step.label.trim()
  }

  const fromSecond = step.from_second
  const toSecond = step.to_second

  if (Number.isFinite(fromSecond) && Number.isFinite(toSecond)) {
    return fromSecond === toSecond
      ? `${fromSecond}s`
      : `${fromSecond}-${toSecond}s`
  }

  if (Number.isFinite(fromSecond)) {
    return `${fromSecond}s+`
  }

  return `${index + 1}s`
}

export function isWeightedFactorsDisplayPricingModel(
  model: PricingModel
): boolean {
  const config = model.display_pricing
  return (
    config?.mode === 'weighted_factors' &&
    (config.unit === 'second' || config.unit === 'request') &&
    Number.isFinite(config.base_price) &&
    config.base_price > 0 &&
    Boolean(config.base_values) &&
    Array.isArray(config.factors) &&
    config.factors.length > 0
  )
}

export function isTieredSecondsDisplayPricingModel(
  model: PricingModel
): boolean {
  const config = model.display_pricing
  return (
    config?.mode === 'tiered_seconds' &&
    config.unit === 'second' &&
    Array.isArray(config.tiers) &&
    getTieredSecondPricingEntries(model, 1).length > 0
  )
}

export function isDisplayPricingModel(model: PricingModel): boolean {
  return (
    isBuiltinDisplayPricingModel(model) ||
    isWeightedFactorsDisplayPricingModel(model) ||
    isTieredSecondsDisplayPricingModel(model)
  )
}

export function formatDisplayBasePrice(
  model: PricingModel,
  groupRatio: number
): string | null {
  const builtinEntries = getBuiltinDisplayPricingEntries(model, groupRatio)
  if (builtinEntries.length > 0) {
    return builtinEntries[0]?.formatted ?? null
  }

  const config = model.display_pricing
  if (!config) return null

  if (isTieredSecondsDisplayPricingModel(model)) {
    const tier = getTieredSecondPricingEntries(model, groupRatio)[0]
    return tier?.primaryStep?.formatted ?? null
  }

  if (!isWeightedFactorsDisplayPricingModel(model)) return null
  if (config.mode !== 'weighted_factors') return null
  const price = config.base_price * groupRatio
  if (!Number.isFinite(price) || price <= 0) return null
  return formatDisplayPricingValue(price)
}

export function getTieredSecondPricingEntries(
  model: PricingModel,
  groupRatio: number
): TieredSecondPricingEntry[] {
  const config = model.display_pricing
  if (!config || config.mode !== 'tiered_seconds' || config.unit !== 'second') {
    return []
  }

  const entries = config.tiers.flatMap((tier) => {
    if (
      typeof tier.value !== 'string' ||
      typeof tier.label !== 'string' ||
      !Array.isArray(tier.steps)
    ) {
      return []
    }

    const steps = tier.steps.flatMap((step, index) => {
      if (!Number.isFinite(step.price) || step.price <= 0) return []

      const numericPrice = step.price * groupRatio
      if (!Number.isFinite(numericPrice) || numericPrice <= 0) return []

      return [
        {
          key: `${tier.value}:${index}`,
          label: formatSecondStepLabel(step, index),
          formatted: formatDisplayPricingValue(numericPrice),
          numericPrice,
          fromSecond: step.from_second,
          toSecond: step.to_second,
        },
      ]
    })
    if (steps.length === 0) {
      return []
    }

    return [
      {
        key: tier.value,
        label: tier.label,
        steps,
        primaryStep: steps[0],
        unit: 'second' as const,
      },
    ]
  })

  if (typeof config.base_value === 'string' && config.base_value) {
    const baseIndex = entries.findIndex(
      (entry) => entry.key === config.base_value
    )
    if (baseIndex > 0) {
      return [
        entries[baseIndex],
        ...entries.slice(0, baseIndex),
        ...entries.slice(baseIndex + 1),
      ]
    }
  }

  return entries
}

export function getDisplayPricingEntries(
  model: PricingModel,
  groupRatio: number
): DisplayPricingEntry[] {
  const builtinEntries = getBuiltinDisplayPricingEntries(model, groupRatio)
  if (builtinEntries.length > 0) {
    return builtinEntries
  }

  const config = model.display_pricing
  if (!config) {
    return []
  }

  if (isTieredSecondsDisplayPricingModel(model)) {
    return getTieredSecondPricingEntries(model, groupRatio).flatMap((tier) =>
      tier.steps.map((step) => ({
        key: step.key,
        label: `${tier.label}: ${step.label}`,
        formatted: step.formatted,
        numericPrice: step.numericPrice,
        unit: tier.unit,
      }))
    )
  }

  if (!isWeightedFactorsDisplayPricingModel(model)) {
    return []
  }
  if (config.mode !== 'weighted_factors') {
    return []
  }

  const baseWeights = new Map<string, number>()
  for (const factor of config.factors) {
    if (
      typeof factor.field !== 'string' ||
      typeof factor.label !== 'string' ||
      !Array.isArray(factor.values)
    ) {
      return []
    }
    const baseValue = config.base_values[factor.field]
    const option = factor.values.find((value) => value.value === baseValue)
    if (!option || !Number.isFinite(option.weight) || option.weight <= 0) {
      return []
    }
    baseWeights.set(factor.field, option.weight)
  }

  const baseWeight = [...baseWeights.values()].reduce(
    (product, weight) => product * weight,
    1
  )
  if (!Number.isFinite(baseWeight) || baseWeight <= 0) return []

  const entries: DisplayPricingEntry[] = []
  for (const factor of config.factors) {
    for (const option of factor.values) {
      if (!Number.isFinite(option.weight) || option.weight <= 0) continue
      const factorBaseWeight = baseWeights.get(factor.field)
      if (!factorBaseWeight) continue

      const selectedWeight = (baseWeight / factorBaseWeight) * option.weight
      const numericPrice =
        (config.base_price * selectedWeight * groupRatio) / baseWeight
      if (!Number.isFinite(numericPrice) || numericPrice <= 0) continue

      entries.push({
        key: `${factor.field}:${option.value}`,
        label: `${factor.label}: ${option.label} ×${option.weight}`,
        formatted: formatDisplayPricingValue(numericPrice),
        numericPrice,
        unit: config.unit,
      })
    }
  }
  return entries
}
