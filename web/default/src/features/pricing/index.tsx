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
import { Link, useSearch } from '@tanstack/react-router'
import {
  ArrowRight,
  ChevronLeft,
  ChevronRight,
  Database,
  RefreshCw,
  Route,
  Search,
  ShieldCheck,
  Sparkles,
} from 'lucide-react'
import {
  useCallback,
  useEffect,
  useMemo,
  useState,
  type ComponentType,
} from 'react'
import { useTranslation } from 'react-i18next'

import { PublicLayout } from '@/components/layout'
import { PageTransition } from '@/components/page-transition'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'

import { LoadingSkeleton, ModelDetailsDrawer } from './components'
import { EXCLUDED_GROUPS, FILTER_ALL, getEndpointTypeLabels } from './constants'
import {
  getDynamicDisplayGroupRatio,
  getDynamicPricingSummary,
} from './lib/dynamic-price'
import { parseTags } from './lib/filters'
import { isTokenBasedModel } from './lib/model-helpers'
import { formatPrice, formatRequestPrice } from './lib/price'
import { usePricingData } from './hooks/use-pricing-data'
import type { PricingModel, TokenUnit } from './types'

type PricingModality = 'all' | 'text' | 'video' | 'image' | 'audio'

type PricingFamilyGroup = {
  key: string
  title: string
  modality: PricingModality
  models: PricingModel[]
}

type PricingFeature = {
  icon: ComponentType<{ className?: string }>
  title: string
  description: string
}

const GROUPS_PER_PAGE = 8
const TOKEN_UNIT: TokenUnit = 'M'

const KNOWN_MODEL_FAMILIES = [
  'claude',
  'deepseek',
  'doubao',
  'gemini',
  'glm',
  'gpt',
  'grok',
  'kimi',
  'kling',
  'minimax',
  'moonshot',
  'qwen',
  'sora',
  'veo',
] as const

const MODALITY_ORDER: PricingModality[] = [
  'all',
  'text',
  'video',
  'image',
  'audio',
]

function normalizeSearch(value: string): string {
  return value.trim().toLowerCase()
}

function includesAny(value: string, terms: string[]): boolean {
  return terms.some((term) => value.includes(term))
}

function getModelModality(model: PricingModel): PricingModality {
  const modelType = Number(model.model_type)
  if (modelType === 3) return 'video'
  if (modelType === 2) return 'image'
  if (modelType === 4) return 'audio'

  const endpoints = model.supported_endpoint_types || []
  const tags = parseTags(model.tags).map((tag) => tag.toLowerCase())
  const modalities = [
    ...(model.input_modalities || []),
    ...(model.output_modalities || []),
  ]
  const name = model.model_name.toLowerCase()
  const haystack = [...endpoints, ...tags, ...modalities, name].join(' ')

  if (
    includesAny(haystack, [
      'video',
      'openai-video',
      'sora',
      'veo',
      'kling',
      'vidu',
      'seedance',
    ])
  ) {
    return 'video'
  }

  if (
    includesAny(haystack, [
      'image',
      'images',
      'dall',
      'gpt-image',
      'flux',
      'midjourney',
      'nano-banana',
    ]) ||
    model.image_ratio != null
  ) {
    return 'image'
  }

  if (
    includesAny(haystack, ['audio', 'tts', 'stt', 'whisper', 'speech']) ||
    model.audio_ratio != null ||
    model.audio_completion_ratio != null
  ) {
    return 'audio'
  }

  return 'text'
}

function getModelFamily(model: PricingModel): string {
  const name = model.model_name.toLowerCase()
  for (const family of KNOWN_MODEL_FAMILIES) {
    if (name.includes(family)) {
      return family
    }
  }

  const firstToken = name.split(/[-_.:/\s]+/).find(Boolean)
  if (firstToken) return firstToken
  return model.vendor_name?.toLowerCase() || 'model'
}

function matchesSearch(model: PricingModel, query: string): boolean {
  if (!query) return true

  const values = [
    model.model_name,
    model.description,
    model.vendor_name,
    model.vendor_description,
    model.tags,
    ...(model.supported_endpoint_types || []),
    ...(model.enable_groups || []),
  ]

  return values.some((value) => value?.toLowerCase().includes(query))
}

