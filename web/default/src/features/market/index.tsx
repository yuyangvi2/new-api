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
import { Link } from '@tanstack/react-router'
import {
  ChevronLeft,
  ChevronRight,
  Copy,
  FileText,
  ImageIcon,
  Layers3,
  Music2,
  Search,
  Video,
} from 'lucide-react'
import {
  useDeferredValue,
  useEffect,
  useMemo,
  useState,
  type ComponentType,
  type MouseEvent,
  type ReactNode,
} from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { CurrencyToggle } from '@/components/currency-toggle'
import { PublicLayout } from '@/components/layout'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { formatBillingCurrencyFromUSD } from '@/lib/currency'
import { getLobeIcon } from '@/lib/lobe-icon'
import { cn } from '@/lib/utils'

import { DEFAULT_TOKEN_UNIT } from '../pricing/constants'
import { usePricingData } from '../pricing/hooks/use-pricing-data'
import {
  getDynamicDisplayGroupRatio,
  getDynamicPricingSummary,
} from '../pricing/lib/dynamic-price'
import {
  getDisplayGroupRatio,
  isTokenBasedModel,
} from '../pricing/lib/model-helpers'
import {
  formatFixedPrice,
  formatGroupPrice,
  formatPrice,
  formatRequestPrice,
} from '../pricing/lib/price'
import {
  FALLBACK_MODELS,
  inferKind,
  marketKindLabelKey,
  splitTags,
  toModelGuideSlug,
  type MarketKind,
  type MarketModel,
} from './lib/model-catalog'

const MODEL_TYPES: Array<{
  value: MarketKind
  label: string
  icon: ComponentType<{ className?: string }>
}> = [
  { value: 'text', label: 'Text', icon: FileText },
  { value: 'image', label: 'Image', icon: ImageIcon },
  { value: 'video', label: 'Video', icon: Video },
  { value: 'audio', label: 'Audio', icon: Music2 },
]

type MarketKindFilter = 'all' | MarketKind

const MODEL_TYPE_FILTERS: Array<{
  value: MarketKindFilter
  label: string
  icon: ComponentType<{ className?: string }>
}> = [{ value: 'all', label: 'All', icon: Layers3 }, ...MODEL_TYPES]

const MARKET_PAGE_SIZE = 18

type MarketSortOption = 'recommended' | 'name' | 'price-low' | 'context-high'

const MARKET_SORT_OPTIONS: Array<{ value: MarketSortOption; label: string }> = [
  { value: 'recommended', label: 'Recommended' },
  { value: 'name', label: 'Name' },
  { value: 'price-low', label: 'Price: Low to High' },
  { value: 'context-high', label: 'Context' },
]

type IndexedMarketModel = MarketModel & {
  marketKind: MarketKind
  searchText: string
}

const MODEL_SORT_COLLATOR = new Intl.Collator(undefined, {
  numeric: true,
  sensitivity: 'base',
})

function compareOptionalText(left?: string, right?: string): number {
  if (left && !right) return -1
  if (!left && right) return 1
  return MODEL_SORT_COLLATOR.compare(left ?? '', right ?? '')
}

function compareMarketModels(
  left: IndexedMarketModel,
  right: IndexedMarketModel
): number {
  const vendorCompare = compareOptionalText(left.vendor_name, right.vendor_name)
  if (vendorCompare !== 0) return vendorCompare

  const nameCompare = MODEL_SORT_COLLATOR.compare(
    left.model_name ?? '',
    right.model_name ?? ''
  )
  if (nameCompare !== 0) return nameCompare

  return (left.id ?? 0) - (right.id ?? 0)
}

function getMarketSortPrice(model: IndexedMarketModel): number {
  const groupRatio = getDisplayGroupRatio(model)

  if (model.billing_mode === 'tiered_expr' && model.billing_expr) {
    const summary = getDynamicPricingSummary(model, {
      tokenUnit: DEFAULT_TOKEN_UNIT,
      groupRatioMultiplier: groupRatio,
    })
    const primaryPrice =
      summary?.primaryEntries[0]?.value ?? summary?.entries[0]?.value
    if (
      typeof primaryPrice === 'number' &&
      Number.isFinite(primaryPrice) &&
      primaryPrice > 0
    ) {
      return primaryPrice * groupRatio
    }
  }

  if (isTokenBasedModel(model)) {
    return model.model_ratio * 2 * groupRatio
  }

  return (model.model_price ?? Number.POSITIVE_INFINITY) * groupRatio
}

