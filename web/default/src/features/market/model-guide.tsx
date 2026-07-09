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
  ArrowRight,
  BookOpen,
  CheckCircle2,
  Code2,
  Database,
  Gauge,
  Layers,
  Network,
  ShieldCheck,
  Sparkles,
  Zap,
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
  splitTags,
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
      glow: 'from-cyan-400/28 via-emerald-300/18 to-orange-300/24',
      orb: 'bg-cyan-300/24',
      line: 'from-cyan-400 via-emerald-300 to-orange-300',
    }
  }
  if (kind === 'video') {
    return {
      glow: 'from-orange-400/28 via-rose-300/18 to-violet-400/24',
      orb: 'bg-orange-300/24',
      line: 'from-orange-400 via-rose-300 to-violet-400',
    }
  }
  if (kind === 'audio') {
    return {
      glow: 'from-emerald-400/26 via-teal-300/18 to-sky-300/24',
      orb: 'bg-emerald-300/24',
      line: 'from-emerald-400 via-teal-300 to-sky-300',
    }
  }
  return {
    glow: 'from-orange-400/26 via-amber-200/18 to-sky-300/22',
    orb: 'bg-orange-300/22',
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
    <div className='min-w-0 border-t border-white/12 px-4 py-4 sm:border-t-0 sm:border-l'>
      <div className='flex items-center gap-2 text-white/58'>
        <Icon className='size-3.5 shrink-0' />
        <span className='truncate text-[11px] font-semibold tracking-[0.16em] uppercase'>
          {props.label}
        </span>
      </div>
      <div className='mt-2 truncate text-sm font-semibold text-white'>
        {props.value}
      </div>
    </div>
  )
}

