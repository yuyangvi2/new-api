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
  Brain,
  ChevronLeft,
  ChevronRight,
  Code2,
  Copy,
  FileText,
  ImageIcon,
  Layers3,
  MessageSquareText,
  Music2,
  Search,
  Sparkles,
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

const MARKET_PAGE_SIZE = 18

type MarketSortOption = 'recommended' | 'name' | 'price-low' | 'context-high'

const MARKET_SORT_OPTIONS: Array<{ value: MarketSortOption; label: string }> = [
  { value: 'recommended', label: 'Recommended' },
  { value: 'name', label: 'Name' },
  { value: 'price-low', label: 'Price: Low to High' },
  { value: 'context-high', label: 'Context' },
]

type MarketTask = {
  value: string
  label: string
  kinds: MarketKind[]
  icon: ComponentType<{ className?: string }>
}

const TASKS: MarketTask[] = [
  {
    value: 'chat',
    label: 'Chat',
    kinds: ['text'],
    icon: MessageSquareText,
  },
  {
    value: 'reasoning',
    label: 'Reasoning',
    kinds: ['text'],
    icon: Brain,
  },
  {
    value: 'coding',
    label: 'Coding',
    kinds: ['text'],
    icon: Code2,
  },
  {
    value: 'long-context',
    label: 'Long context',
    kinds: ['text'],
    icon: FileText,
  },
  {
    value: 'vision',
    label: 'Vision',
    kinds: ['text'],
    icon: ImageIcon,
  },
  {
    value: 'text-to-image',
    label: 'Text to Image',
    kinds: ['image'],
    icon: Sparkles,
  },
  {
    value: 'image-edit',
    label: 'Image edit',
    kinds: ['image'],
    icon: ImageIcon,
  },
  {
    value: 'text-to-video',
    label: 'Text to Video',
    kinds: ['video'],
    icon: Video,
  },
  {
    value: 'image-to-video',
    label: 'Image to Video',
    kinds: ['video'],
    icon: ImageIcon,
  },
  {
    value: 'text-to-speech',
    label: 'Text to Speech',
    kinds: ['audio'],
    icon: Music2,
  },
  {
    value: 'speech-to-text',
    label: 'Speech to Text',
    kinds: ['audio'],
    icon: Music2,
  },
]
const CHAT_ENDPOINTS = new Set([
  'openai',
  'openai-response',
  'openai-response-compact',
  'anthropic',
  'gemini',
])

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

function hasSignal(model: IndexedMarketModel, signals: string[]): boolean {
  return signals.some((signal) => model.searchText.includes(signal))
}

function modelHasEndpoint(
  model: IndexedMarketModel,
  endpoint: string
): boolean {
  return (model.supported_endpoint_types ?? []).some(
    (item) => item.toLowerCase() === endpoint
  )
}

function modelHasModality(
  model: IndexedMarketModel,
  direction: 'input' | 'output',
  modality: string
): boolean {
  const values =
    direction === 'input'
      ? (model.input_modalities ?? [])
      : (model.output_modalities ?? [])
  return values.some((item) => item.toLowerCase() === modality)
}

function modelIsVision(model: IndexedMarketModel): boolean {
  return (
    modelHasModality(model, 'input', 'image') ||
    modelHasModality(model, 'input', 'video') ||
    hasSignal(model, ['vision', 'visual', 'multimodal', 'multi-modal', '-vl'])
  )
}

function modelIsReasoning(model: IndexedMarketModel): boolean {
  return (
    (model.capabilities ?? []).includes('reasoning') ||
    hasSignal(model, [
      'reasoning',
      'reasoner',
      'thinking',
      'think',
      'qwq',
      'qvq',
      'r1',
      'o1',
      'o3',
      'o4',
    ])
  )
}

function modelIsCoding(model: IndexedMarketModel): boolean {
  return hasSignal(model, [
    'code',
    'coder',
    'coding',
    'codex',
    'codestral',
    'devstral',
    'programming',
    'software',
  ])
}

function modelIsLongContext(model: IndexedMarketModel): boolean {
  const contextLength = Number(model.context_length ?? 0)
  return (
    contextLength >= 128000 ||
    hasSignal(model, [
      'long context',
      'long-context',
      'long',
      '128k',
      '200k',
      '256k',
      '1m',
    ])
  )
}