function sortMarketModels(
  models: IndexedMarketModel[],
  sortBy: MarketSortOption
): IndexedMarketModel[] {
  const sorted = [...models]

  if (sortBy === 'price-low') {
    sorted.sort((left, right) => {
      const priceCompare = getMarketSortPrice(left) - getMarketSortPrice(right)
      if (priceCompare !== 0) return priceCompare
      return compareMarketModels(left, right)
    })
    return sorted
  }

  if (sortBy === 'context-high') {
    sorted.sort((left, right) => {
      const contextCompare =
        Number(right.context_length ?? 0) - Number(left.context_length ?? 0)
      if (contextCompare !== 0) return contextCompare
      return compareMarketModels(left, right)
    })
    return sorted
  }

  if (sortBy === 'name') {
    sorted.sort((left, right) => {
      const nameCompare = MODEL_SORT_COLLATOR.compare(
        left.model_name ?? '',
        right.model_name ?? ''
      )
      if (nameCompare !== 0) return nameCompare
      return compareMarketModels(left, right)
    })
    return sorted
  }

  sorted.sort(compareMarketModels)
  return sorted
}

function modelSignals(model: MarketModel): string[] {
  return [
    model.model_name,
    model.description,
    model.vendor_name,
    model.tags,
    ...(model.supported_endpoint_types ?? []),
    ...(model.input_modalities ?? []),
    ...(model.output_modalities ?? []),
    ...(model.capabilities ?? []),
  ].filter(Boolean) as string[]
}

type MarketPriceEntry = {
  key: string
  labelKey: string
  formatted: string
  unit: 'M' | 'request'
}

type SeedancePriceTier = {
  key: string
  noVideoLabelKey: string
  videoInputLabelKey: string
  noVideoPriceCNY: number
  videoInputPriceCNY: number
  primary?: boolean
}

const SEEDANCE_OFFICIAL_PRICE_TIERS: Record<string, SeedancePriceTier[]> = {
  'doubao-seedance-2.0': [
    {
      key: 'base',
      noVideoLabelKey: '480p/720p no video input',
      videoInputLabelKey: '480p/720p video input',
      noVideoPriceCNY: 46,
      videoInputPriceCNY: 28,
      primary: true,
    },
    {
      key: '1080p',
      noVideoLabelKey: '1080p no video input',
      videoInputLabelKey: '1080p video input',
      noVideoPriceCNY: 51,
      videoInputPriceCNY: 31,
    },
    {
      key: '4k',
      noVideoLabelKey: '4K no video input',
      videoInputLabelKey: '4K video input',
      noVideoPriceCNY: 26,
      videoInputPriceCNY: 16,
    },
  ],
  'doubao-seedance-2-0-fast': [
    {
      key: 'base',
      noVideoLabelKey: 'No video input',
      videoInputLabelKey: 'Video input',
      noVideoPriceCNY: 37,
      videoInputPriceCNY: 22,
      primary: true,
    },
  ],
  'doubao-seedance-2-0-mini': [
    {
      key: 'base',
      noVideoLabelKey: 'No video input',
      videoInputLabelKey: 'Video input',
      noVideoPriceCNY: 23,
      videoInputPriceCNY: 14,
      primary: true,
    },
  ],
}

function formatSeedanceOfficialPrice(
  priceCNY: number,
  groupRatio: number,
  usdExchangeRate: number
): string {
  const safeExchangeRate =
    Number.isFinite(usdExchangeRate) && usdExchangeRate > 0
      ? usdExchangeRate
      : 7.14

  return formatBillingCurrencyFromUSD(
    (priceCNY * groupRatio) / safeExchangeRate,
    {
      digitsLarge: 4,
      digitsSmall: 6,
      abbreviate: false,
    }
  )
}

function buildSeedanceOfficialEntries(
  model: MarketModel,
  groupRatio: number,
  usdExchangeRate: number
): {
  primaryEntries: MarketPriceEntry[]
  extraEntries: MarketPriceEntry[]
  officialEntries: MarketPriceEntry[]
} | null {
  const tiers =
    SEEDANCE_OFFICIAL_PRICE_TIERS[model.model_name.toLowerCase().trim()]
  if (!tiers) return null

  const primaryEntries: MarketPriceEntry[] = []
  const extraEntries: MarketPriceEntry[] = []
  const officialEntries: MarketPriceEntry[] = []

  for (const tier of tiers) {
    const targetEntries = tier.primary ? primaryEntries : extraEntries
    targetEntries.push(
      {
        key: `${tier.key}-no-video`,
        labelKey: tier.noVideoLabelKey,
        formatted: formatSeedanceOfficialPrice(
          tier.noVideoPriceCNY,
          groupRatio,
          usdExchangeRate
        ),
        unit: 'M',
      },
      {
        key: `${tier.key}-video-input`,
        labelKey: tier.videoInputLabelKey,
        formatted: formatSeedanceOfficialPrice(
          tier.videoInputPriceCNY,
          groupRatio,
          usdExchangeRate
        ),
        unit: 'M',
      }
    )

    officialEntries.push(
      {
        key: `official-${tier.key}-no-video`,
        labelKey: tier.noVideoLabelKey,
        formatted: formatSeedanceOfficialPrice(
          tier.noVideoPriceCNY,
          1,
          usdExchangeRate
        ),
        unit: 'M',
      },
      {
        key: `official-${tier.key}-video-input`,
        labelKey: tier.videoInputLabelKey,
        formatted: formatSeedanceOfficialPrice(
          tier.videoInputPriceCNY,
          1,
          usdExchangeRate
        ),
        unit: 'M',
      }
    )
  }

  return { primaryEntries, extraEntries, officialEntries }
}

