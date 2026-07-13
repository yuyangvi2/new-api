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
import { Copy, FileText, ImageIcon, Music2, Search, Video } from 'lucide-react'
import {
  useDeferredValue,
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
import { Input } from '@/components/ui/input'
import { getLobeIcon } from '@/lib/lobe-icon'
import { cn } from '@/lib/utils'

import { DEFAULT_TOKEN_UNIT } from '../pricing/constants'
import {
  getDynamicDisplayGroupRatio,
  getDynamicPricingSummary,
} from '../pricing/lib/dynamic-price'
import { usePricingData } from '../pricing/hooks/use-pricing-data'
import { isTokenBasedModel } from '../pricing/lib/model-helpers'
import { formatPrice, formatRequestPrice } from '../pricing/lib/price'
import {
  FALLBACK_MODELS,
  inferKind,
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

const TASKS = [
  'Chat',
  'Text to Image',
  'Image to Video',
  'Image edit',
  'Audio',
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

function modelMatchesTask(model: IndexedMarketModel, task: string): boolean {
  if (task === 'all') return true

  const normalizedTask = task.toLowerCase()
  const endpoints = model.supported_endpoint_types ?? []
  const endpointSet = new Set(
    endpoints.map((endpoint) => endpoint.toLowerCase())
  )
  const inputModalities = model.input_modalities ?? []
  const outputModalities = model.output_modalities ?? []

  if (task === 'Chat') {
    if (
      endpoints.some((endpoint) =>
        CHAT_ENDPOINTS.has(endpoint.toLowerCase())
      )
    ) {
      return model.marketKind === 'text' || !endpointSet.has('image-generation')
    }
    return model.marketKind === 'text' || model.searchText.includes('chat')
  }

  if (task === 'Text to Image') {
    return (
      endpointSet.has('image-generation') ||
      (inputModalities.includes('text') &&
        outputModalities.includes('image')) ||
      model.searchText.includes('text to image') ||
      model.searchText.includes('text-to-image')
    )
  }

  if (task === 'Image to Video') {
    return (
      (inputModalities.includes('image') &&
        outputModalities.includes('video')) ||
      model.searchText.includes('image to video') ||
      model.searchText.includes('image-to-video') ||
      model.searchText.includes('i2v')
    )
  }

  if (task === 'Image edit') {
    return (
      model.searchText.includes('image edit') ||
      model.searchText.includes('image-edit') ||
      model.searchText.includes('edit image') ||
      model.searchText.includes('image variation')
    )
  }

  if (model.marketKind === normalizedTask) return true

  return model.searchText.includes(normalizedTask)
}

function MarketModelCard(props: {
  model: MarketModel
  priceRate: number
  usdExchangeRate: number
}) {
  const { t } = useTranslation()
  const model = props.model
  const vendor = model.vendor_name || t('Unknown provider')
  const tags = splitTags(model.tags).slice(0, 4)
  const endpoints = (model.supported_endpoint_types ?? []).slice(0, 2)
  const groups = (model.enable_groups ?? []).slice(0, 2)
  const detailHref = `/model-guide/${toModelGuideSlug(model.model_name)}`
  const modelIconKey = model.icon || model.vendor_icon
  const modelIcon = modelIconKey ? getLobeIcon(modelIconKey, 28) : null
  const initial = model.model_name?.charAt(0).toUpperCase() || '?'
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

  let priceSummary: ReactNode
  if (dynamicSummary?.isSpecialExpression) {
    priceSummary = t('Special billing expression')
  } else if (dynamicSummary && dynamicSummary.primaryEntries.length > 0) {
    priceSummary = dynamicSummary.primaryEntries.map((entry) => (
      <span key={entry.key} className='whitespace-nowrap'>
        {t(entry.shortLabel)}{' '}
        <span className='font-mono font-semibold'>{entry.formatted}</span>/1M
      </span>
    ))
  } else if (dynamicSummary) {
    priceSummary = t('Dynamic Pricing')
  } else if (isTokenBasedModel(model)) {
    priceSummary = (
      <>
        <span className='whitespace-nowrap'>
          {t('Input')}{' '}
          <span className='font-mono font-semibold'>
            {formatPrice(
              model,
              'input',
              DEFAULT_TOKEN_UNIT,
              false,
              props.priceRate,
              props.usdExchangeRate
            )}
          </span>
          /1M
        </span>
        <span className='whitespace-nowrap'>
          {t('Output')}{' '}
          <span className='font-mono font-semibold'>
            {formatPrice(
              model,
              'output',
              DEFAULT_TOKEN_UNIT,
              false,
              props.priceRate,
              props.usdExchangeRate
            )}
          </span>
          /1M
        </span>
      </>
    )
  } else {
    priceSummary = (
      <span className='whitespace-nowrap'>
        <span className='font-mono font-semibold'>
          {formatRequestPrice(
            model,
            false,
            props.priceRate,
            props.usdExchangeRate
          )}
        </span>{' '}
        / {t('request')}
      </span>
    )
  }

  const handleCopy = async (event: MouseEvent<HTMLButtonElement>) => {
    event.stopPropagation()
    await navigator.clipboard.writeText(model.model_name)
    toast.success(t('Copied'))
  }

  return (
    <article className='group bg-card overflow-hidden rounded-2xl border shadow-sm transition-all duration-200 hover:-translate-y-0.5 hover:shadow-md'>
      <Link
        to={detailHref}
        className='focus-visible:ring-ring block focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:outline-none'
        aria-label={t('Open {{model}} model guide', {
          model: model.model_name,
        })}
      >
        <div
          className={cn(
            'relative flex aspect-[1.75] items-center justify-center overflow-hidden bg-gradient-to-br',
            model.coverTone || 'from-slate-900 via-slate-700 to-orange-300'
          )}
        >
          <div className='absolute inset-0 bg-[linear-gradient(120deg,transparent_0%,rgb(255_255_255/0.22)_48%,transparent_100%)] opacity-60' />
          <div className='relative flex flex-col items-center gap-3 px-5 text-center text-white drop-shadow-sm sm:px-6'>
            <div className='flex size-12 items-center justify-center rounded-2xl bg-white/14 ring-1 ring-white/20 backdrop-blur'>
              {modelIcon || (
                <span className='text-lg font-black'>{initial}</span>
              )}
            </div>
            <div className='text-xl font-bold tracking-tight break-all sm:text-2xl'>
              {model.model_name}
            </div>
          </div>
        </div>
      </Link>
      <div className='space-y-3 p-4'>
        <div className='flex items-start justify-between gap-3'>
          <div className='min-w-0'>
            <div className='flex items-center gap-2'>
              <h3 className='truncate text-base font-bold'>
                <Link to={detailHref} className='hover:text-brand'>
                  {model.model_name}
                </Link>
              </h3>
              <button
                type='button'
                onClick={handleCopy}
                className='text-muted-foreground hover:text-foreground shrink-0 transition-colors'
                aria-label={t('Copy model name')}
              >
                <Copy className='size-3.5' />
              </button>
            </div>
            <p className='text-muted-foreground mt-1 text-xs'>{vendor}</p>
          </div>
          <div className='bg-muted text-muted-foreground flex max-w-[52%] shrink-0 flex-wrap justify-end gap-x-2 gap-y-0.5 rounded-lg px-2.5 py-1 text-right text-[11px] leading-5'>
            {priceSummary}
          </div>
        </div>
        <p className='text-muted-foreground line-clamp-2 min-h-10 text-xs leading-5'>
          {model.description || t('No description available.')}
        </p>
        <div className='flex flex-wrap gap-2'>
          {tags.map((tag) => (
            <Badge
              key={tag}
              variant='outline'
              className='bg-background/70 text-foreground/80 text-[11px]'
            >
              {t(tag)}
            </Badge>
          ))}
          {endpoints.map((endpoint) => (
            <Badge key={endpoint} variant='secondary' className='text-[11px]'>
              {endpoint}
            </Badge>
          ))}
        </div>
        <div className='flex items-center justify-between gap-3'>
          <div className='text-muted-foreground truncate text-xs'>
            {groups.length > 0 ? groups.join(', ') : t('Default group')}
          </div>
          <Link
            to={detailHref}
            className='text-muted-foreground hover:text-foreground shrink-0 text-xs font-semibold'
          >
            {t('View model guide')}
          </Link>
        </div>
      </div>
    </article>
  )
}

function MarketSidebar(props: {
  vendors: Array<{ name: string; count: number; icon?: string }>
  tasks: Array<{ name: string; count: number }>
  activeTask: string
  activeVendor: string
  onTaskChange: (task: string) => void
  onVendorChange: (vendor: string) => void
}) {
  const { t } = useTranslation()

  return (
    <aside className='bg-card/92 rounded-2xl border p-4 shadow-sm'>
      <div className='space-y-5'>
        <div>
          <div className='text-sm font-semibold'>{t('Tasks')}</div>
          <div className='mt-2 flex flex-wrap gap-2'>
            <button
              type='button'
              onClick={() => props.onTaskChange('all')}
              className={cn(
                'rounded-xl border px-3 py-1.5 text-xs transition-colors',
                props.activeTask === 'all'
                  ? 'bg-foreground text-background'
                  : 'bg-background text-muted-foreground hover:text-foreground'
              )}
            >
              {t('All')}
            </button>
            {props.tasks.map((task) => (
              <button
                key={task.name}
                type='button'
                onClick={() => props.onTaskChange(task.name)}
                className={cn(
                  'rounded-xl border px-3 py-1.5 text-xs transition-colors',
                  props.activeTask === task.name
                    ? 'bg-foreground text-background'
                    : 'bg-background text-muted-foreground hover:text-foreground'
                )}
              >
                {t(task.name)} ({task.count})
              </button>
            ))}
          </div>
        </div>

        <div>
          <div className='text-sm font-semibold'>{t('Providers')}</div>
          <div className='mt-3 flex flex-wrap gap-2'>
            <button
              type='button'
              onClick={() => props.onVendorChange('all')}
              className={cn(
                'rounded-xl border px-3 py-1.5 text-xs transition-colors',
                props.activeVendor === 'all'
                  ? 'bg-foreground text-background'
                  : 'bg-background text-muted-foreground hover:text-foreground'
              )}
            >
              {t('All')}
            </button>
            {props.vendors.map((vendor) => (
              <button
                key={vendor.name}
                type='button'
                onClick={() => props.onVendorChange(vendor.name)}
                className={cn(
                  'inline-flex items-center gap-1.5 rounded-xl border px-3 py-1.5 text-xs transition-colors',
                  props.activeVendor === vendor.name
                    ? 'bg-foreground text-background'
                    : 'bg-background text-muted-foreground hover:text-foreground'
                )}
              >
                {vendor.icon ? (
                  <span className='flex size-4 shrink-0 items-center justify-center'>
                    {getLobeIcon(vendor.icon, 16)}
                  </span>
                ) : null}
                {vendor.name} ({vendor.count})
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
  const pricing = usePricingData()

  const models = useMemo<IndexedMarketModel[]>(() => {
    const source = pricing.models.length > 0 ? pricing.models : FALLBACK_MODELS

    return source.map((model) => {
      const marketKind = inferKind(model)
      return {
        ...model,
        marketKind,
        searchText: modelSignals(model).join(' ').toLowerCase(),
      }
    })
  }, [pricing.models])

  const marketSummary = useMemo(() => {
    const vendorInfo = new Map<string, { count: number; icon?: string }>()
    const taskCounts = new Map<string, number>(TASKS.map((task) => [task, 0]))
    const kindCounts = new Map<MarketKind, number>(
      MODEL_TYPES.map((item) => [item.value, 0])
    )

    for (const model of models) {
      if (model.vendor_name) {
        const existing = vendorInfo.get(model.vendor_name)
        vendorInfo.set(model.vendor_name, {
          count: (existing?.count ?? 0) + 1,
          icon: existing?.icon || model.vendor_icon || model.icon,
        })
      }
      kindCounts.set(
        model.marketKind,
        (kindCounts.get(model.marketKind) ?? 0) + 1
      )
      for (const task of TASKS) {
        if (modelMatchesTask(model, task)) {
          taskCounts.set(task, (taskCounts.get(task) ?? 0) + 1)
        }
      }
    }

    return {
      vendors: [...vendorInfo.entries()].map(([name, info]) => ({
        name,
        count: info.count,
        icon: info.icon,
      })),
      tasks: TASKS.map((task) => ({
        name: task,
        count: taskCounts.get(task) ?? 0,
      })),
      kindCounts,
    }
  }, [models])

  const filteredModels = useMemo(() => {
    const normalizedQuery = deferredQuery.trim().toLowerCase()
    return models.filter((model) => {
      if (model.marketKind !== activeKind) return false
      if (!modelMatchesTask(model, activeTask)) return false
      if (activeVendor !== 'all' && model.vendor_name !== activeVendor) {
        return false
      }
      if (!normalizedQuery) return true
      return model.searchText.includes(normalizedQuery)
    })
  }, [activeKind, activeTask, activeVendor, deferredQuery, models])

  return (
    <PublicLayout showMainContainer={false} showNotifications={false}>
      <main className='mx-auto max-w-7xl px-3 pt-24 pb-12 sm:px-4 md:px-6'>
        <section className='bg-card/92 rounded-2xl border px-4 py-8 text-center shadow-sm sm:rounded-3xl sm:px-6 md:px-10'>
          <div className='text-brand mx-auto inline-flex rounded-full border border-orange-200/80 bg-orange-50/80 px-3 py-1 font-mono text-[11px] font-black tracking-[0.22em] uppercase shadow-[inset_0_-1px_0_rgb(234_117_20/0.18)] dark:border-orange-500/20 dark:bg-orange-500/10'>
            MODEL MARKETPLACE
          </div>
          <h1 className='mx-auto mt-4 max-w-4xl text-3xl leading-tight font-bold tracking-tight sm:text-4xl md:text-5xl'>
            {t('Choose the right model before you write integration code')}
          </h1>
          <p className='text-muted-foreground mx-auto mt-3 max-w-3xl text-sm leading-7 md:text-base'>
            {t(
              'Compare providers, modalities, pricing signals, and endpoint compatibility in one scan, then open a model guide for implementation details.'
            )}
          </p>
        </section>

        <section className='mt-5 grid gap-5 lg:grid-cols-[240px_minmax(0,1fr)]'>
          <MarketSidebar
            vendors={marketSummary.vendors}
            tasks={marketSummary.tasks}
            activeTask={activeTask}
            activeVendor={activeVendor}
            onTaskChange={setActiveTask}
            onVendorChange={setActiveVendor}
          />

          <div className='bg-card/92 rounded-2xl border p-3 shadow-sm sm:rounded-3xl sm:p-4 md:p-5'>
            <div className='bg-background/65 space-y-4 rounded-2xl border p-3 sm:p-4'>
              <div>
                <h2 className='text-lg font-bold'>{t('Available models')}</h2>
                <p className='text-muted-foreground mt-1 text-sm'>
                  {t('Choose a text, image, video, or audio model first.')}
                </p>
                <div className='mt-3 flex flex-wrap gap-2'>
                  <Badge variant='outline'>
                    {t('Models')}: {models.length}
                  </Badge>
                  <Badge variant='outline'>
                    {t('Showing')}: {filteredModels.length}
                  </Badge>
                  <Badge variant='outline'>
                    {t('Providers')}: {marketSummary.vendors.length}
                  </Badge>
                </div>
              </div>

              <div className='grid gap-3 sm:grid-cols-2 lg:grid-cols-4'>
                {MODEL_TYPES.map((item) => {
                  const Icon = item.icon
                  const active = activeKind === item.value
                  return (
                    <button
                      key={item.value}
                      type='button'
                      onClick={() => setActiveKind(item.value)}
                      className={cn(
                        'group flex min-h-20 items-center justify-between gap-3 rounded-xl border p-3 text-left transition-all',
                        active
                          ? 'border-brand/60 bg-brand text-white shadow-[0_12px_28px_rgb(242_107_47/0.22)]'
                          : 'bg-card text-foreground hover:border-brand/35 hover:bg-muted/40'
                      )}
                    >
                      <span className='flex min-w-0 items-center gap-3'>
                        <span
                          className={cn(
                            'flex size-10 shrink-0 items-center justify-center rounded-lg transition-colors',
                            active
                              ? 'bg-white/18 text-white'
                              : 'bg-muted text-muted-foreground group-hover:text-foreground'
                          )}
                        >
                          <Icon className='size-5' />
                        </span>
                        <span className='min-w-0'>
                          <span className='block text-sm font-bold'>
                            {t(item.label)}
                          </span>
                          <span
                            className={cn(
                              'mt-0.5 block text-xs',
                              active ? 'text-white/72' : 'text-muted-foreground'
                            )}
                          >
                            {t('{{count}} models', {
                              count:
                                marketSummary.kindCounts.get(item.value) ?? 0,
                            })}
                          </span>
                        </span>
                      </span>
                    </button>
                  )
                })}
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
              {filteredModels.map((model) => (
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
          </div>
        </section>
      </main>
    </PublicLayout>
  )
}
