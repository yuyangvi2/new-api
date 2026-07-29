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

import type { PricingModel } from '../types'

export type DisplayPricingEntry = {
  key: string
  label: string
  formatted: string
  numericPrice: number
  unit: 'second'
}

function formatDisplayPricingValue(price: number): string {
  return formatBillingCurrencyFromUSD(price, {
    digitsLarge: 4,
    digitsSmall: 6,
    abbreviate: false,
  })
}

export function isDisplayPricingModel(model: PricingModel): boolean {
  const config = model.display_pricing
  return (
    config?.mode === 'weighted_factors' &&
    config.unit === 'second' &&
    Number.isFinite(config.base_price) &&
    config.base_price > 0 &&
    Boolean(config.base_values) &&
    Array.isArray(config.factors) &&
    config.factors.length > 0
  )
}

export function formatDisplayBaseSecondPrice(
  model: PricingModel,
  groupRatio: number
): string | null {
  const config = model.display_pricing
  if (!isDisplayPricingModel(model) || !config) return null
  const price = config.base_price * groupRatio
  if (!Number.isFinite(price) || price <= 0) return null
  return formatDisplayPricingValue(price)
}

export function getDisplayPricingEntries(
  model: PricingModel,
  groupRatio: number
): DisplayPricingEntry[] {
  const config = model.display_pricing
  if (!isDisplayPricingModel(model) || !config) {
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
        unit: 'second',
      })
    }
  }
  return entries
}