function modelSupportsChat(model: IndexedMarketModel): boolean {
  if (model.marketKind !== 'text') return false
  return (model.supported_endpoint_types ?? []).some((endpoint) =>
    CHAT_ENDPOINTS.has(endpoint.toLowerCase())
  )
}

function modelMatchesTask(model: IndexedMarketModel, task: string): boolean {
  if (task === 'all') return true

  if (task === 'chat') {
    return (
      modelSupportsChat(model) &&
      !modelIsReasoning(model) &&
      !modelIsCoding(model) &&
      !modelIsLongContext(model) &&
      !modelIsVision(model)
    )
  }

  if (task === 'reasoning') {
    return model.marketKind === 'text' && modelIsReasoning(model)
  }
  if (task === 'coding') {
    return model.marketKind === 'text' && modelIsCoding(model)
  }
  if (task === 'long-context') {
    return model.marketKind === 'text' && modelIsLongContext(model)
  }
  if (task === 'vision') {
    return model.marketKind === 'text' && modelIsVision(model)
  }

  if (task === 'text-to-image') {
    return (
      modelHasEndpoint(model, 'image-generation') ||
      (modelHasModality(model, 'input', 'text') &&
        modelHasModality(model, 'output', 'image')) ||
      model.searchText.includes('text to image') ||
      model.searchText.includes('text-to-image')
    )
  }

  if (task === 'image-edit') {
    return (
      model.marketKind === 'image' &&
      (model.searchText.includes('image edit') ||
        model.searchText.includes('image-edit') ||
        model.searchText.includes('edit image') ||
        model.searchText.includes('image variation'))
    )
  }

  if (task === 'text-to-video') {
    return (
      model.marketKind === 'video' &&
      (modelHasEndpoint(model, 'openai-video') ||
        modelHasModality(model, 'output', 'video') ||
        model.searchText.includes('text to video') ||
        model.searchText.includes('text-to-video') ||
        model.searchText.includes('t2v'))
    )
  }

  if (task === 'image-to-video') {
    return (
      model.marketKind === 'video' &&
      (modelHasModality(model, 'input', 'image') ||
        model.searchText.includes('image to video') ||
        model.searchText.includes('image-to-video') ||
        model.searchText.includes('i2v') ||
        model.searchText.includes('vision'))
    )
  }

  if (task === 'text-to-speech') {
    return (
      model.marketKind === 'audio' &&
      (model.searchText.includes('tts') ||
        model.searchText.includes('text to speech') ||
        model.searchText.includes('speech synthesis') ||
        model.searchText.includes('voice'))
    )
  }

  if (task === 'speech-to-text') {
    return (
      model.marketKind === 'audio' &&
      (model.searchText.includes('stt') ||
        model.searchText.includes('speech to text') ||
        model.searchText.includes('asr') ||
        model.searchText.includes('transcription') ||
        model.searchText.includes('whisper'))
    )
  }

  return false
}

