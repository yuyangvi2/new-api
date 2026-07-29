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
  formatted: string
  numericPrice: number
  unit: DisplayPricingUnit
}

export type TieredSecondPricingEntry = {
  key: string
  label: string
  firstSecondFormatted: string
  additionalSecondFormatted: string
  firstSecondPrice: number
  additionalSecondPrice: number
  unit: 'second'
}

function formatDisplayPricingValue(price: number): string {
  return formatBillingCurrencyFromUSD(price, {
    digitsLarge: 4,
    digitsSmall: 6,
    abbreviate: false,
  })
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
    config.tiers.some(
      (tier) =>
        typeof tier.value === 'string' &&
        typeof tier.label === 'string' &&
        Number.isFinite(tier.first_second_price) &&
        tier.first_second_price > 0 &&
        Number.isFinite(tier.additional_second_price) &&
        tier.additional_second_price > 0
    )
  )
}

export function isDisplayPricingModel(model: PricingModel): boolean {
  return (
    isWeightedFactorsDisplayPricingModel(model) ||
    isTieredSecondsDisplayPricingModel(model)
  )
}

export function formatDisplayBasePrice(
  model: PricingModel,
  groupRatio: number
): string | null {
  const config = model.display_pricing
  if (!config) return null

  if (isTieredSecondsDisplayPricingModel(model)) {
    const tier = getTieredSecondPricingEntries(model, groupRatio)[0]
    return tier?.firstSecondFormatted ?? null
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
  if (
    !isTieredSecondsDisplayPricingModel(model) ||
    !config ||
    config.mode !== 'tiered_seconds'
  ) {
    return []
  }

  return config.tiers.flatMap((tier) => {
    if (
      typeof tier.value !== 'string' ||
      typeof tier.label !== 'string' ||
      !Number.isFinite(tier.first_second_price) ||
      tier.first_second_price <= 0 ||
      !Number.isFinite(tier.additional_second_price) ||
      tier.additional_second_price <= 0
    ) {
      return []
    }

    const firstSecondPrice = tier.first_second_price * groupRatio
    const additionalSecondPrice = tier.additional_second_price * groupRatio
    if (
      !Number.isFinite(firstSecondPrice) ||
      firstSecondPrice <= 0 ||
      !Number.isFinite(additionalSecondPrice) ||
      additionalSecondPrice <= 0
    ) {
      return []
    }

    return [
      {
        key: tier.value,
        label: tier.label,
        firstSecondFormatted: formatDisplayPricingValue(firstSecondPrice),
        additionalSecondFormatted: formatDisplayPricingValue(
          additionalSecondPrice
        ),
        firstSecondPrice,
        additionalSecondPrice,
        unit: 'second' as const,
      },
    ]
  })
}

export function getDisplayPricingEntries(
  model: PricingModel,
  groupRatio: number
): DisplayPricingEntry[] {
  const config = model.display_pricing
  if (!config) {
    return []
  }

  if (isTieredSecondsDisplayPricingModel(model)) {
    return getTieredSecondPricingEntries(model, groupRatio).flatMap((tier) => [
      {
        key: `${tier.key}:first`,
        label: `${tier.label}: 1s`,
        formatted: tier.firstSecondFormatted,
        numericPrice: tier.firstSecondPrice,
        unit: tier.unit,
      },
      {
        key: `${tier.key}:additional`,
        label: `${tier.label}: 2s+`,
        formatted: tier.additionalSecondFormatted,
        numericPrice: tier.additionalSecondPrice,
        unit: tier.unit,
      },
    ])
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