function formatCompactTokenCount(value?: number): string | null {
  const numericValue = Number(value ?? 0)
  if (!Number.isFinite(numericValue) || numericValue <= 0) return null

  if (numericValue >= 1_000_000) {
    const compactValue = numericValue / 1_000_000
    const formatted = Number.isInteger(compactValue)
      ? compactValue.toFixed(0)
      : compactValue.toFixed(1).replace(/\.0$/, '')
    return `${formatted}M`
  }

  if (numericValue >= 1_000) {
    return `${Math.round(numericValue / 1_000)}K`
  }

  return Math.round(numericValue).toString()
}

function getSavingPercent(groupRatio: number): number | null {
  if (!Number.isFinite(groupRatio) || groupRatio <= 0 || groupRatio >= 0.995) {
    return null
  }

  return Math.max(1, Math.round((1 - groupRatio) * 100))
}

function isMutedPriceEntry(entry: MarketPriceEntry): boolean {
  const normalizedLabel = entry.labelKey.toLowerCase()
  return entry.key.includes('cache') || normalizedLabel.includes('cache')
}

function formatMarketPriceValue(formatted: string): string {
  const match = formatted.match(/^([^\d-]*)([-\d,]+(?:\.\d+)?)(.*)$/)
  if (!match) return formatted

  const [, prefix, rawNumber, suffix] = match
  const numericValue = Number(rawNumber.replaceAll(',', ''))
  if (!Number.isFinite(numericValue)) return formatted

  const absValue = Math.abs(numericValue)
  let fractionDigits = 2
  if (absValue > 0 && absValue < 0.0001) {
    fractionDigits = 6
  } else if (absValue > 0 && absValue < 1) {
    fractionDigits = 4
  }

  const rounded = numericValue
    .toFixed(fractionDigits)
    .replace(/\.0+$/, '')
    .replace(/(\.\d*?)0+$/, '$1')

  return `${prefix}${rounded}${suffix}`
}

