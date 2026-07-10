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
import { Link, useParams } from '@tanstack/react-router'
import {
  ArrowLeft,
  BookOpen,
  Database,
  Gauge,
  ShieldCheck,
  Sparkles,
} from 'lucide-react'
import { useMemo, type ComponentType } from 'react'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { PublicLayout } from '@/components/layout'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { ModelDetailsContent } from '@/features/pricing/components/model-details'
import { DEFAULT_TOKEN_UNIT } from '@/features/pricing/constants'
import { usePricingData } from '@/features/pricing/hooks/use-pricing-data'
import type { PricingModel } from '@/features/pricing/types'
import { cn } from '@/lib/utils'

import {
  FALLBACK_MODELS,
  inferKind,
  marketKindLabelKey,
  toModelGuideSlug,
  type MarketKind,
} from './lib/model-catalog'

function modelDescription(
  model: PricingModel,
  t: ReturnType<typeof useTranslation>['t']
) {
  if (model.description) return model.description
  const provider = model.vendor_name || t('Unknown provider')
  const kind = t(marketKindLabelKey(inferKind(model)))
  return t(
    '{{model}} is a {{kind}} model from {{provider}}. Use this guide to review pricing, supported endpoints, capabilities, and the fastest compatible API path.',
    {
      model: model.model_name,
      kind,
      provider,
    }
  )
}

function modelTone(kind: MarketKind) {
  if (kind === 'image') {
    return {
      accent: 'text-cyan-700 bg-cyan-500/10 dark:text-cyan-300',
      surface:
        'from-cyan-50 via-white to-emerald-50 dark:from-cyan-950/35 dark:via-background dark:to-emerald-950/25',
      line: 'from-cyan-400 via-emerald-300 to-orange-300',
    }
  }
  if (kind === 'video') {
    return {
      accent: 'text-orange-700 bg-orange-500/10 dark:text-orange-300',
      surface:
        'from-orange-50 via-white to-rose-50 dark:from-orange-950/35 dark:via-background dark:to-rose-950/25',
      line: 'from-orange-400 via-rose-300 to-violet-400',
    }
  }
  if (kind === 'audio') {
    return {
      accent: 'text-emerald-700 bg-emerald-500/10 dark:text-emerald-300',
      surface:
        'from-emerald-50 via-white to-sky-50 dark:from-emerald-950/35 dark:via-background dark:to-sky-950/25',
      line: 'from-emerald-400 via-teal-300 to-sky-300',
    }
  }
  return {
    accent: 'text-orange-700 bg-orange-500/10 dark:text-orange-300',
    surface:
      'from-slate-50 via-white to-orange-50 dark:from-slate-950/35 dark:via-background dark:to-orange-950/25',
    line: 'from-orange-400 via-amber-300 to-sky-300',
  }
}

function formatCompactCount(
  count: number,
  t: ReturnType<typeof useTranslation>['t']
) {
  if (count <= 0) return t('Not configured')
  return String(count)
}

function ModelStat(props: {
  label: string
  value: string
  icon: ComponentType<{ className?: string }>
}) {
  const Icon = props.icon
  return (
    <div className='border-border/70 bg-card/80 min-w-0 rounded-2xl border px-4 py-4 shadow-sm'>
      <div className='text-muted-foreground flex items-center gap-2'>
        <Icon className='size-3.5 shrink-0' />
        <span className='truncate text-[11px] font-semibold tracking-[0.16em] uppercase'>
          {props.label}
        </span>
      </div>
      <div className='text-foreground mt-2 truncate text-sm font-semibold'>
        {props.value}
      </div>
    </div>
  )
}

function ModelHero(props: {
  model: PricingModel
  description: string
  kind: MarketKind
}) {
  const { t } = useTranslation()
  const model = props.model
  const tone = modelTone(props.kind)
  const kindLabel = marketKindLabelKey(props.kind)
  const groups = model.enable_groups ?? []

  return (
    <section
      className={cn(
        'relative overflow-hidden border-b bg-gradient-to-br',
        tone.surface
      )}
    >
      <div className='pointer-events-none absolute inset-0 bg-[linear-gradient(120deg,rgb(255_255_255/0.72)_0%,rgb(255_255_255/0.28)_45%,transparent_100%)] dark:bg-[linear-gradient(120deg,rgb(255_255_255/0.06)_0%,transparent_56%)]' />
      <div className='via-border pointer-events-none absolute inset-x-0 bottom-0 h-px bg-gradient-to-r from-transparent to-transparent' />

      <div className='relative mx-auto max-w-7xl px-4 pt-24 pb-10 md:px-6 md:pt-28 md:pb-14'>
        <div className='mb-6 flex flex-wrap items-center justify-between gap-3'>
          <Button
            variant='outline'
            className='bg-background/72 h-9 rounded-full shadow-sm backdrop-blur'
            render={<Link to='/market' />}
          >
            <ArrowLeft className='size-4' />
            {t('Back to Model Marketplace')}
          </Button>
          <div className='flex flex-wrap gap-2'>
            <CopyButton
              value={model.model_name}
              variant='outline'
              size='sm'
              className='bg-background/72 h-9 rounded-full shadow-sm backdrop-blur'
              iconClassName='size-3.5'
              tooltip={t('Copy model name')}
              successTooltip={t('Copied!')}
              aria-label={t('Copy model name')}
            >
              {t('Copy model name')}
            </CopyButton>
            <Button
              variant='outline'
              className='bg-background/72 hidden h-9 rounded-full shadow-sm backdrop-blur sm:inline-flex'
              render={<Link to='/pricing' />}
            >
              <BookOpen className='size-4' />
              {t('View pricing')}
            </Button>
          </div>
        </div>

        <div className='min-w-0'>
          <div
            className={cn(
              'inline-flex rounded-full border px-3 py-1 font-mono text-[11px] font-black tracking-[0.22em] uppercase',
              tone.accent
            )}
          >
            {t('Production model guide')}
          </div>
          <h1 className='text-foreground mt-5 max-w-5xl text-4xl leading-[1.05] font-black tracking-tight break-words sm:text-5xl lg:text-6xl'>
            {model.model_name}
          </h1>
          <p className='text-muted-foreground mt-5 max-w-3xl text-base leading-8 md:text-lg'>
            {props.description}
          </p>
          <div className='mt-7 flex flex-wrap gap-2'>
            <Badge
              variant='secondary'
              className='rounded-full px-3 py-1 shadow-sm'
            >
              {model.vendor_name || t('Unknown provider')}
            </Badge>
            <Badge
              variant='secondary'
              className='rounded-full px-3 py-1 shadow-sm'
            >
              {t(kindLabel)}
            </Badge>
            <Badge
              variant='secondary'
              className='rounded-full px-3 py-1 shadow-sm'
            >
              {model.quota_type === 0 ? t('Token-based') : t('Per Request')}
            </Badge>
          </div>
        </div>

        <div className='mt-8 grid gap-3 md:grid-cols-4'>
          <ModelStat
            icon={Sparkles}
            label={t('Provider')}
            value={model.vendor_name || t('Unknown provider')}
          />
          <ModelStat icon={Gauge} label={t('Type')} value={t(kindLabel)} />
          <ModelStat
            icon={Database}
            label={t('Billing')}
            value={model.quota_type === 0 ? t('Token-based') : t('Per Request')}
          />
          <ModelStat
            icon={ShieldCheck}
            label={t('Available groups')}
            value={formatCompactCount(groups.length, t)}
          />
        </div>
      </div>
    </section>
  )
}