function HeroRouteMap(props: {
  kind: MarketKind
  endpoints: string[]
  groups: string[]
}) {
  const { t } = useTranslation()
  const tone = modelTone(props.kind)
  const endpointLabel =
    props.endpoints.length > 0 ? props.endpoints[0] : t('Compatible endpoint')
  const groupLabel =
    props.groups.length > 0
      ? props.groups.slice(0, 2).join(', ')
      : t('Default group')

  const nodes = [
    { label: 'Model request', value: t('Client request'), icon: Code2 },
    { label: 'Endpoint', value: endpointLabel, icon: Network },
    { label: 'Routing group', value: groupLabel, icon: Layers },
    { label: 'Production response', value: t('Observable output'), icon: Zap },
  ]

  return (
    <div className='relative min-h-[320px] overflow-hidden rounded-[26px] border border-white/12 bg-white/[0.07] p-5 shadow-2xl backdrop-blur-xl'>
      <div
        className={cn(
          'pointer-events-none absolute -top-20 -right-16 size-56 rounded-full blur-3xl',
          tone.orb
        )}
      />
      <div className='relative'>
        <div className='flex items-center justify-between gap-3'>
          <div>
            <div className='font-mono text-[11px] font-black tracking-[0.24em] text-white/52 uppercase'>
              {t('Gateway profile')}
            </div>
            <div className='mt-1 text-sm font-semibold text-white'>
              {t('Production routing preview')}
            </div>
          </div>
          <div className='rounded-full border border-white/14 bg-white/10 px-3 py-1 text-xs font-semibold text-white/75'>
            {t('Gateway ready')}
          </div>
        </div>

        <div className='mt-7 space-y-3'>
          {nodes.map((node, index) => {
            const Icon = node.icon
            return (
              <div key={node.label} className='relative'>
                {index > 0 && (
                  <div className='absolute -top-3 left-5 h-3 w-px bg-white/16' />
                )}
                <div className='flex items-center gap-3 rounded-2xl border border-white/10 bg-black/18 px-3 py-3'>
                  <div className='flex size-10 shrink-0 items-center justify-center rounded-xl bg-white/10 text-white'>
                    <Icon className='size-4' />
                  </div>
                  <div className='min-w-0 flex-1'>
                    <div className='text-[11px] font-semibold tracking-[0.14em] text-white/48 uppercase'>
                      {t(node.label)}
                    </div>
                    <div className='mt-0.5 truncate text-sm font-semibold text-white'>
                      {node.value}
                    </div>
                  </div>
                  {index < nodes.length - 1 && (
                    <ArrowRight className='size-4 text-white/35' />
                  )}
                </div>
              </div>
            )
          })}
        </div>

        <div
          className={cn('mt-6 h-1.5 rounded-full bg-gradient-to-r', tone.line)}
        />
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
  const endpoints = model.supported_endpoint_types ?? []
  const groups = model.enable_groups ?? []

  return (
    <section className='relative overflow-hidden bg-[#17110d] text-white'>
      <div
        className={cn(
          'pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_20%_10%,var(--tw-gradient-from),transparent_30%),radial-gradient(circle_at_75%_20%,var(--tw-gradient-via),transparent_28%),linear-gradient(135deg,rgb(23_17_13),rgb(35_30_25)_58%,rgb(12_16_20))]',
          tone.glow
        )}
      />
      <div className='pointer-events-none absolute inset-0 bg-[linear-gradient(rgba(255,255,255,0.05)_1px,transparent_1px),linear-gradient(90deg,rgba(255,255,255,0.04)_1px,transparent_1px)] bg-[size:44px_44px] opacity-25' />

      <div className='relative mx-auto max-w-7xl px-4 pt-24 pb-10 md:px-6 md:pt-28 md:pb-14'>
        <div className='mb-6 flex flex-wrap items-center justify-between gap-3'>
          <Button
            variant='outline'
            className='h-9 rounded-full border-white/18 bg-white/10 text-white shadow-none backdrop-blur hover:bg-white/16 hover:text-white'
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
              className='h-9 rounded-full border-white/18 bg-white/10 text-white shadow-none hover:bg-white/16 hover:text-white'
              iconClassName='size-3.5'
              tooltip={t('Copy model name')}
              successTooltip={t('Copied!')}
              aria-label={t('Copy model name')}
            >
              {t('Copy model name')}
            </CopyButton>
            <Button
              variant='outline'
              className='hidden h-9 rounded-full border-white/18 bg-white/10 text-white shadow-none backdrop-blur hover:bg-white/16 hover:text-white sm:inline-flex'
              render={<Link to='/pricing' />}
            >
              <BookOpen className='size-4' />
              {t('View pricing')}
            </Button>
          </div>
        </div>

        <div className='grid gap-8 lg:grid-cols-[minmax(0,1fr)_430px] lg:items-end'>
          <div className='min-w-0'>
            <div className='inline-flex rounded-full border border-white/14 bg-white/10 px-3 py-1 font-mono text-[11px] font-black tracking-[0.22em] text-white/72 uppercase'>
              {t('Production model guide')}
            </div>
            <h1 className='mt-5 max-w-5xl text-4xl leading-[1.05] font-black tracking-tight break-words text-white sm:text-5xl lg:text-6xl'>
              {model.model_name}
            </h1>
            <p className='mt-5 max-w-3xl text-base leading-8 text-white/72 md:text-lg'>
              {props.description}
            </p>
            <div className='mt-7 flex flex-wrap gap-2'>
              <Badge className='rounded-full border-white/12 bg-white/12 px-3 py-1 text-white hover:bg-white/12'>
                {model.vendor_name || t('Unknown provider')}
              </Badge>
              <Badge className='rounded-full border-white/12 bg-white/12 px-3 py-1 text-white hover:bg-white/12'>
                {t(kindLabel)}
              </Badge>
              <Badge className='rounded-full border-white/12 bg-white/12 px-3 py-1 text-white hover:bg-white/12'>
                {model.quota_type === 0 ? t('Token-based') : t('Per Request')}
              </Badge>
            </div>
          </div>

          <HeroRouteMap
            kind={props.kind}
            endpoints={endpoints}
            groups={groups}
          />
        </div>

        <div className='mt-8 grid overflow-hidden rounded-[22px] border border-white/12 bg-white/[0.06] backdrop-blur md:grid-cols-4'>
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

function GuideStep(props: {
  index: string
  title: string
  description: string
  icon: ComponentType<{ className?: string }>
}) {
  const { t } = useTranslation()
  const Icon = props.icon
  return (
    <article className='group bg-card relative overflow-hidden rounded-[24px] border p-5 shadow-sm transition-all duration-300 hover:-translate-y-1 hover:shadow-xl'>
      <div className='flex items-start justify-between gap-4'>
        <div className='flex size-11 items-center justify-center rounded-2xl bg-orange-100 text-orange-700 dark:bg-orange-500/15 dark:text-orange-300'>
          <Icon className='size-5' />
        </div>
        <span className='text-muted-foreground/14 font-mono text-4xl leading-none font-black'>
          {props.index}
        </span>
      </div>
      <h2 className='mt-5 text-base font-bold'>{t(props.title)}</h2>
      <p className='text-muted-foreground mt-2 text-sm leading-7'>
        {t(props.description)}
      </p>
    </article>
  )
}

function CompatibilitySignals(props: { model: PricingModel }) {
  const { t } = useTranslation()
  const endpoints = props.model.supported_endpoint_types ?? []
  const tags = splitTags(props.model.tags)

  return (
    <aside className='bg-card rounded-[24px] border p-5 shadow-sm'>
      <div className='flex items-center justify-between gap-4'>
        <div>
          <div className='text-brand font-mono text-[11px] font-black tracking-[0.2em] uppercase'>
            {t('Compatibility snapshot')}
          </div>
          <h2 className='mt-2 text-lg font-bold'>
            {t('Supported model signals')}
          </h2>
        </div>
        <Network className='text-muted-foreground size-5' />
      </div>

      <div className='mt-5 space-y-5'>
        <div>
          <div className='text-muted-foreground text-xs font-semibold tracking-[0.14em] uppercase'>
            {t('Supported endpoints')}
          </div>
          <div className='mt-3 flex flex-wrap gap-2'>
            {endpoints.length > 0 ? (
              endpoints.map((endpoint) => (
                <Badge
                  key={endpoint}
                  variant='outline'
                  className='rounded-full'
                >
                  {endpoint}
                </Badge>
              ))
            ) : (
              <span className='text-muted-foreground text-sm'>
                {t('No endpoint data yet')}
              </span>
            )}
          </div>
        </div>

        <div>
          <div className='text-muted-foreground text-xs font-semibold tracking-[0.14em] uppercase'>
            {t('Capability tags')}
          </div>
          <div className='mt-3 flex flex-wrap gap-2'>
            {tags.length > 0 ? (
              tags.map((tag) => (
                <Badge key={tag} variant='secondary' className='rounded-full'>
                  {t(tag)}
                </Badge>
              ))
            ) : (
              <span className='text-muted-foreground text-sm'>
                {t('No capability tags yet')}
              </span>
            )}
          </div>
        </div>
      </div>
    </aside>
  )
}

function IntegrationSection(props: { model: PricingModel }) {
  const { t } = useTranslation()
  const steps = [
    {
      index: '01',
      title: 'Pick the endpoint',
      description:
        'Match the model to chat, response, image, audio, or video APIs before implementation.',
      icon: Code2,
    },
    {
      index: '02',
      title: 'Estimate production cost',
      description:
        'Review token, request, and group pricing before routing live traffic.',
      icon: Database,
    },
    {
      index: '03',
      title: 'Validate and route',
      description:
        'Test prompts, streaming, retries, and fallback groups with representative requests.',
      icon: CheckCircle2,
    },
  ]

  return (
    <section className='mx-auto grid max-w-7xl gap-5 px-4 md:px-6 lg:grid-cols-[minmax(0,1fr)_360px]'>
      <div>
        <div className='mb-4 flex flex-wrap items-end justify-between gap-3'>
          <div>
            <div className='text-brand font-mono text-[11px] font-black tracking-[0.22em] uppercase'>
              {t('Integration path')}
            </div>
            <h2 className='mt-2 text-2xl font-bold tracking-tight'>
              {t('From selection to stable traffic')}
            </h2>
          </div>
          <p className='text-muted-foreground max-w-md text-sm leading-7'>
            {t(
              'Pricing and routing data are pulled from the current gateway configuration.'
            )}
          </p>
        </div>
        <div className='grid gap-4 md:grid-cols-3'>
          {steps.map((step) => (
            <GuideStep key={step.index} {...step} />
          ))}
        </div>
      </div>

      <CompatibilitySignals model={props.model} />
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
      <main className='dark:bg-background bg-[#fbf7ef] pb-16'>
        <ModelHero model={model} description={description} kind={kind} />

        <div className='relative -mt-5 space-y-6'>
          <IntegrationSection model={model} />

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