function MarketPricePanel(props: {
  model: MarketModel
  priceRate: number
  usdExchangeRate: number
}) {
  const { t } = useTranslation()
  const model = props.model
  const displayGroupRatio = getDisplayGroupRatio(model)
  const savingPercent = getSavingPercent(displayGroupRatio)
  const isDynamicPricing =
    model.billing_mode === 'tiered_expr' && Boolean(model.billing_expr)
  const dynamicSummary = isDynamicPricing
    ? getDynamicPricingSummary(model, {
        tokenUnit: DEFAULT_TOKEN_UNIT,
        showRechargePrice: false,
        priceRate: props.priceRate,
        usdExchangeRate: props.usdExchangeRate,
        groupRatioMultiplier: getDynamicDisplayGroupRatio(model),
      })
    : null
  const seedanceOfficialEntries = buildSeedanceOfficialEntries(
    model,
    displayGroupRatio,
    props.usdExchangeRate
  )

  const primaryEntries: MarketPriceEntry[] = []
  const extraEntries: MarketPriceEntry[] = []
  const officialEntries: MarketPriceEntry[] = []
  let primaryFallback: ReactNode | null = null

  if (seedanceOfficialEntries) {
    primaryEntries.push(...seedanceOfficialEntries.primaryEntries)
    extraEntries.push(...seedanceOfficialEntries.extraEntries)
    if (savingPercent !== null) {
      officialEntries.push(...seedanceOfficialEntries.officialEntries)
    }
  } else if (dynamicSummary?.isSpecialExpression) {
    primaryFallback = t('Special billing expression')
  } else if (dynamicSummary) {
    const visibleDynamicEntries = dynamicSummary.primaryEntries.length
      ? dynamicSummary.primaryEntries
      : dynamicSummary.entries.slice(0, 2)
    const visibleDynamicKeys = new Set(
      visibleDynamicEntries.map((entry) => entry.key)
    )

    for (const entry of visibleDynamicEntries) {
      primaryEntries.push({
        key: entry.key,
        labelKey: entry.shortLabel,
        formatted: entry.formatted,
        unit: 'M',
      })
    }

    for (const entry of dynamicSummary.entries) {
      if (visibleDynamicKeys.has(entry.key)) continue
      extraEntries.push({
        key: entry.key,
        labelKey: entry.shortLabel,
        formatted: entry.formatted,
        unit: 'M',
      })
    }

    if (primaryEntries.length === 0) {
      primaryFallback = t('Dynamic Pricing')
    }

    if (savingPercent !== null) {
      const officialSummary = getDynamicPricingSummary(model, {
        tokenUnit: DEFAULT_TOKEN_UNIT,
        showRechargePrice: false,
        priceRate: props.priceRate,
        usdExchangeRate: props.usdExchangeRate,
        groupRatioMultiplier: 1,
      })
      const officialDynamicEntries = officialSummary?.primaryEntries.length
        ? officialSummary.primaryEntries
        : (officialSummary?.entries.slice(0, 2) ?? [])

      for (const entry of officialDynamicEntries.slice(0, 2)) {
        officialEntries.push({
          key: `official-${entry.key}`,
          labelKey: entry.shortLabel,
          formatted: entry.formatted,
          unit: 'M',
        })
      }
    }
  } else if (isTokenBasedModel(model)) {
    primaryEntries.push(
      {
        key: 'input',
        labelKey: 'Input',
        formatted: formatPrice(
          model,
          'input',
          DEFAULT_TOKEN_UNIT,
          false,
          props.priceRate,
          props.usdExchangeRate
        ),
        unit: 'M',
      },
      {
        key: 'output',
        labelKey: 'Output',
        formatted: formatPrice(
          model,
          'output',
          DEFAULT_TOKEN_UNIT,
          false,
          props.priceRate,
          props.usdExchangeRate
        ),
        unit: 'M',
      }
    )

    if (model.cache_ratio != null) {
      extraEntries.push({
        key: 'cache-read',
        labelKey: 'Cache Read',
        formatted: formatPrice(
          model,
          'cache',
          DEFAULT_TOKEN_UNIT,
          false,
          props.priceRate,
          props.usdExchangeRate
        ),
        unit: 'M',
      })
    }

    if (model.create_cache_ratio != null) {
      extraEntries.push({
        key: 'cache-write',
        labelKey: 'Cache Write',
        formatted: formatPrice(
          model,
          'create_cache',
          DEFAULT_TOKEN_UNIT,
          false,
          props.priceRate,
          props.usdExchangeRate
        ),
        unit: 'M',
      })
    }

    if (model.image_ratio != null) {
      extraEntries.push({
        key: 'image-input',
        labelKey: 'Image In',
        formatted: formatPrice(
          model,
          'image',
          DEFAULT_TOKEN_UNIT,
          false,
          props.priceRate,
          props.usdExchangeRate
        ),
        unit: 'M',
      })
    }

    if (model.audio_ratio != null) {
      extraEntries.push({
        key: 'audio-input',
        labelKey: 'Audio In',
        formatted: formatPrice(
          model,
          'audio_input',
          DEFAULT_TOKEN_UNIT,
          false,
          props.priceRate,
          props.usdExchangeRate
        ),
        unit: 'M',
      })
    }

    if (model.audio_ratio != null && model.audio_completion_ratio != null) {
      extraEntries.push({
        key: 'audio-output',
        labelKey: 'Audio Out',
        formatted: formatPrice(
          model,
          'audio_output',
          DEFAULT_TOKEN_UNIT,
          false,
          props.priceRate,
          props.usdExchangeRate
        ),
        unit: 'M',
      })
    }

    if (savingPercent !== null) {
      officialEntries.push(
        {
          key: 'official-input',
          labelKey: 'Input',
          formatted: formatGroupPrice(
            model,
            '__base',
            'input',
            DEFAULT_TOKEN_UNIT,
            false,
            props.priceRate,
            props.usdExchangeRate,
            { __base: 1 }
          ),
          unit: 'M',
        },
        {
          key: 'official-output',
          labelKey: 'Output',
          formatted: formatGroupPrice(
            model,
            '__base',
            'output',
            DEFAULT_TOKEN_UNIT,
            false,
            props.priceRate,
            props.usdExchangeRate,
            { __base: 1 }
          ),
          unit: 'M',
        }
      )
    }
  } else {
    primaryEntries.push({
      key: 'request',
      labelKey: 'Per Request',
      formatted: formatRequestPrice(
        model,
        false,
        props.priceRate,
        props.usdExchangeRate
      ),
      unit: 'request',
    })

    if (savingPercent !== null) {
      officialEntries.push({
        key: 'official-request',
        labelKey: 'Per Request',
        formatted: formatFixedPrice(
          model,
          '__base',
          false,
          props.priceRate,
          props.usdExchangeRate,
          { __base: 1 }
        ),
        unit: 'request',
      })
    }
  }

  const detailEntries = [...primaryEntries, ...extraEntries]
  const visibleEntries = detailEntries.slice(0, 4)
  const extraMoreCount = Math.max(
    0,
    detailEntries.length - visibleEntries.length
  )
  const officialLine = officialEntries.length > 0 && savingPercent !== null

  const renderUnit = (entry: MarketPriceEntry) => {
    if (entry.unit === 'request') {
      return `/ ${t('request')}`
    }

    return '/ 1M'
  }

  const renderPriceRow = (entry: MarketPriceEntry) => {
    const muted = isMutedPriceEntry(entry)

    return (
      <div
        key={entry.key}
        className={cn(
          'flex min-w-0 items-baseline justify-between gap-2 text-[13px] leading-4',
          muted && 'text-muted-foreground'
        )}
      >
        <span className='text-muted-foreground min-w-0 truncate text-xs leading-4 font-bold'>
          {t(entry.labelKey)}
        </span>
        <span className='shrink-0 text-right whitespace-nowrap'>
          <span
            className={cn(
              'font-mono text-[15px] leading-4 font-extrabold tracking-[-0.03em] tabular-nums',
              muted ? 'text-muted-foreground' : 'text-foreground'
            )}
          >
            {formatMarketPriceValue(entry.formatted)}
          </span>{' '}
          <span className='text-muted-foreground text-[11px] leading-4 font-semibold'>
            {renderUnit(entry)}
          </span>
        </span>
      </div>
    )
  }

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <div className='bg-muted/30 hover:bg-muted/40 h-[124px] overflow-hidden rounded-2xl border p-3 text-left transition-colors' />
        }
      >
        {primaryFallback ? (
          <div className='text-foreground flex min-h-16 items-center text-sm font-semibold'>
            {primaryFallback}
          </div>
        ) : (
          <div className='space-y-0.5'>
            {visibleEntries.map(renderPriceRow)}
          </div>
        )}
        {extraMoreCount > 0 ? (
          <div className='text-muted-foreground mt-1 text-right text-[11px] leading-3 font-medium'>
            {t('+{{count}} more', { count: extraMoreCount })}
          </div>
        ) : null}
      </TooltipTrigger>
      <TooltipContent className='block max-w-sm text-left' side='top'>
        <div className='space-y-2'>
          <div className='font-semibold'>{t('Price')}</div>
          <div className='space-y-1.5'>
            {detailEntries.length > 0 ? (
              detailEntries.map((entry) => (
                <div
                  key={entry.key}
                  className='flex items-center justify-between gap-4'
                >
                  <span>{t(entry.labelKey)}</span>
                  <span className='font-mono'>
                    {entry.formatted} {renderUnit(entry)}
                  </span>
                </div>
              ))
            ) : (
              <div>{primaryFallback}</div>
            )}
          </div>
          {officialLine ? (
            <div className='border-background/25 space-y-1 border-t pt-2'>
              <div className='font-semibold'>{t('Official')}</div>
              {officialEntries.map((entry) => (
                <div
                  key={entry.key}
                  className='flex items-center justify-between gap-4'
                >
                  <span>{t(entry.labelKey)}</span>
                  <span className='font-mono'>
                    {entry.formatted} {renderUnit(entry)}
                  </span>
                </div>
              ))}
            </div>
          ) : null}
        </div>
      </TooltipContent>
    </Tooltip>
  )
}

