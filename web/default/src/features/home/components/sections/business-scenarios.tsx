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
import {
  BarChart3,
  Code2,
  FileText,
  Headphones,
  ShoppingBag,
  Workflow,
  type LucideIcon,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { cn } from '@/lib/utils'

interface ScenarioItem {
  title: string
  description: string
  icon: LucideIcon
  accent: string
}

const SCENARIOS: ScenarioItem[] = [
  {
    title: 'Campaign content factory',
    description:
      'Generate scripts, captions, visuals, and short-form video drafts from one routed model stack.',
    icon: FileText,
    accent: 'from-orange-500/20 to-amber-300/10 text-orange-600',
  },
  {
    title: 'Commerce visual operations',
    description:
      'Create product scenes, try-on concepts, listing images, and localized ad variants without rebuilding pipelines.',
    icon: ShoppingBag,
    accent: 'from-sky-500/20 to-cyan-300/10 text-sky-600',
  },
  {
    title: 'Customer support copilot',
    description:
      'Route simple questions to fast models and complex cases to stronger reasoning models while keeping cost predictable.',
    icon: Headphones,
    accent: 'from-emerald-500/20 to-lime-300/10 text-emerald-600',
  },
  {
    title: 'Engineering assistant layer',
    description:
      'Give teams access to coding, review, and debugging models through one governed endpoint.',
    icon: Code2,
    accent: 'from-violet-500/20 to-fuchsia-300/10 text-violet-600',
  },
  {
    title: 'Research and reporting',
    description:
      'Summarize filings, calls, documents, and market signals with long-context models and traceable prompts.',
    icon: BarChart3,
    accent: 'from-rose-500/20 to-orange-300/10 text-rose-600',
  },
  {
    title: 'Workflow automation',
    description:
      'Connect agents, tools, and approvals so recurring business tasks run with observable model usage.',
    icon: Workflow,
    accent: 'from-slate-700/15 to-teal-300/10 text-teal-600',
  },
]

export function BusinessScenarios() {
  const { t } = useTranslation()

  return (
    <section className='dark:border-border/60 dark:bg-background relative overflow-hidden border-y border-orange-100/70 bg-[#fffaf2] py-16 md:py-20'>
      <div className='pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_15%_18%,rgb(255_171_45/0.16),transparent_32%),radial-gradient(circle_at_88%_42%,rgb(14_165_233/0.12),transparent_28%),linear-gradient(180deg,rgb(255_255_255/0.55),transparent_54%)] dark:opacity-30' />
      <div className='relative mx-auto max-w-7xl px-4 md:px-6'>
        <div className='mx-auto max-w-3xl text-center'>
          <div className='text-brand text-xs font-bold tracking-[0.24em] uppercase'>
            {t('Business scenarios')}
          </div>
          <h2 className='mt-3 text-3xl leading-tight font-bold sm:text-4xl'>
            {t('From creative production to enterprise automation')}
          </h2>
          <p className='text-muted-foreground mt-4 text-sm leading-7 md:text-base'>
            {t(
              'Use the same model gateway for content, operations, engineering, and research teams, with routing and billing managed in one place.'
            )}
          </p>
        </div>

        <div className='mt-10 grid gap-4 sm:grid-cols-2 lg:grid-cols-3'>
          {SCENARIOS.map((scenario) => {
            const Icon = scenario.icon
            return (
              <article
                key={scenario.title}
                className='group bg-card/86 relative min-h-48 overflow-hidden rounded-2xl border p-5 shadow-sm transition-all duration-300 hover:-translate-y-0.5 hover:shadow-md'
              >
                <div
                  className={cn(
                    'absolute inset-x-0 top-0 h-24 bg-gradient-to-br opacity-80',
                    scenario.accent
                  )}
                />
                <div className='relative'>
                  <div
                    className={cn(
                      'flex size-11 items-center justify-center rounded-2xl bg-white/80 ring-1 ring-black/5 transition-transform duration-300 group-hover:scale-105 dark:bg-white/10 dark:ring-white/10',
                      scenario.accent
                    )}
                  >
                    <Icon className='size-5' aria-hidden='true' />
                  </div>
                  <h3 className='mt-6 text-base font-bold'>
                    {t(scenario.title)}
                  </h3>
                  <p className='text-muted-foreground mt-3 text-sm leading-6'>
                    {t(scenario.description)}
                  </p>
                </div>
              </article>
            )
          })}
        </div>
      </div>
    </section>
  )
}