type MarketPriceEntry = {
  key: string
  labelKey: string
  formatted: string
  unit: 'M' | 'request'
  direction?: 'in' | 'out'
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

  const primaryEntries: MarketPriceEntry[] = []
  const extraEntries: MarketPriceEntry[] = []
  const officialEntries: MarketPriceEntry[] = []
  let primaryFallback: ReactNode | null = null
  let secondaryFallback: ReactNode | null = null

  if (dynamicSummary?.isSpecialExpression) {
    primaryFallback = t('Special billing expression')
    secondaryFallback = t('Dynamic Pricing')
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
        direction:
          entry.field === 'inputPrice'
            ? 'in'
            : entry.field === 'outputPrice'
              ? 'out'
              : undefined,
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
        direction: 'in',
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
        direction: 'out',
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
    secondaryFallback = t('per request')

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

  const firstExtraEntry = extraEntries[0]
  const secondExtraEntry = extraEntries[1]
  const visibleExtraEntry =
    officialEntries.length > 0 ? firstExtraEntry : secondExtraEntry
  const inlineExtraEntry = officialEntries.length > 0 ? null : firstExtraEntry
  const consumedExtraCount = [visibleExtraEntry, inlineExtraEntry].filter(
    Boolean
  ).length
  const extraMoreCount = Math.max(0, extraEntries.length - consumedExtraCount)
  const detailEntries = [...primaryEntries, ...extraEntries]

  const renderUnit = (entry: MarketPriceEntry) => {
    if (entry.unit === 'request') {
      return `/ ${t('request')}`
    }

    if (entry.direction) {
      return `/M ${entry.direction}`
    }

    return '/M'
  }

  const renderPriceEntry = (entry: MarketPriceEntry) => (
    <span key={entry.key} className='min-w-0 whitespace-nowrap'>
      <span className='text-foreground font-mono font-semibold'>
        {entry.formatted}
      </span>{' '}
      <span className='text-muted-foreground font-medium'>
        {renderUnit(entry)}
      </span>
    </span>
  )

  const primaryLine = primaryFallback ?? (
    <div className='flex min-w-0 items-baseline gap-1.5 overflow-hidden text-sm leading-5'>
      {primaryEntries.slice(0, 2).map((entry, index) => (
        <span
          key={entry.key}
          className='inline-flex min-w-0 items-baseline gap-1.5'
        >
          {index > 0 && <span className='text-muted-foreground'>·</span>}
          {renderPriceEntry(entry)}
        </span>
      ))}
    </div>
  )

  const officialLine = officialEntries.length > 0 && savingPercent !== null
  const extraLineEntry = officialLine ? firstExtraEntry : inlineExtraEntry
  const bottomExtraEntry = visibleExtraEntry

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <div className='bg-muted/30 hover:bg-muted/45 min-h-[104px] rounded-2xl border p-3 text-left transition-colors' />
        }
      >
        <div className='min-h-5 min-w-0 overflow-hidden'>{primaryLine}</div>

        <div className='mt-2 flex min-h-5 items-center justify-between gap-2 text-xs'>
          {officialLine ? (
            <>
              <span className='text-muted-foreground min-w-0 truncate'>
                {t('Official')}{' '}
                {officialEntries.map((entry) => entry.formatted).join(' / ')}
              </span>
              <span className='shrink-0 rounded-full border border-emerald-200 bg-emerald-50 px-2 py-0.5 font-semibold text-emerald-700 dark:border-emerald-900/60 dark:bg-emerald-950/35 dark:text-emerald-300'>
                {t('Save {{percent}}%', { percent: savingPercent })}
              </span>
            </>
          ) : extraLineEntry ? (
            <div className='text-muted-foreground min-w-0 truncate'>
              {t(extraLineEntry.labelKey)} {renderPriceEntry(extraLineEntry)}
            </div>
          ) : secondaryFallback ? (
            <span className='text-muted-foreground truncate'>
              {secondaryFallback}
            </span>
          ) : null}
        </div>

        <div className='text-muted-foreground mt-1.5 flex min-h-5 items-center gap-2 text-xs'>
          {bottomExtraEntry ? (
            <span className='min-w-0 truncate'>
              {t(bottomExtraEntry.labelKey)} {renderPriceEntry(bottomExtraEntry)}
            </span>
          ) : null}
          {extraMoreCount > 0 ? (
            <span className='shrink-0 rounded-full border px-1.5 py-0.5 text-[10px] font-medium'>
              {t('+{{count}} more', { count: extraMoreCount })}
            </span>
          ) : null}
        </div>
      </TooltipTrigger>
      <TooltipContent className='block max-w-sm text-left' side='top'>
        <div className='space-y-2'>
          <div className='font-semibold'>{t('Price')}</div>
          <div className='space-y-1'>
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
  const endpoints = model.supported_endpoint_types ?? []
  const groups = model.enable_groups ?? []
  const detailHref = `/model-guide/${toModelGuideSlug(model.model_name)}`
  const modelIconKey = model.icon || model.vendor_icon
  const modelIcon = modelIconKey ? getLobeIcon(modelIconKey, 28) : null
  const initial = model.model_name?.charAt(0).toUpperCase() || '?'
  const contextSize = formatCompactTokenCount(model.context_length)
  const badgeValues = [...new Set([...tags, ...endpoints])]
  const visibleTags = badgeValues.slice(0, 3)
  const hiddenTagCount = Math.max(0, badgeValues.length - visibleTags.length)

  const handleCopy = async (event: MouseEvent<HTMLButtonElement>) => {
    event.preventDefault()
    event.stopPropagation()
    await navigator.clipboard.writeText(model.model_name)
    toast.success(t('Copied'))
  }

  return (
    <article className='group bg-card flex h-full min-h-[342px] flex-col rounded-2xl border p-4 shadow-sm transition-all duration-200 hover:-translate-y-0.5 hover:border-brand/40 hover:shadow-md'>
      <div className='flex items-start justify-between gap-3'>
        <div className='flex min-w-0 items-start gap-3'>
          <div
            className={cn(
              'flex size-12 shrink-0 items-center justify-center rounded-2xl bg-gradient-to-br text-white shadow-sm ring-1 ring-border/60',
              model.coverTone || 'from-slate-900 via-slate-700 to-orange-300'
            )}
          >
            {modelIcon || <span className='text-lg font-black'>{initial}</span>}
          </div>
          <div className='min-w-0'>
            <div className='flex min-w-0 items-center gap-1.5'>
              <h3 className='min-w-0 truncate font-mono text-base font-bold'>
                <Link to={detailHref} className='hover:text-brand'>
                  {model.model_name}
                </Link>
              </h3>
              <button
                type='button'
                onClick={handleCopy}
                className='text-muted-foreground hover:text-foreground hover:bg-muted shrink-0 rounded-md border p-1 transition-colors'
                aria-label={t('Copy model name')}
              >
                <Copy className='size-3.5' />
              </button>
            </div>
            <p className='text-muted-foreground mt-1 truncate text-xs'>
              {vendor} · {t(marketKindLabelKey(inferKind(model)))}
            </p>
          </div>
        </div>
        {contextSize ? (
          <Badge
            variant='outline'
            className='shrink-0 border-blue-200 bg-blue-50 px-2 py-1 text-[11px] font-semibold text-blue-700 dark:border-blue-900/60 dark:bg-blue-950/35 dark:text-blue-300'
          >
            {t('{{size}} context', { size: contextSize })}
          </Badge>
        ) : null}
      </div>

      <div className='mt-4'>
        <MarketPricePanel
          model={model}
          priceRate={props.priceRate}
          usdExchangeRate={props.usdExchangeRate}
        />
      </div>

      <div className='mt-3 flex min-h-8 flex-wrap gap-2 overflow-hidden'>
        {visibleTags.map((tag) => (
          <Badge
            key={tag}
            variant='outline'
            className='bg-background/70 text-foreground/80 max-w-full truncate text-[11px]'
          >
            {t(tag)}
          </Badge>
        ))}
        {hiddenTagCount > 0 ? (
          <Badge variant='secondary' className='text-[11px]'>
            {t('+{{count}} more', { count: hiddenTagCount })}
          </Badge>
        ) : null}
      </div>

      <p className='text-muted-foreground mt-3 line-clamp-2 min-h-10 text-xs leading-5'>
        {model.description || t('No description available.')}
      </p>

      <div className='mt-auto flex items-center justify-between gap-3 pt-4'>
        <div className='text-muted-foreground min-w-0 truncate text-xs'>
          {groups.length > 0
            ? groups.slice(0, 2).join(', ')
            : t('Default group')}
        </div>
        <Link
          to={detailHref}
          className='text-muted-foreground group-hover:text-brand shrink-0 text-xs font-semibold transition-colors'
        >
          {t('View model guide')} →
        </Link>
      </div>
    </article>
  )
}