function MarketModelCard(props: {
  model: MarketModel
  priceRate: number
  usdExchangeRate: number
}) {
  const { t } = useTranslation()
  const model = props.model
  const vendor = model.vendor_name || t('Unknown provider')
  const tags = splitTags(model.tags)
  const endpoints = (model.supported_endpoint_types ?? []).filter(
    (endpoint) => endpoint !== 'openai-video'
  )
  const detailHref = `/model-guide/${toModelGuideSlug(model.model_name)}`
  const modelIconKey = model.icon || model.vendor_icon
  const modelIcon = modelIconKey ? getLobeIcon(modelIconKey, 32) : null
  const initial = model.model_name?.charAt(0).toUpperCase() || '?'
  const contextSize = formatCompactTokenCount(model.context_length)
  const badgeValues = [...new Set([...tags, ...endpoints])]
  const visibleTags = badgeValues.slice(0, 4)
  const hiddenTagCount = Math.max(0, badgeValues.length - visibleTags.length)

  const handleCopy = async (event: MouseEvent<HTMLButtonElement>) => {
    event.preventDefault()
    event.stopPropagation()
    await navigator.clipboard.writeText(model.model_name)
    toast.success(t('Copied'))
  }

  return (
    <article className='group bg-card hover:border-brand/35 border-border/70 flex h-[360px] min-w-0 flex-col rounded-[22px] border p-4 shadow-[0_14px_36px_rgba(15,23,42,0.07)] transition-all duration-200 hover:shadow-[0_18px_44px_rgba(15,23,42,0.1)] dark:shadow-[0_14px_36px_rgba(0,0,0,0.24)] dark:hover:shadow-[0_18px_44px_rgba(0,0,0,0.32)]'>
      <div className='flex min-w-0 items-start gap-3'>
        <div className='text-muted-foreground flex size-10 shrink-0 items-center justify-center rounded-xl border bg-transparent'>
          {modelIcon || <span className='text-base font-bold'>{initial}</span>}
        </div>

        <div className='min-w-0 flex-1'>
          <div className='flex min-w-0 items-start gap-2'>
            <Tooltip>
              <TooltipTrigger render={<div className='min-w-0 flex-1' />}>
                <Link to={detailHref} className='hover:text-brand block'>
                  <h3 className='line-clamp-2 font-mono text-[15px] leading-[1.35] font-bold tracking-[-0.02em] break-all'>
                    {model.model_name}
                  </h3>
                </Link>
              </TooltipTrigger>
              <TooltipContent className='max-w-xs font-mono break-all'>
                {model.model_name}
              </TooltipContent>
            </Tooltip>
            <div className='mt-0.5 flex shrink-0 items-center gap-1.5'>
              <Link
                to={detailHref}
                className='text-muted-foreground hover:text-foreground hover:bg-muted inline-flex items-center gap-1 rounded-md border px-2 py-1 text-xs leading-none font-medium transition-colors'
                aria-label={t('View details')}
              >
                {t('Details')}
                <ChevronRight className='size-3.5' />
              </Link>
              <button
                type='button'
                onClick={handleCopy}
                className='text-muted-foreground hover:text-foreground hover:bg-muted shrink-0 rounded-md border p-1 transition-colors'
                aria-label={t('Copy model name')}
              >
                <Copy className='size-3.5' />
              </button>
            </div>
          </div>

          <div className='text-muted-foreground mt-[7px] flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1 text-xs'>
            <span className='min-w-0 truncate'>{vendor}</span>
            <span aria-hidden='true'>·</span>
            <span>{t(marketKindLabelKey(inferKind(model)))}</span>
            {contextSize ? (
              <>
                <span aria-hidden='true'>·</span>
                <span>{t('{{size}} context', { size: contextSize })}</span>
              </>
            ) : null}
          </div>
        </div>
      </div>

      <p className='text-muted-foreground mt-3 line-clamp-3 min-h-[54px] text-xs leading-[18px]'>
        {model.description || t('No description available.')}
      </p>

      <div className='mt-3 flex max-h-12 min-h-7 flex-wrap gap-1.5 overflow-hidden'>
        {visibleTags.map((tag) => (
          <Badge
            key={tag}
            variant='outline'
            className='bg-muted/30 text-muted-foreground max-w-full truncate rounded-full px-2 py-0.5 text-[11px] font-semibold'
          >
            {t(tag)}
          </Badge>
        ))}
        {hiddenTagCount > 0 ? (
          <Badge variant='secondary' className='rounded-full text-[11px]'>
            {t('+{{count}} more', { count: hiddenTagCount })}
          </Badge>
        ) : null}
      </div>

      <div className='mt-auto shrink-0 pt-3'>
        <MarketPricePanel
          model={model}
          priceRate={props.priceRate}
          usdExchangeRate={props.usdExchangeRate}
        />
      </div>
    </article>
  )
}

