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

export type SeedanceOfficialPriceTier = {
  key: string
  resolutionLabelKey: string
  noVideoLabelKey: string
  videoInputLabelKey: string
  noVideoPriceCNY: number
  videoInputPriceCNY: number
  primary?: boolean
}

export type SeedanceOfficialPriceEntry = {
  key: string
  labelKey: string
  formatted: string
  unit: 'M'
}

type SeedanceOfficialPriceOptions = {
  groupRatio?: number
  usdExchangeRate: number
  showRechargePrice?: boolean
  priceRate?: number
}

const SEEDANCE_STANDARD_TIERS: SeedanceOfficialPriceTier[] = [
  {
    key: 'base',
    resolutionLabelKey: '480p/720p',
    noVideoLabelKey: '480p/720p no video input',
    videoInputLabelKey: '480p/720p video input',
    noVideoPriceCNY: 46,
    videoInputPriceCNY: 28,
    primary: true,
  },
  {
    key: '1080p',
    resolutionLabelKey: '1080p',
    noVideoLabelKey: '1080p no video input',
    videoInputLabelKey: '1080p video input',
    noVideoPriceCNY: 51,
    videoInputPriceCNY: 31,
  },
  {
    key: '4k',
    resolutionLabelKey: '4K',
    noVideoLabelKey: '4K no video input',
    videoInputLabelKey: '4K video input',
    noVideoPriceCNY: 26,
    videoInputPriceCNY: 16,
  },
]

const SEEDANCE_2_5_TIERS: SeedanceOfficialPriceTier[] = [
  {
    key: 'base',
    resolutionLabelKey: '480p/720p',
    noVideoLabelKey: '480p/720p no video input',
    videoInputLabelKey: '480p/720p video input',
    noVideoPriceCNY: 70,
    videoInputPriceCNY: 42,
    primary: true,
  },
  {
    key: '1080p',
    resolutionLabelKey: '1080p',
    noVideoLabelKey: '1080p no video input',
    videoInputLabelKey: '1080p video input',
    noVideoPriceCNY: 77,
    videoInputPriceCNY: 46,
  },
]

const SEEDANCE_FAST_TIERS: SeedanceOfficialPriceTier[] = [
  {
    key: 'base',
    resolutionLabelKey: '480p/720p',
    noVideoLabelKey: '480p/720p no video input',
    videoInputLabelKey: '480p/720p video input',
    noVideoPriceCNY: 37,
    videoInputPriceCNY: 22,
    primary: true,
  },
]

const SEEDANCE_MINI_TIERS: SeedanceOfficialPriceTier[] = [
  {
    key: 'base',
    resolutionLabelKey: '480p/720p',
    noVideoLabelKey: '480p/720p no video input',
    videoInputLabelKey: '480p/720p video input',
    noVideoPriceCNY: 23,
    videoInputPriceCNY: 14,
    primary: true,
  },
]

const SEEDANCE_OFFICIAL_PRICE_TIERS: Record<
  string,
  SeedanceOfficialPriceTier[]
> = {
  'doubao-seedance-2-5': SEEDANCE_2_5_TIERS,
  'doubao-seedance-2.0': SEEDANCE_STANDARD_TIERS,
  'doubao-seedance-2-0-260128': SEEDANCE_STANDARD_TIERS,
  'doubao-seedance-2-0-fast': SEEDANCE_FAST_TIERS,
  'doubao-seedance-2-0-fast-260128': SEEDANCE_FAST_TIERS,
  'doubao-seedance-2-0-mini': SEEDANCE_MINI_TIERS,
  'doubao-seedance-2-0-mini-260615': SEEDANCE_MINI_TIERS,
}

export function getSeedanceOfficialPriceTiers(
  modelName: string
): SeedanceOfficialPriceTier[] | null {
  return SEEDANCE_OFFICIAL_PRICE_TIERS[modelName.toLowerCase().trim()] ?? null
}

export function formatSeedanceOfficialPrice(
  priceCNY: number,
  options: SeedanceOfficialPriceOptions
): string {
  const safeExchangeRate =
    Number.isFinite(options.usdExchangeRate) && options.usdExchangeRate > 0
      ? options.usdExchangeRate
      : 7.14
  const groupRatio = options.groupRatio ?? 1
  const priceRate =
    Number.isFinite(options.priceRate) && Number(options.priceRate) > 0
      ? Number(options.priceRate)
      : 1

  let priceUSD = (priceCNY * groupRatio) / safeExchangeRate
  if (options.showRechargePrice) {
    priceUSD = (priceUSD * priceRate) / safeExchangeRate
  }

  return formatBillingCurrencyFromUSD(priceUSD, {
    digitsLarge: 4,
    digitsSmall: 6,
    abbreviate: false,
  })
}

export function buildSeedanceOfficialPriceEntries(
  modelName: string,
  options: SeedanceOfficialPriceOptions
): {
  primaryEntries: SeedanceOfficialPriceEntry[]
  extraEntries: SeedanceOfficialPriceEntry[]
  officialEntries: SeedanceOfficialPriceEntry[]
} | null {
  const tiers = getSeedanceOfficialPriceTiers(modelName)
  if (!tiers) return null

  const primaryEntries: SeedanceOfficialPriceEntry[] = []
  const extraEntries: SeedanceOfficialPriceEntry[] = []
  const officialEntries: SeedanceOfficialPriceEntry[] = []

  for (const tier of tiers) {
    const targetEntries = tier.primary ? primaryEntries : extraEntries
    targetEntries.push(
      {
        key: `${tier.key}-no-video`,
        labelKey: tier.noVideoLabelKey,
        formatted: formatSeedanceOfficialPrice(tier.noVideoPriceCNY, options),
        unit: 'M',
      },
      {
        key: `${tier.key}-video-input`,
        labelKey: tier.videoInputLabelKey,
        formatted: formatSeedanceOfficialPrice(
          tier.videoInputPriceCNY,
          options
        ),
        unit: 'M',
      }
    )

    officialEntries.push(
      {
        key: `official-${tier.key}-no-video`,
        labelKey: tier.noVideoLabelKey,
        formatted: formatSeedanceOfficialPrice(tier.noVideoPriceCNY, {
          ...options,
          groupRatio: 1,
        }),
        unit: 'M',
      },
      {
        key: `official-${tier.key}-video-input`,
        labelKey: tier.videoInputLabelKey,
        formatted: formatSeedanceOfficialPrice(tier.videoInputPriceCNY, {
          ...options,
          groupRatio: 1,
        }),
        unit: 'M',
      }
    )
  }

  return { primaryEntries, extraEntries, officialEntries }
}