function buildPricingGroups(models: PricingModel[]): PricingFamilyGroup[] {
  const grouped = new Map<string, PricingFamilyGroup>()

  for (const model of models) {
    const family = getModelFamily(model)
    const modality = getModelModality(model)
    const key = `${family}:${modality}`
    const current = grouped.get(key)

    if (current) {
      current.models.push(model)
    } else {
      grouped.set(key, {
        key,
        title: family,
        modality,
        models: [model],
      })
    }
  }

  return [...grouped.values()]
    .map((group) => ({
      ...group,
      models: [...group.models].sort((a, b) =>
        a.model_name.localeCompare(b.model_name)
      ),
    }))
    .sort((a, b) => {
      if (b.models.length !== a.models.length) {
        return b.models.length - a.models.length
      }
      return a.title.localeCompare(b.title)
    })
}

function getAvailableGroups(
  usableGroup: Record<string, { desc: string; ratio: number }>
): string[] {
  return Object.keys(usableGroup || {}).filter(
    (group) => !EXCLUDED_GROUPS.includes(group)
  )
}

function getGroupPreview(model: PricingModel): string[] {
  return (model.enable_groups || [])
    .filter((group) => group && group !== 'auto')
    .slice(0, 3)
}

function getModalityLabel(
  t: ReturnType<typeof useTranslation>['t'],
  modality: PricingModality
): string {
  switch (modality) {
    case 'all':
      return t('All')
    case 'video':
      return t('Video')
    case 'image':
      return t('Image')
    case 'audio':
      return t('Audio')
    case 'text':
    default:
      return t('Text')
  }
}

function getModelEndpointLabel(
  model: PricingModel,
  endpointLabels: Record<string, string>
): string {
  const endpoint = model.supported_endpoint_types?.[0]
  if (!endpoint) return 'API'
  return endpointLabels[endpoint] || endpoint
}

function PriceSummary(props: {
  model: PricingModel
  priceRate: number
  usdExchangeRate: number
  selectedGroup: string
}) {
  const { t } = useTranslation()
  const isDynamicPricing =
    props.model.billing_mode === 'tiered_expr' &&
    Boolean(props.model.billing_expr)
  const dynamicSummary = isDynamicPricing
    ? getDynamicPricingSummary(props.model, {
        tokenUnit: TOKEN_UNIT,
        showRechargePrice: false,
        priceRate: props.priceRate,
        usdExchangeRate: props.usdExchangeRate,
        groupRatioMultiplier: getDynamicDisplayGroupRatio(
          props.model,
          props.selectedGroup
        ),
      })
    : null

  if (dynamicSummary) {
    if (dynamicSummary.isSpecialExpression) {
      return (
        <div className='space-y-1'>
          <Badge variant='outline' className='border-amber-500/30 text-amber-700 dark:text-amber-300'>
            {t('Special billing expression')}
          </Badge>
          <code className='text-muted-foreground line-clamp-1 block text-xs break-all'>
            {dynamicSummary.rawExpression}
          </code>
        </div>
      )
    }

    if (dynamicSummary.primaryEntries.length > 0) {
      return (
        <div className='flex flex-wrap gap-x-4 gap-y-1'>
          {dynamicSummary.primaryEntries.map((entry) => (
            <span key={entry.key} className='text-muted-foreground text-xs'>
              {t(entry.shortLabel)}{' '}
              <span className='font-mono font-semibold text-orange-600 dark:text-orange-300'>
                {entry.formatted}
              </span>
              /1M
            </span>
          ))}
        </div>
      )
    }

    return (
      <Badge variant='outline' className='border-amber-500/30 text-amber-700 dark:text-amber-300'>
        {t('Dynamic Pricing')}
      </Badge>
    )
  }

  if (isTokenBasedModel(props.model)) {
    return (
      <div className='flex flex-wrap gap-x-4 gap-y-1'>
        <span className='text-muted-foreground text-xs'>
          {t('Input')}{' '}
          <span className='font-mono font-semibold text-orange-600 dark:text-orange-300'>
            {formatPrice(
              props.model,
              'input',
              TOKEN_UNIT,
              false,
              props.priceRate,
              props.usdExchangeRate,
              props.selectedGroup
            )}
          </span>
          /1M
        </span>
        <span className='text-muted-foreground text-xs'>
          {t('Output')}{' '}
          <span className='font-mono font-semibold text-orange-600 dark:text-orange-300'>
            {formatPrice(
              props.model,
              'output',
              TOKEN_UNIT,
              false,
              props.priceRate,
              props.usdExchangeRate,
              props.selectedGroup
            )}
          </span>
          /1M
        </span>
      </div>
    )
  }

  return (
    <span className='text-muted-foreground text-xs'>
      <span className='font-mono font-semibold text-orange-600 dark:text-orange-300'>
        {formatRequestPrice(
          props.model,
          false,
          props.priceRate,
          props.usdExchangeRate,
          props.selectedGroup
        )}
      </span>{' '}
      / {t('request')}
    </span>
  )
}