function MarketSidebar(props: {
  kindCounts: Map<MarketKind, number>
  vendors: Array<{ name: string; count: number; icon?: string }>
  providerScopeCount: number
  activeVendor: string
  activeKind: MarketKindFilter
  onKindChange: (kind: MarketKindFilter) => void
  onVendorChange: (vendor: string) => void
}) {
  const { t } = useTranslation()
  const allKindCount = [...props.kindCounts.values()].reduce(
    (sum, count) => sum + count,
    0
  )

  return (
    <aside className='bg-card/92 rounded-2xl border p-4 shadow-sm'>
      <div className='space-y-5'>
        <div>
          <div className='text-sm font-semibold'>{t('Model types')}</div>
          <div className='mt-3 grid gap-2'>
            {MODEL_TYPE_FILTERS.map((item) => {
              const Icon = item.icon
              const count =
                item.value === 'all'
                  ? allKindCount
                  : (props.kindCounts.get(item.value) ?? 0)
              return (
                <button
                  key={item.value}
                  type='button'
                  onClick={() => props.onKindChange(item.value)}
                  className={cn(
                    'flex items-center justify-between gap-2 rounded-xl border px-3 py-2 text-left text-xs transition-colors',
                    props.activeKind === item.value
                      ? 'bg-foreground text-background'
                      : 'bg-background text-muted-foreground hover:text-foreground'
                  )}
                >
                  <span className='inline-flex min-w-0 items-center gap-2'>
                    <Icon className='size-3.5 shrink-0' />
                    <span className='truncate'>{t(item.label)}</span>
                  </span>
                  <span className='font-mono'>{count}</span>
                </button>
              )
            })}
          </div>
        </div>

        <div>
          <div className='text-sm font-semibold'>{t('Providers')}</div>
          <div className='mt-3 grid gap-2'>
            <button
              type='button'
              onClick={() => props.onVendorChange('all')}
              className={cn(
                'flex items-center justify-between gap-2 rounded-xl border px-3 py-2 text-left text-xs transition-colors',
                props.activeVendor === 'all'
                  ? 'bg-foreground text-background'
                  : 'bg-background text-muted-foreground hover:text-foreground'
              )}
            >
              <span>{t('All')}</span>
              <span className='font-mono'>{props.providerScopeCount}</span>
            </button>
            {props.vendors.map((vendor) => (
              <button
                key={vendor.name}
                type='button'
                onClick={() => props.onVendorChange(vendor.name)}
                className={cn(
                  'flex items-center justify-between gap-2 rounded-xl border px-3 py-2 text-left text-xs transition-colors',
                  props.activeVendor === vendor.name
                    ? 'bg-foreground text-background'
                    : 'bg-background text-muted-foreground hover:text-foreground'
                )}
              >
                <span className='inline-flex min-w-0 items-center gap-2'>
                  {vendor.icon ? (
                    <span className='flex size-4 shrink-0 items-center justify-center'>
                      {getLobeIcon(vendor.icon, 16)}
                    </span>
                  ) : null}
                  <span className='truncate'>{vendor.name}</span>
                </span>
                <span className='font-mono'>{vendor.count}</span>
              </button>
            ))}
          </div>
        </div>
      </div>
    </aside>
  )
}

