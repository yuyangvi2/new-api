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
  Cloud,
  Code,
  DollarSign,
  Gauge,
  Shield,
  Users,
  Zap,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { AnimateInView } from '@/components/animate-in-view'
import { cn } from '@/lib/utils'

interface FeaturesProps {
  className?: string
}

export function Features(props: FeaturesProps) {
  const { t } = useTranslation()

  const primaryFeatures = [
    {
      id: 'latency',
      title: t('Ultra-low latency'),
      desc: t(
        'Optimized routing and nearby scheduling keep responses fast and stable.'
      ),
      icon: Zap,
      span: 'md:col-span-8',
      visual: <ModelChips />,
    },
    {
      id: 'security',
      title: t('Secure and reliable'),
      desc: t(
        'Enterprise key custody, token groups, permissions, Passkeys and OAuth sign-in.'
      ),
      icon: Shield,
      span: 'md:col-span-4',
      visual: (
        <div className='text-brand/25 mt-8 flex justify-center'>
          <Shield className='size-16' strokeWidth={1.4} />
        </div>
      ),
    },
    {
      id: 'dispatch',
      title: t('Intelligent scheduling'),
      desc: t(
        'Multi-channel load balancing, fallback retry, pass-through routing and health checks.'
      ),
      icon: Gauge,
      span: 'md:col-span-4',
      visual: null,
    },
    {
      id: 'developer',
      title: t('Developer friendly'),
      desc: t(
        'OpenAI, Claude and Gemini compatible routes with familiar API and SDK workflows.'
      ),
      icon: Code,
      span: 'md:col-span-8',
      visual: <DeveloperDots />,
    },
  ] as const

  const secondaryFeatures = [
    {
      title: t('High performance'),
      desc: t('Automatic load balancing'),
      icon: Gauge,
    },
    {
      title: t('Transparent billing'),
      desc: t('Real-time usage monitoring'),
      icon: DollarSign,
    },
    {
      title: t('Team collaboration'),
      desc: t('Multi-user permissions'),
      icon: Users,
    },
    {
      title: t('Open and self-hosted'),
      desc: t('Community driven deployment'),
      icon: Cloud,
    },
  ] as const

  return (
    <section
      className={cn(
        'relative z-10 bg-background px-4 py-16 md:px-8',
        props.className
      )}
    >
      <div className='mx-auto max-w-7xl'>
        <AnimateInView className='mb-10 text-center'>
          <p className='text-brand mb-3 text-xs font-semibold tracking-[0.22em] uppercase'>
            {t('Features')}
          </p>
          <h2 className='text-2xl leading-tight font-bold tracking-tight md:text-3xl'>
            {t('Built for developers, designed for scale')}
          </h2>
        </AnimateInView>

        <div className='grid gap-6 md:grid-cols-12'>
          {primaryFeatures.map((feature, index) => {
            const Icon = feature.icon
            return (
              <AnimateInView
                key={feature.id}
                delay={index * 80}
                animation='fade-up'
                className={cn(
                  'min-h-[220px] rounded-lg border border-border bg-card p-7 transition-colors hover:bg-muted/20',
                  feature.span
                )}
              >
                <div className='bg-brand/10 text-brand mb-7 flex size-10 items-center justify-center rounded-md'>
                  <Icon className='size-5' strokeWidth={1.8} />
                </div>
                <h3 className='text-lg font-semibold tracking-tight'>
                  {feature.title}
                </h3>
                <p className='text-muted-foreground mt-4 max-w-2xl text-sm leading-6'>
                  {feature.desc}
                </p>
                {feature.visual}
              </AnimateInView>
            )
          })}
        </div>

        <div className='mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4'>
          {secondaryFeatures.map((feature, index) => {
            const Icon = feature.icon
            return (
              <AnimateInView
                key={feature.title}
                delay={index * 70}
                animation='fade-up'
                className='border-border bg-card flex min-h-24 items-start gap-4 rounded-lg border p-5'
              >
                <Icon className='text-brand mt-0.5 size-5 shrink-0' />
                <div>
                  <h3 className='text-sm font-semibold'>{feature.title}</h3>
                  <p className='text-muted-foreground mt-1 text-xs leading-5'>
                    {feature.desc}
                  </p>
                </div>
              </AnimateInView>
            )
          })}
        </div>
      </div>
    </section>
  )
}

function ModelChips() {
  return (
    <div className='mt-12 flex flex-wrap gap-3'>
      {['OpenAI', 'Claude', 'DeepSeek', 'Gemini', 'Qwen'].map((name) => (
        <span
          key={name}
          className='border-border bg-muted/60 text-muted-foreground rounded border px-4 py-2 text-xs font-medium'
        >
          {name}
        </span>
      ))}
    </div>
  )
}

function DeveloperDots() {
  return (
    <div className='mt-9 flex items-center gap-4'>
      <div className='flex -space-x-2'>
        {['API', 'SDK', 'CLI'].map((label, index) => (
          <span
            key={label}
            className={cn(
              'flex size-9 items-center justify-center rounded-full border-2 border-background text-[10px] font-bold',
              index === 0 && 'bg-foreground text-background',
              index === 1 && 'bg-brand text-white',
              index === 2 && 'bg-muted text-muted-foreground'
            )}
          >
            {label}
          </span>
        ))}
      </div>
    </div>
  )
}