function PricingGroupCard(props: {
  group: PricingFamilyGroup
  endpointLabels: Record<string, string>
  priceRate: number
  usdExchangeRate: number
  selectedGroup: string
  onModelClick: (modelName: string) => void
}) {
  const { t } = useTranslation()

  return (
    <section className='overflow-hidden rounded-xl border border-border bg-card shadow-sm'>
      <div className='border-border/70 border-b bg-muted/20 px-4 py-3 sm:px-5'>
        <div className='flex flex-wrap items-center justify-between gap-2'>
          <div>
            <h2 className='text-foreground text-base font-semibold'>
              {props.group.title}
            </h2>
            <p className='text-muted-foreground mt-1 text-xs'>
              {t('{{count}} prices', {
                count: props.group.models.length,
              })}{' '}
              · {t('Direct pricing from gateway configuration')}
            </p>
          </div>
          <Badge variant='secondary'>
            {getModalityLabel(t, props.group.modality)}
          </Badge>
        </div>
      </div>

      <div className='text-muted-foreground hidden grid-cols-[minmax(0,1.35fr)_minmax(220px,1fr)_minmax(220px,1fr)] gap-4 px-4 py-2 text-xs font-medium sm:grid sm:px-5'>
        <span>{t('Model')}</span>
        <span>{t('Model pricing')}</span>
        <span>{t('Available groups')}</span>
      </div>

      <div className='divide-border/70 divide-y'>
        {props.group.models.map((model) => {
          const groups = getGroupPreview(model)
          const endpointLabel = getModelEndpointLabel(
            model,
            props.endpointLabels
          )
          return (
            <button
              key={model.model_name}
              type='button'
              onClick={() => props.onModelClick(model.model_name)}
              className='grid w-full gap-3 px-4 py-3 text-left transition-colors hover:bg-muted/30 sm:grid-cols-[minmax(0,1.35fr)_minmax(220px,1fr)_minmax(220px,1fr)] sm:items-center sm:px-5'
            >
              <span className='min-w-0'>
                <span className='text-foreground block truncate font-mono text-sm font-semibold'>
                  {model.model_name}
                </span>
                <span className='text-muted-foreground mt-1 flex flex-wrap items-center gap-1.5 text-xs'>
                  <span>{endpointLabel}</span>
                  {model.vendor_name && (
                    <>
                      <span aria-hidden='true'>·</span>
                      <span>{model.vendor_name}</span>
                    </>
                  )}
                </span>
              </span>

              <PriceSummary
                model={model}
                priceRate={props.priceRate}
                usdExchangeRate={props.usdExchangeRate}
                selectedGroup={props.selectedGroup}
              />

              <span className='flex flex-wrap items-center gap-1.5'>
                {groups.length > 0 ? (
                  groups.map((group) => (
                    <Badge
                      key={group}
                      variant='outline'
                      className='bg-background/70 font-mono'
                    >
                      {group}
                    </Badge>
                  ))
                ) : (
                  <span className='text-muted-foreground text-xs'>-</span>
                )}
                {model.enable_groups.length > groups.length && (
                  <span className='text-muted-foreground text-xs'>
                    +{model.enable_groups.length - groups.length}
                  </span>
                )}
              </span>
            </button>
          )
        })}
      </div>
    </section>
  )
}

