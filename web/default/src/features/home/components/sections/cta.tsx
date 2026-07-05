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
    <section className='bg-background relative z-10 px-4 py-16 md:px-8'>
      <AnimateInView
        className='relative mx-auto max-w-7xl overflow-hidden rounded-lg bg-black px-6 py-14 text-center text-white md:px-12 md:py-20'
        animation='scale-in'
      >
        <div className='relative z-10'>
          <h2 className='text-3xl leading-tight font-extrabold tracking-tight md:text-5xl'>
            {t('Ready to simplify')}
            <br />
            {t('your AI integration?')}
          </h2>
          <p className='mx-auto mt-6 max-w-2xl text-sm leading-6 text-white/75 md:text-base'>
            {t(
              'Deploy your own gateway and start routing requests through your configured upstream services.'
            )}
          </p>
          <div className='mt-8 flex flex-wrap items-center justify-center gap-3'>
            <Button
              className='group h-12 rounded-md bg-white px-7 font-semibold text-black hover:bg-white/90'
              render={<Link to='/sign-up' />}
            >
              {t('Get Started')}
              <ArrowRight className='ml-1 size-3.5 transition-transform duration-200 group-hover:translate-x-0.5' />
            </Button>
            <Button
              variant='outline'
              className='h-12 rounded-md border-white/30 bg-transparent px-7 font-semibold text-white hover:bg-white/10 hover:text-white'
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
