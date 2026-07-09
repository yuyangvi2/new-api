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
import { ArrowRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { AnimateInView } from '@/components/animate-in-view'
import { Button } from '@/components/ui/button'

interface CTAProps {
  className?: string
  isAuthenticated?: boolean
}

export function CTA(props: CTAProps) {
  const { t } = useTranslation()

  if (props.isAuthenticated) {
    return null
  }

  return (
    <section className='dark:bg-background relative z-10 overflow-hidden bg-[#fbf7ef] px-4 py-16 sm:py-20 md:px-8'>
      <div className='pointer-events-none absolute inset-0 bg-[linear-gradient(180deg,rgb(251_247_239)_0%,rgb(255_250_243)_46%,rgb(251_247_239)_100%)] dark:bg-[linear-gradient(180deg,rgb(9_9_11)_0%,rgb(20_20_22)_48%,rgb(9_9_11)_100%)]' />
      <div className='pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_50%_36%,rgb(255_171_45/0.2),transparent_38%)] dark:bg-[radial-gradient(circle_at_50%_36%,rgb(234_117_20/0.18),transparent_40%)]' />
      <AnimateInView
        className='relative mx-auto max-w-4xl text-center'
        animation='fade-up'
      >
        <p className='text-brand mb-3 text-xs font-semibold tracking-[0.22em] uppercase'>
          {t('Ready to start?')}
        </p>
        <h2 className='text-3xl leading-[1.08] font-semibold tracking-normal sm:text-4xl md:text-5xl [font-family:var(--font-playfair-display),Georgia,serif]'>
          {t('Ready to unify access to AI models?')}
        </h2>
        <p className='text-muted-foreground mx-auto mt-5 max-w-2xl text-sm leading-6 md:text-base'>
          {t(
            'Use one compatible API to connect models, billing, routing, and operations.'
          )}
        </p>
        <div className='mt-8 flex flex-wrap items-center justify-center gap-3'>
          <Button
            className='group bg-brand hover:bg-brand-hover h-14 w-full max-w-48 rounded-full px-10 text-lg font-semibold text-white shadow-[0_10px_28px_rgb(234_117_20/0.22)] sm:w-48'
            render={<Link to='/sign-up' />}
          >
            {t('Start now')}
            <ArrowRight className='ml-1 size-3.5 transition-transform duration-200 group-hover:translate-x-0.5' />
          </Button>
          <Button
            variant='outline'
            className='border-primary/18 dark:bg-card/80 h-14 w-full max-w-48 rounded-full bg-white/72 px-10 text-lg font-semibold shadow-[0_10px_28px_rgb(234_117_20/0.1)] backdrop-blur hover:bg-white dark:hover:bg-card sm:w-48'
            render={<Link to='/market' />}
          >
            {t('View models')}
          </Button>
        </div>
      </AnimateInView>
    </section>
  )
}
