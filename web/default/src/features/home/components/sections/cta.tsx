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
    <section className='relative z-10 px-6 py-16 md:py-24'>
      <AnimateInView
        className='bg-primary text-primary-foreground relative mx-auto max-w-6xl overflow-hidden rounded-3xl px-6 py-16 text-center md:px-12 md:py-20'
        animation='scale-in'
      >
        {/* Subtle radial glow inside the dark card */}
        <div
          aria-hidden
          className='pointer-events-none absolute inset-0 -z-0 opacity-20'
          style={{
            background:
              'radial-gradient(ellipse 55% 60% at 30% 40%, #2d4bff 0%, transparent 70%)',
          }}
        />
        <div className='relative z-10'>
          <h2 className='text-2xl leading-tight font-bold tracking-tight md:text-4xl'>
            {t('Ready to simplify')}
            <br />
            {t('your AI integration?')}
          </h2>
          <p className='text-primary-foreground/70 mx-auto mt-5 max-w-xl text-sm leading-relaxed md:text-base'>
            {t(
              'Deploy your own gateway and start routing requests through your configured upstream services.'
            )}
          </p>
          <div className='mt-8 flex flex-wrap items-center justify-center gap-3'>
            <Button
              className='bg-background text-foreground hover:bg-background/90 group h-11 rounded-lg px-6 font-medium'
              render={<Link to='/sign-up' />}
            >
              {t('Get Started')}
              <ArrowRight className='ml-1 size-3.5 transition-transform duration-200 group-hover:translate-x-0.5' />
            </Button>
            <Button
              variant='outline'
              className='border-primary-foreground/30 text-primary-foreground hover:bg-primary-foreground/10 hover:text-primary-foreground h-11 rounded-lg bg-transparent px-6 font-medium'
              render={<Link to='/pricing' />}
            >
              {t('View Pricing')}
            </Button>
          </div>
        </div>
      </AnimateInView>
    </section>
  )
}