export function Market() {
  const { t } = useTranslation()
  const [query, setQuery] = useState('')
  const deferredQuery = useDeferredValue(query)
  const [activeKind, setActiveKind] = useState<MarketKindFilter>('all')
  const [activeVendor, setActiveVendor] = useState('all')
  const [sortBy, setSortBy] = useState<MarketSortOption>('recommended')
  const [page, setPage] = useState(1)
  const pricing = usePricingData()

  const models = useMemo<IndexedMarketModel[]>(() => {
    const source = pricing.models.length > 0 ? pricing.models : FALLBACK_MODELS

    return source
      .map((model) => {
        const marketKind = inferKind(model)
        return {
          ...model,
          marketKind,
          searchText: modelSignals(model).join(' ').toLowerCase(),
        }
      })
      .sort(compareMarketModels)
  }, [pricing.models])

  const marketSummary = useMemo(() => {
    const vendorInfo = new Map<string, { count: number; icon?: string }>()
    const kindCounts = new Map<MarketKind, number>(
      MODEL_TYPES.map((item) => [item.value, 0])
    )
    let kindCount = 0
    let providerScopeCount = 0

    for (const model of models) {
      kindCounts.set(
        model.marketKind,
        (kindCounts.get(model.marketKind) ?? 0) + 1
      )

      if (activeKind !== 'all' && model.marketKind !== activeKind) {
        continue
      }

      kindCount += 1
      providerScopeCount += 1
      if (model.vendor_name) {
        const existing = vendorInfo.get(model.vendor_name)
        vendorInfo.set(model.vendor_name, {
          count: (existing?.count ?? 0) + 1,
          icon: existing?.icon || model.vendor_icon || model.icon,
        })
      }
    }

    return {
      vendors: [...vendorInfo.entries()]
        .map(([name, info]) => ({
          name,
          count: info.count,
          icon: info.icon,
        }))
        .sort((left, right) => {
          const countCompare = right.count - left.count
          if (countCompare !== 0) return countCompare
          return MODEL_SORT_COLLATOR.compare(left.name, right.name)
        }),
      kindCounts,
      kindCount,
      providerScopeCount,
    }
  }, [activeKind, models])

  const filteredModels = useMemo(() => {
    const normalizedQuery = deferredQuery.trim().toLowerCase()
    const result = models.filter((model) => {
      if (activeKind !== 'all' && model.marketKind !== activeKind) return false
      if (activeVendor !== 'all' && model.vendor_name !== activeVendor) {
        return false
      }
      if (!normalizedQuery) return true
      return model.searchText.includes(normalizedQuery)
    })

    return sortMarketModels(result, sortBy)
  }, [activeKind, activeVendor, deferredQuery, models, sortBy])

  const totalPages = Math.max(
    1,
    Math.ceil(filteredModels.length / MARKET_PAGE_SIZE)
  )
  const currentPage = Math.min(page, totalPages)

  const pagedModels = useMemo(() => {
    const start = (currentPage - 1) * MARKET_PAGE_SIZE
    return filteredModels.slice(start, start + MARKET_PAGE_SIZE)
  }, [currentPage, filteredModels])

  const displayStart =
    filteredModels.length === 0 ? 0 : (currentPage - 1) * MARKET_PAGE_SIZE + 1
  const displayEnd = Math.min(
    currentPage * MARKET_PAGE_SIZE,
    filteredModels.length
  )

  useEffect(() => {
    setPage(1)
  }, [activeKind, activeVendor, deferredQuery, sortBy])

  useEffect(() => {
    if (activeVendor === 'all') return
    const vendorStillVisible = marketSummary.vendors.some(
      (vendor) => vendor.name === activeVendor
    )
    if (!vendorStillVisible) {
      setActiveVendor('all')
    }
  }, [activeVendor, marketSummary.vendors])

  return (
    <PublicLayout showMainContainer={false} showNotifications={false}>
      <main className='mx-auto max-w-[1540px] px-3 pt-24 pb-12 sm:px-4 md:px-6'>
        <section className='bg-card/92 rounded-2xl border px-4 py-8 text-center shadow-sm sm:rounded-3xl sm:px-6 md:px-10'>
          <h1 className='mx-auto max-w-4xl text-3xl leading-tight font-bold tracking-tight sm:text-4xl md:text-5xl'>
            {t('Find the right AI model, faster')}
          </h1>
          <p className='text-muted-foreground mx-auto mt-3 max-w-3xl text-sm leading-7 md:text-base'>
            {t(
              'Compare providers, modalities, pricing signals, and endpoint compatibility in one scan, then open a model guide for implementation details.'
            )}
          </p>
        </section>

        <section className='mt-5 grid gap-5 lg:grid-cols-[260px_minmax(0,1fr)] xl:grid-cols-[280px_minmax(0,1fr)]'>
          <MarketSidebar
            kindCounts={marketSummary.kindCounts}
            vendors={marketSummary.vendors}
            providerScopeCount={marketSummary.providerScopeCount}
            activeVendor={activeVendor}
            activeKind={activeKind}
            onKindChange={setActiveKind}
            onVendorChange={setActiveVendor}
          />

          <div className='bg-card/92 rounded-2xl border p-3 shadow-sm sm:rounded-3xl sm:p-4 md:p-5'>
            <div className='bg-background/65 space-y-4 rounded-2xl border p-3 sm:p-4'>
              <div className='flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between'>
                <h2 className='text-lg font-bold'>{t('Available models')}</h2>
                <div className='flex items-center gap-2'>
                  <CurrencyToggle />
                  <Select
                    value={sortBy}
                    onValueChange={(value) => {
                      if (value !== null) {
                        setSortBy(value as MarketSortOption)
                      }
                    }}
                  >
                    <SelectTrigger
                      className='bg-card h-9 w-full rounded-full sm:w-[190px]'
                      aria-label={t('Sort')}
                    >
                      <SelectValue>
                        {t(
                          MARKET_SORT_OPTIONS.find(
                            (option) => option.value === sortBy
                          )?.label ?? 'Recommended'
                        )}
                      </SelectValue>
                    </SelectTrigger>
                    <SelectContent alignItemWithTrigger={false}>
                      {MARKET_SORT_OPTIONS.map((option) => (
                        <SelectItem key={option.value} value={option.value}>
                          {t(option.label)}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              </div>

              <div className='relative'>
                <div className='relative'>
                  <Search className='text-muted-foreground pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2' />
                  <Input
                    value={query}
                    onChange={(event) => setQuery(event.target.value)}
                    placeholder={t('Search model, task, or provider')}
                    className='bg-card h-11 rounded-full pr-4 pl-9'
                  />
                </div>
              </div>
            </div>

            <div className='mt-4 grid auto-rows-fr gap-4 xl:grid-cols-2 2xl:grid-cols-3'>
              {pagedModels.map((model) => (
                <MarketModelCard
                  key={model.model_name}
                  model={model}
                  priceRate={pricing.priceRate}
                  usdExchangeRate={pricing.usdExchangeRate}
                />
              ))}
            </div>

            {filteredModels.length === 0 && (
              <div className='text-muted-foreground flex min-h-48 items-center justify-center text-sm'>
                {t('No models match your current filters.')}
              </div>
            )}

            {filteredModels.length > 0 && (
              <div className='mt-5 flex flex-col items-center justify-between gap-3 border-t pt-4 text-sm sm:flex-row'>
                <p className='text-muted-foreground'>
                  {t('Showing {{start}}-{{end}} of {{total}} models', {
                    start: displayStart,
                    end: displayEnd,
                    total: filteredModels.length,
                  })}
                </p>
                {totalPages > 1 && (
                  <div className='flex items-center gap-2'>
                    <Button
                      type='button'
                      variant='outline'
                      size='sm'
                      onClick={() =>
                        setPage((current) => Math.max(1, current - 1))
                      }
                      disabled={currentPage <= 1}
                    >
                      <ChevronLeft className='size-4' />
                      {t('Previous page')}
                    </Button>
                    <span className='text-muted-foreground px-1 text-xs'>
                      {t('Page {{current}} of {{total}}', {
                        current: currentPage,
                        total: totalPages,
                      })}
                    </span>
                    <Button
                      type='button'
                      variant='outline'
                      size='sm'
                      onClick={() =>
                        setPage((current) => Math.min(totalPages, current + 1))
                      }
                      disabled={currentPage >= totalPages}
                    >
                      {t('Next page')}
                      <ChevronRight className='size-4' />
                    </Button>
                  </div>
                )}
              </div>
            )}
          </div>
        </section>
      </main>
    </PublicLayout>
  )
}
