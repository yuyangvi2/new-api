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
  textureClassName: string
  iconClassName: string
}

const SCENARIOS: ScenarioItem[] = [
  {
    title: 'Campaign content factory',
    description:
      'Generate scripts, captions, visuals, and short-form video drafts from one routed model stack.',
    icon: FileText,
    textureClassName:
      'bg-[radial-gradient(circle_at_18%_18%,rgb(249_115_22/0.055),transparent_42%),linear-gradient(135deg,rgb(249_115_22/0.035),transparent_62%)]',
    iconClassName:
      'bg-orange-500/[0.08] text-orange-700/80 dark:text-orange-300/80',
  },
  {
    title: 'Commerce visual operations',
    description:
      'Create product scenes, try-on concepts, listing images, and localized ad variants without rebuilding pipelines.',
    icon: ShoppingBag,
    textureClassName:
      'bg-[radial-gradient(circle_at_18%_18%,rgb(14_165_233/0.055),transparent_42%),linear-gradient(135deg,rgb(14_165_233/0.035),transparent_62%)]',
    iconClassName: 'bg-sky-500/[0.08] text-sky-700/80 dark:text-sky-300/80',
  },
  {
    title: 'Customer support copilot',
    description:
      'Route simple questions to fast models and complex cases to stronger reasoning models while keeping cost predictable.',
    icon: Headphones,
    textureClassName:
      'bg-[radial-gradient(circle_at_18%_18%,rgb(16_185_129/0.055),transparent_42%),linear-gradient(135deg,rgb(16_185_129/0.035),transparent_62%)]',
    iconClassName:
      'bg-emerald-500/[0.08] text-emerald-700/80 dark:text-emerald-300/80',
  },
  {
    title: 'Engineering assistant layer',
    description:
      'Give teams access to coding, review, and debugging models through one governed endpoint.',
    icon: Code2,
    textureClassName:
      'bg-[radial-gradient(circle_at_18%_18%,rgb(139_92_246/0.05),transparent_42%),linear-gradient(135deg,rgb(139_92_246/0.032),transparent_62%)]',
    iconClassName:
      'bg-violet-500/[0.08] text-violet-700/80 dark:text-violet-300/80',
  },
  {
    title: 'Research and reporting',
    description:
      'Summarize filings, calls, documents, and market signals with long-context models and traceable prompts.',
    icon: BarChart3,
    textureClassName:
      'bg-[radial-gradient(circle_at_18%_18%,rgb(244_63_94/0.05),transparent_42%),linear-gradient(135deg,rgb(244_63_94/0.032),transparent_62%)]',
    iconClassName: 'bg-rose-500/[0.08] text-rose-700/80 dark:text-rose-300/80',
  },
  {
    title: 'Workflow automation',
    description:
      'Connect agents, tools, and approvals so recurring business tasks run with observable model usage.',
    icon: Workflow,
    textureClassName:
      'bg-[radial-gradient(circle_at_18%_18%,rgb(20_184_166/0.055),transparent_42%),linear-gradient(135deg,rgb(20_184_166/0.035),transparent_62%)]',
    iconClassName: 'bg-teal-500/[0.08] text-teal-700/80 dark:text-teal-300/80',
  },
]

export function BusinessScenarios() {
  const { t } = useTranslation()

  return (
    <section className='dark:border-border/60 dark:bg-background relative overflow-hidden border-y border-orange-100/70 bg-[#fffaf2] py-16 md:py-20'>
      <div className='pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_15%_18%,rgb(255_171_45/0.16),transparent_32%),radial-gradient(circle_at_88%_42%,rgb(14_165_233/0.12),transparent_28%),linear-gradient(180deg,rgb(255_255_255/0.55),transparent_54%)] dark:bg-[radial-gradient(circle_at_15%_18%,rgb(234_117_20/0.14),transparent_34%),radial-gradient(circle_at_88%_42%,rgb(14_165_233/0.12),transparent_30%),linear-gradient(180deg,rgb(255_255_255/0.06),transparent_56%)]' />
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
                className='group bg-card/92 relative min-h-48 overflow-hidden rounded-xl border border-border/80 p-5 shadow-sm transition-all duration-300 hover:-translate-y-0.5 hover:border-brand/30 hover:shadow-md dark:bg-card/80'
              >
                <div
                  className={cn(
                    'pointer-events-none absolute inset-0 opacity-80 dark:opacity-45',
                    scenario.textureClassName
                  )}
                />
                <div className='relative'>
                  <div
                    className={cn(
                      'flex size-11 items-center justify-center rounded-lg ring-1 ring-black/5 transition-transform duration-300 group-hover:scale-105 dark:ring-white/10',
                      scenario.iconClassName
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