function DecisionCard(props: PricingFeature) {
  const Icon = props.icon

  return (
    <div className='rounded-xl border border-border bg-card p-4 shadow-sm'>
      <div className='mb-3 flex size-8 items-center justify-center rounded-lg bg-primary/10 text-primary'>
        <Icon className='size-4' />
      </div>
      <h2 className='text-foreground text-sm font-semibold'>{props.title}</h2>
      <p className='text-muted-foreground mt-2 text-sm leading-relaxed'>
        {props.description}
      </p>
    </div>
  )
}

export function Pricing() {
  const { t } = useTranslation()
  const routeSearch = useSearch({ from: '/pricing/' })
  const [selectedModelName, setSelectedModelName] = useState<string | null>(
    null
  )
  const [searchInput, setSearchInput] = useState(routeSearch.search || '')
  const [activeModality, setActiveModality] = useState<PricingModality>('all')
  const [selectedGroup, setSelectedGroup] = useState(FILTER_ALL)
  const [currentPage, setCurrentPage] = useState(1)

  const {
    models,
    groupRatio,
    usableGroup,
    endpointMap,
    autoGroups,
    isLoading,
    refetch,
    priceRate,
    usdExchangeRate,
  } = usePricingData()

  const endpointLabels = getEndpointTypeLabels(t)
  const query = normalizeSearch(searchInput)

  const modalityCounts = useMemo(() => {
    const counts: Record<PricingModality, number> = {
      all: models.length,
      text: 0,
      video: 0,
      image: 0,
      audio: 0,
    }

    for (const model of models) {
      const modality = getModelModality(model)
      counts[modality] += 1
    }

    return counts
  }, [models])

  const availableGroups = useMemo(
    () => getAvailableGroups(usableGroup),
    [usableGroup]
  )

  const filteredModels = useMemo(
    () =>
      models.filter((model) => {
        const matchesModality =
          activeModality === 'all' || getModelModality(model) === activeModality
        const matchesGroup =
          selectedGroup === FILTER_ALL ||
          (model.enable_groups || []).includes(selectedGroup)
        return matchesModality && matchesGroup && matchesSearch(model, query)
      }),
    [activeModality, models, query, selectedGroup]
  )

  const pricingGroups = useMemo(
    () => buildPricingGroups(filteredModels),
    [filteredModels]
  )

  const totalPages = Math.max(
    1,
    Math.ceil(pricingGroups.length / GROUPS_PER_PAGE)
  )
  const visibleGroups = pricingGroups.slice(
    (currentPage - 1) * GROUPS_PER_PAGE,
    currentPage * GROUPS_PER_PAGE
  )

  const selectedModel = useMemo(() => {
    if (!selectedModelName) return null
    return models.find((model) => model.model_name === selectedModelName) || null
  }, [models, selectedModelName])

  const heroFeatures: PricingFeature[] = [
    {
      icon: Database,
      title: t('Clear billing units'),
      description: t(
        'Token, request, image, audio, and video pricing stay in one view.'
      ),
    },
    {
      icon: Route,
      title: t('Routing friendly'),
      description: t(
        'Use pricing data to design default models and fallback models.'
      ),
    },
    {
      icon: Sparkles,
      title: t('Multimodal coverage'),
      description: t('Compare text, image, video, and audio models together.'),
    },
    {
      icon: ShieldCheck,
      title: t('Production ready'),
      description: t(
        'Combine groups, endpoints, and quota policy to control live traffic risk.'
      ),
    },
  ]

  const startGroup = pricingGroups.length
    ? (currentPage - 1) * GROUPS_PER_PAGE + 1
    : 0
  const endGroup = Math.min(currentPage * GROUPS_PER_PAGE, pricingGroups.length)

  useEffect(() => {
    setCurrentPage(1)
  }, [activeModality, query, selectedGroup])

  useEffect(() => {
    if (
      selectedGroup !== FILTER_ALL &&
      !availableGroups.includes(selectedGroup)
    ) {
      setSelectedGroup(FILTER_ALL)
    }
  }, [availableGroups, selectedGroup])

  useEffect(() => {
    if (currentPage > totalPages) {
      setCurrentPage(totalPages)
    }
  }, [currentPage, totalPages])

  const handleModelClick = useCallback((modelName: string) => {
    setSelectedModelName(modelName)
  }, [])

  const handleRefresh = useCallback(() => {
    void refetch()
  }, [refetch])

  if (isLoading) {
    return (
      <PublicLayout showMainContainer={false}>
        <div className='mx-auto w-full max-w-7xl px-4 pt-24 pb-10 sm:px-6 lg:px-8'>
          <LoadingSkeleton viewMode='table' />
        </div>
      </PublicLayout>
    )
  }

  return (
    <PublicLayout showMainContainer={false}>
      <div className='relative'>
        <div
          aria-hidden
          className='pointer-events-none absolute inset-x-0 top-0 h-[620px] bg-[radial-gradient(circle_at_50%_0%,color-mix(in_oklch,var(--brand)_16%,transparent),transparent_48%),linear-gradient(180deg,color-mix(in_oklch,var(--accent)_14%,transparent),transparent_70%)]'
        />
        <PageTransition className='relative mx-auto w-full max-w-7xl px-4 pt-24 pb-12 sm:px-6 lg:px-8'>
          <section className='mx-auto max-w-4xl text-center'>
            <Badge
              variant='secondary'
              className='mb-5 rounded-full border border-border bg-card/80 px-3 py-1'
            >
              {t('Transparent pricing, pay as you go')}
            </Badge>
            <h1 className='text-foreground text-4xl leading-[1.05] font-semibold tracking-tight sm:text-5xl lg:text-6xl [font-family:var(--font-playfair-display),Georgia,serif]'>
              {t('Simple transparent API pricing')}
            </h1>
            <p className='text-muted-foreground mx-auto mt-5 max-w-2xl text-base leading-relaxed sm:text-lg'>
              {t(
                'No hidden fees. Compare enabled models by token, request, and endpoint support before routing production traffic.'
              )}
            </p>
          </section>

          <section className='mx-auto mt-7 max-w-3xl rounded-xl border border-border bg-card/85 p-5 shadow-sm backdrop-blur'>
            <h2 className='text-foreground text-lg font-semibold'>
              {t('Public pricing decision page')}
            </h2>
            <p className='text-muted-foreground mt-2 text-sm leading-relaxed'>
              {t(
                'Understand the billing unit first, then decide whether to compare models, open a model guide, or route from the console.'
              )}
            </p>
          </section>

          <div className='mt-7 flex flex-wrap items-center justify-center gap-3'>
            <Button
              render={<Link to='/market' />}
              className='h-11 rounded-full px-6'
            >
              {t('View model marketplace')}
              <ArrowRight className='size-4' />
            </Button>
            <Button
              variant='outline'
              render={<Link to='/pricing' />}
              className='h-11 rounded-full bg-card/70 px-6'
            >
              {t('Pricing')}
            </Button>
          </div>

          <section className='mt-8 grid gap-3 sm:grid-cols-2 lg:grid-cols-4'>
            {heroFeatures.map((feature) => (
              <DecisionCard key={feature.title} {...feature} />
            ))}
          </section>

          <section className='mt-8 rounded-xl border border-border bg-card/80 p-4 shadow-sm backdrop-blur sm:p-5'>
            <div className='flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between'>
              <div>
                <p className='text-primary text-xs font-semibold tracking-wide uppercase'>
                  {t('Model pricing overview')}
                </p>
                <h1 className='text-foreground mt-2 text-2xl font-bold'>
                  {t('Pricing')}
                </h1>
                <p className='text-muted-foreground mt-2 max-w-3xl text-sm leading-relaxed'>
                  {t(
                    'Reference prices are calculated from the current gateway configuration. Actual cost may vary by user group and promotion.'
                  )}
                </p>
                <p className='text-muted-foreground mt-1 max-w-3xl text-sm leading-relaxed'>
                  {t(
                    'Recommended order: narrow candidates in the model marketplace first, then confirm billing units and budget boundaries here.'
                  )}
                </p>
              </div>
              <Button
                type='button'
                variant='outline'
                size='sm'
                onClick={handleRefresh}
                className='w-fit rounded-full bg-card'
              >
                <RefreshCw className='size-4' />
                {t('Refresh')}
              </Button>
            </div>

            <div className='mt-5 flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between'>
              <div className='hover-scrollbar flex gap-2 overflow-x-auto pb-1'>
                {MODALITY_ORDER.map((modality) => (
                  <button
                    key={modality}
                    type='button'
                    onClick={() => setActiveModality(modality)}
                    className={cn(
                      'inline-flex shrink-0 items-center gap-1.5 rounded-full border px-3 py-1.5 text-sm transition-colors',
                      activeModality === modality
                        ? 'border-primary bg-primary text-primary-foreground'
                        : 'border-border bg-card text-muted-foreground hover:text-foreground'
                    )}
                  >
                    {getModalityLabel(t, modality)}
                    <span className='font-mono text-xs'>
                      {modalityCounts[modality]}
                    </span>
                  </button>
                ))}
              </div>

              <div className='relative w-full lg:max-w-sm'>
                <Search className='text-muted-foreground pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2' />
                <Input
                  value={searchInput}
                  onChange={(event) => setSearchInput(event.target.value)}
                  placeholder={t(
                    'Search models, modalities, providers, or endpoints...'
                  )}
                  className='h-10 rounded-lg bg-background pl-9'
                />
              </div>
            </div>

            {availableGroups.length > 0 && (
              <div className='mt-3 flex flex-col gap-2 sm:flex-row sm:items-center'>
                <span className='text-muted-foreground text-xs font-medium'>
                  {t('Groups')}
                </span>
                <div className='hover-scrollbar flex gap-2 overflow-x-auto pb-1'>
                  {[FILTER_ALL, ...availableGroups].map((group) => (
                    <button
                      key={group}
                      type='button'
                      onClick={() => setSelectedGroup(group)}
                      className={cn(
                        'inline-flex shrink-0 items-center rounded-full border px-3 py-1.5 text-xs transition-colors',
                        selectedGroup === group
                          ? 'border-primary bg-primary text-primary-foreground'
                          : 'border-border bg-card text-muted-foreground hover:text-foreground'
                      )}
                    >
                      {group === FILTER_ALL ? t('All') : group}
                    </button>
                  ))}
                </div>
              </div>
            )}
          </section>

          <main className='mt-4 space-y-4'>
            {visibleGroups.length > 0 ? (
              visibleGroups.map((group) => (
                <PricingGroupCard
                  key={group.key}
                  group={group}
                  endpointLabels={endpointLabels}
                  priceRate={priceRate}
                  usdExchangeRate={usdExchangeRate}
                  selectedGroup={selectedGroup}
                  onModelClick={handleModelClick}
                />
              ))
            ) : (
              <div className='rounded-xl border border-border bg-card p-10 text-center'>
                <h2 className='text-foreground text-lg font-semibold'>
                  {t('No pricing groups found')}
                </h2>
                <p className='text-muted-foreground mt-2 text-sm'>
                  {t('Adjust search or modality filters to continue.')}
                </p>
              </div>
            )}
          </main>

          {pricingGroups.length > 0 && (
            <div className='mt-4 flex flex-col gap-3 rounded-xl border border-border bg-card p-3 text-sm text-muted-foreground shadow-sm sm:flex-row sm:items-center sm:justify-between'>
              <span>
                {t('Show {{start}}-{{end}} of {{total}} model groups', {
                  start: startGroup,
                  end: endGroup,
                  total: pricingGroups.length,
                })}
              </span>
              <div className='flex items-center justify-between gap-3 sm:justify-end'>
                <Button
                  type='button'
                  variant='outline'
                  size='sm'
                  onClick={() => setCurrentPage((page) => Math.max(1, page - 1))}
                  disabled={currentPage <= 1}
                  className='rounded-full bg-card'
                >
                  <ChevronLeft className='size-4' />
                  {t('Previous page')}
                </Button>
                <span className='font-mono text-xs'>
                  {t('{{page}} / {{total}}', {
                    page: currentPage,
                    total: totalPages,
                  })}
                </span>
                <Button
                  type='button'
                  variant='outline'
                  size='sm'
                  onClick={() =>
                    setCurrentPage((page) => Math.min(totalPages, page + 1))
                  }
                  disabled={currentPage >= totalPages}
                  className='rounded-full bg-card'
                >
                  {t('Next page')}
                  <ChevronRight className='size-4' />
                </Button>
              </div>
            </div>
          )}

          <section className='mt-8 grid gap-4 lg:grid-cols-[1.2fr_0.8fr]'>
            <div className='rounded-xl border border-border bg-card p-5 shadow-sm'>
              <h2 className='text-foreground text-lg font-semibold'>
                {t('Pricing helps turn model cost into deployable route decisions.')}
              </h2>
              <div className='mt-4 grid gap-3 sm:grid-cols-3'>
                {[
                  t('Choose default and fallback models for different workloads.'),
                  t('Estimate how resolution, duration, or token ratio changes total cost.'),
                  t('Convert model unit price into budgets and routing rules.'),
                ].map((item) => (
                  <div key={item} className='rounded-lg bg-muted/30 p-3 text-sm text-muted-foreground'>
                    {item}
                  </div>
                ))}
              </div>
            </div>

            <div className='rounded-xl border border-border bg-card p-5 shadow-sm'>
              <h2 className='text-foreground text-lg font-semibold'>
                {t('Recommended workflow')}
              </h2>
              <ol className='text-muted-foreground mt-4 space-y-3 text-sm'>
                {[
                  t('Start in model marketplace to shortlist capability.'),
                  t('Confirm billing units and cost curves on pricing.'),
                  t('Open a model guide for request parameters and integration details.'),
                ].map((item, index) => (
                  <li key={item} className='flex gap-3'>
                    <span className='bg-primary/10 text-primary flex size-6 shrink-0 items-center justify-center rounded-full text-xs font-semibold'>
                      {index + 1}
                    </span>
                    <span>{item}</span>
                  </li>
                ))}
              </ol>
            </div>
          </section>

          <section className='mt-4 rounded-xl border border-border bg-card p-5 shadow-sm'>
            <div className='flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between'>
              <div>
                <h2 className='text-foreground text-base font-semibold'>
                  {t('Ready to route traffic?')}
                </h2>
                <p className='text-muted-foreground mt-1 text-sm'>
                  {t(
                    'Review the selected model in the dashboard and configure groups, keys, and budget controls.'
                  )}
                </p>
              </div>
              <div className='text-muted-foreground flex flex-wrap gap-2 text-xs'>
                {availableGroups.slice(0, 4).map((group) => (
                  <Badge key={group} variant='outline' className='font-mono'>
                    {group}
                  </Badge>
                ))}
              </div>
            </div>
          </section>

          {selectedModel && (
            <ModelDetailsDrawer
              open={Boolean(selectedModel)}
              onOpenChange={(open) => {
                if (!open) setSelectedModelName(null)
              }}
              model={selectedModel}
              groupRatio={groupRatio || {}}
              usableGroup={usableGroup || {}}
              endpointMap={
                (endpointMap as Record<
                  string,
                  { path?: string; method?: string }
                >) || {}
              }
              autoGroups={autoGroups || []}
              priceRate={priceRate ?? 1}
              usdExchangeRate={usdExchangeRate ?? 1}
              tokenUnit={TOKEN_UNIT}
              showRechargePrice={false}
            />
          )}
        </PageTransition>
      </div>
    </PublicLayout>
  )
}
