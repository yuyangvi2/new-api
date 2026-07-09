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
import { useMemo, useState, type ComponentType, type MouseEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { PublicLayout } from '@/components/layout'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'

import { usePricingData } from '../pricing/hooks/use-pricing-data'
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

const TASKS = ['Chat']

function MarketModelCard(props: { model: MarketModel }) {
  const { t } = useTranslation()
  const model = props.model
  const vendor = model.vendor_name || t('Unknown provider')
  const tags = splitTags(model.tags).slice(0, 4)
  const detailHref = `/model-guide/${toModelGuideSlug(model.model_name)}`
  const price =
    model.quota_type === 1
      ? t('{{amount}} credits', {
          amount: model.model_price ?? model.model_ratio,
        })
      : t('{{amount}} credits/M', {
          amount: Math.round(model.model_ratio * 100),
        })

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
          <div className='relative px-5 text-center text-xl font-bold tracking-tight break-all text-white drop-shadow-sm sm:px-6 sm:text-2xl'>
            {model.model_name}
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
          <Badge variant='secondary' className='h-6 rounded-full'>
            {price}
          </Badge>
        </div>
        <div className='flex flex-wrap gap-2'>
          {tags.map((tag) => (
            <Badge
              key={tag}
              variant='outline'
              className='bg-background/70 text-[11px] text-emerald-700'
            >
              {t(tag)}
            </Badge>
          ))}
        </div>
        <Link
          to={detailHref}
          className='text-muted-foreground hover:text-foreground inline-flex text-xs font-semibold'
        >
          {t('View model guide')}
        </Link>
      </div>
    </article>
  )
}

function MarketSidebar(props: {
  vendors: Array<{ name: string; count: number }>
  activeVendor: string
  onVendorChange: (vendor: string) => void
}) {
  const { t } = useTranslation()

  return (
    <aside className='bg-card/92 rounded-2xl border p-4 shadow-sm'>
      <div className='space-y-5'>
        <div>
          <div className='text-sm font-semibold'>{t('Tasks')}</div>
          <div className='text-muted-foreground mt-2 text-xs font-semibold uppercase'>
            {t('Chat')}
          </div>
          <div className='mt-2 flex flex-wrap gap-2'>
            {TASKS.map((task) => (
              <Button
                key={task}
                variant='outline'
                size='sm'
                className='bg-background h-8 rounded-xl px-3 text-xs'
              >
                {t(task)}
              </Button>
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
                  'rounded-xl border px-3 py-1.5 text-xs transition-colors',
                  props.activeVendor === vendor.name
                    ? 'bg-foreground text-background'
                    : 'bg-background text-muted-foreground hover:text-foreground'
                )}
              >
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
  const [activeKind, setActiveKind] = useState<MarketKind>('text')
  const [activeVendor, setActiveVendor] = useState('all')
  const pricing = usePricingData()

  const models = useMemo<MarketModel[]>(() => {
    const source =
      pricing.models.length > 0
        ? pricing.models.map((model) => ({
            ...model,
            marketKind: inferKind(model),
          }))
        : FALLBACK_MODELS

    return source
  }, [pricing.models])

  const vendors = useMemo(() => {
    const counts = new Map<string, number>()
    for (const model of models) {
      if (!model.vendor_name) continue
      counts.set(model.vendor_name, (counts.get(model.vendor_name) ?? 0) + 1)
    }
    return [...counts.entries()].map(([name, count]) => ({ name, count }))
  }, [models])

  const filteredModels = useMemo(() => {
    const normalizedQuery = query.trim().toLowerCase()
    return models.filter((model) => {
      if ((model.marketKind ?? inferKind(model)) !== activeKind) return false
      if (activeVendor !== 'all' && model.vendor_name !== activeVendor) {
        return false
      }
      if (!normalizedQuery) return true
      return [model.model_name, model.vendor_name, model.tags]
        .filter(Boolean)
        .join(' ')
        .toLowerCase()
        .includes(normalizedQuery)
    })
  }, [activeKind, activeVendor, models, query])

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
            vendors={vendors}
            activeVendor={activeVendor}
            onVendorChange={setActiveVendor}
          />

          <div className='bg-card/92 rounded-2xl border p-3 shadow-sm sm:rounded-3xl sm:p-4 md:p-5'>
            <div className='bg-background/65 grid gap-4 rounded-2xl border p-3 sm:p-4 md:grid-cols-[1fr_360px] md:items-center'>
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
                    {t('Providers')}: {vendors.length}
                  </Badge>
                </div>
              </div>

              <div className='space-y-3'>
                <div className='relative'>
                  <Search className='text-muted-foreground pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2' />
                  <Input
                    value={query}
                    onChange={(event) => setQuery(event.target.value)}
                    placeholder={t('Search model, task, or provider')}
                    className='bg-card h-11 rounded-full pr-4 pl-9'
                  />
                </div>
                <div className='grid grid-cols-2 gap-2 sm:grid-cols-4'>
                  {MODEL_TYPES.map((item) => {
                    const Icon = item.icon
                    const active = activeKind === item.value
                    return (
                      <button
                        key={item.value}
                        type='button'
                        onClick={() => setActiveKind(item.value)}
                        className={cn(
                          'flex h-10 min-w-0 items-center justify-center gap-2 rounded-xl border px-2 text-sm font-semibold transition-colors',
                          active
                            ? 'bg-foreground text-background'
                            : 'bg-card text-muted-foreground hover:text-foreground'
                        )}
                      >
                        <Icon className='size-4' />
                        {t(item.label)}
                      </button>
                    )
                  })}
                </div>
              </div>
            </div>

            <div className='mt-4 grid gap-5 md:grid-cols-2 xl:grid-cols-3'>
              {filteredModels.map((model) => (
                <MarketModelCard key={model.model_name} model={model} />
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