export function ModelGuide() {
  const { t } = useTranslation()
  const { modelSlug } = useParams({ from: '/model-guide/$modelSlug/' })
  const {
    models,
    groupRatio,
    usableGroup,
    endpointMap,
    autoGroups,
    isLoading,
    priceRate,
    usdExchangeRate,
  } = usePricingData()

  const allModels = useMemo(() => {
    const source = models.length > 0 ? models : FALLBACK_MODELS
    return source.map((model) => ({
      ...model,
      marketKind: inferKind(model),
    }))
  }, [models])

  const model = useMemo(
    () =>
      allModels.find(
        (item) => toModelGuideSlug(item.model_name) === modelSlug
      ) ?? null,
    [allModels, modelSlug]
  )

  if (isLoading && models.length === 0 && !model) {
    return (
      <PublicLayout showMainContainer={false} showNotifications={false}>
        <main className='mx-auto max-w-6xl px-4 pt-28 pb-16 md:px-6'>
          <Skeleton className='h-10 w-40 rounded-full' />
          <Skeleton className='mt-6 h-72 w-full rounded-[28px]' />
          <div className='mt-6 grid gap-4 md:grid-cols-3'>
            {['endpoint', 'pricing', 'validation'].map((item) => (
              <Skeleton key={item} className='h-44 rounded-2xl' />
            ))}
          </div>
        </main>
      </PublicLayout>
    )
  }

  if (!model) {
    return (
      <PublicLayout showMainContainer={false} showNotifications={false}>
        <main className='mx-auto flex min-h-[70vh] max-w-2xl flex-col items-center justify-center px-4 text-center'>
          <h1 className='text-2xl font-bold'>{t('Model not found')}</h1>
          <p className='text-muted-foreground mt-3 text-sm'>
            {t("The model you're looking for doesn't exist.")}
          </p>
          <Button
            className='mt-6 rounded-full'
            variant='outline'
            render={<Link to='/market' />}
          >
            <ArrowLeft className='size-4' />
            {t('Back to Model Marketplace')}
          </Button>
        </main>
      </PublicLayout>
    )
  }

  const description = modelDescription(model, t)
  const kind = inferKind(model)

  return (
    <PublicLayout showMainContainer={false} showNotifications={false}>
      <main className='bg-muted/20 pb-16'>
        <ModelHero model={model} description={description} kind={kind} />

        <div className='relative mt-6 space-y-6'>
          <section className='mx-auto max-w-7xl px-4 md:px-6'>
            <div className='mb-4 flex flex-wrap items-end justify-between gap-3'>
              <div>
                <div className='text-brand font-mono text-[11px] font-black tracking-[0.22em] uppercase'>
                  {t('Pricing, performance, and API contract')}
                </div>
                <h2 className='mt-2 text-2xl font-bold tracking-tight'>
                  {t('Inspect the model before production rollout')}
                </h2>
              </div>
              <p className='text-muted-foreground max-w-md text-sm leading-7'>
                {t(
                  'Use the tabs below to inspect pricing rules, live performance signals, and compatible API paths.'
                )}
              </p>
            </div>

            <div className='bg-card/92 rounded-[28px] border p-4 shadow-[0_24px_70px_rgb(15_23_42/0.08)] backdrop-blur md:p-6'>
              <ModelDetailsContent
                model={model}
                groupRatio={groupRatio || {}}
                usableGroup={usableGroup || {}}
                autoGroups={autoGroups || []}
                priceRate={priceRate ?? 1}
                usdExchangeRate={usdExchangeRate ?? 1}
                tokenUnit={DEFAULT_TOKEN_UNIT}
                showHeader={false}
                endpointMap={
                  (endpointMap as Record<
                    string,
                    { path?: string; method?: string }
                  >) || {}
                }
              />
            </div>
          </section>
        </div>
      </main>
    </PublicLayout>
  )
}