function MarketSidebar(props: {
  kindCounts: Map<MarketKind, number>
  vendors: Array<{ name: string; count: number; icon?: string }>
  tasks: Array<MarketTask & { count: number }>
  kindLabel: string
  kindCount: number
  providerScopeCount: number
  activeTask: string
  activeVendor: string
  activeKind: MarketKind
  onKindChange: (kind: MarketKind) => void
  onTaskChange: (task: string) => void
  onVendorChange: (vendor: string) => void
}) {
  const { t } = useTranslation()

  return (
    <aside className='bg-card/92 rounded-2xl border p-4 shadow-sm'>
      <div className='space-y-5'>
        <div>
          <div className='text-sm font-semibold'>{t('Model types')}</div>
          <div className='mt-3 grid gap-2'>
            {MODEL_TYPES.map((item) => {
              const Icon = item.icon
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
                  <span className='font-mono'>
                    {props.kindCounts.get(item.value) ?? 0}
                  </span>
                </button>
              )
            })}
          </div>
        </div>

        <div>
          <div className='flex items-center justify-between gap-2'>
            <div className='text-sm font-semibold'>{t('Tasks')}</div>
            <Badge variant='outline' className='text-[11px]'>
              {t(props.kindLabel)} · {props.kindCount}
            </Badge>
          </div>
          <div className='mt-3 grid gap-2'>
            <button
              type='button'
              onClick={() => props.onTaskChange('all')}
              className={cn(
                'flex items-center justify-between gap-2 rounded-xl border px-3 py-2 text-left text-xs transition-colors',
                props.activeTask === 'all'
                  ? 'bg-foreground text-background'
                  : 'bg-background text-muted-foreground hover:text-foreground'
              )}
            >
              <span className='inline-flex min-w-0 items-center gap-2'>
                <Layers3 className='size-3.5 shrink-0' />
                <span className='truncate'>{t('All tasks')}</span>
              </span>
              <span className='font-mono'>{props.kindCount}</span>
            </button>
            {props.tasks.map((task) => {
              const TaskIcon = task.icon
              return (
                <button
                  key={task.value}
                  type='button'
                  onClick={() => props.onTaskChange(task.value)}
                  className={cn(
                    'flex items-center justify-between gap-2 rounded-xl border px-3 py-2 text-left text-xs transition-colors',
                    props.activeTask === task.value
                      ? 'bg-foreground text-background'
                      : 'bg-background text-muted-foreground hover:text-foreground'
                  )}
                >
                  <span className='inline-flex min-w-0 items-center gap-2'>
                    <TaskIcon className='size-3.5 shrink-0' />
                    <span className='truncate'>{t(task.label)}</span>
                  </span>
                  <span className='font-mono'>{task.count}</span>
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
  const [activeKind, setActiveKind] = useState<MarketKind>('text')
  const [activeTask, setActiveTask] = useState('all')
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
    const activeTasks = TASKS.filter((task) => task.kinds.includes(activeKind))
    const taskCounts = new Map<string, number>(
      activeTasks.map((task) => [task.value, 0])
    )
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

      if (model.marketKind !== activeKind) {
        continue
      }

      kindCount += 1
      for (const task of activeTasks) {
        if (modelMatchesTask(model, task.value)) {
          taskCounts.set(task.value, (taskCounts.get(task.value) ?? 0) + 1)
        }
      }

      if (!modelMatchesTask(model, activeTask)) {
        continue
      }

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
      tasks: activeTasks.map((task) => ({
        ...task,
        count: taskCounts.get(task.value) ?? 0,
      })),
      kindCounts,
      kindCount,
      providerScopeCount,
    }
  }, [activeKind, activeTask, models])

  const filteredModels = useMemo(() => {
    const normalizedQuery = deferredQuery.trim().toLowerCase()
    const result = models.filter((model) => {
      if (model.marketKind !== activeKind) return false
      if (!modelMatchesTask(model, activeTask)) return false
      if (activeVendor !== 'all' && model.vendor_name !== activeVendor) {
        return false
      }
      if (!normalizedQuery) return true
      return model.searchText.includes(normalizedQuery)
    })

    return sortMarketModels(result, sortBy)
  }, [activeKind, activeTask, activeVendor, deferredQuery, models, sortBy])

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
  }, [activeKind, activeTask, activeVendor, deferredQuery, sortBy])

  useEffect(() => {
    if (activeTask === 'all') return
    const taskStillVisible = marketSummary.tasks.some(
      (task) => task.value === activeTask
    )
    if (!taskStillVisible) {
      setActiveTask('all')
    }
  }, [activeTask, marketSummary.tasks])

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
      <main className='mx-auto max-w-7xl px-3 pt-24 pb-12 sm:px-4 md:px-6'>
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

        <section className='mt-5 grid gap-5 lg:grid-cols-[240px_minmax(0,1fr)]'>
          <MarketSidebar
            kindCounts={marketSummary.kindCounts}
            vendors={marketSummary.vendors}
            tasks={marketSummary.tasks}
            kindLabel={
              MODEL_TYPES.find((item) => item.value === activeKind)?.label ??
              'Text'
            }
            kindCount={marketSummary.kindCount}
            providerScopeCount={marketSummary.providerScopeCount}
            activeTask={activeTask}
            activeVendor={activeVendor}
            activeKind={activeKind}
            onKindChange={(kind) => {
              setActiveKind(kind)
              setActiveTask('all')
            }}
            onTaskChange={setActiveTask}
            onVendorChange={setActiveVendor}
          />

          <div className='bg-card/92 rounded-2xl border p-3 shadow-sm sm:rounded-3xl sm:p-4 md:p-5'>
            <div className='bg-background/65 space-y-4 rounded-2xl border p-3 sm:p-4'>
              <div className='flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between'>
                <h2 className='text-lg font-bold'>{t('Available models')}</h2>
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

            <div className='mt-4 grid gap-5 md:grid-cols-2 xl:grid-cols-3'>
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
