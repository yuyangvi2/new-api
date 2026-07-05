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
import { ArrowRight, FileText } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { useStatus } from '@/hooks/use-status'

import { HeroTerminalDemo } from '../hero-terminal-demo'

interface HeroProps {
  className?: string
  isAuthenticated?: boolean
}

export function Hero(props: HeroProps) {
  const { t } = useTranslation()
  const { status } = useStatus()
  const docsUrl =
    (status?.docs_link as string | undefined) || 'https://docs.newapi.pro'

  const renderDocsButton = () => {
    const isExternal = docsUrl.startsWith('http')
    if (isExternal) {
      return (
        <Button
          variant='outline'
          className='border-border hover:bg-muted/50 inline-flex h-11 items-center gap-2 rounded-md px-5 text-sm font-medium'
          render={
            <a href={docsUrl} target='_blank' rel='noopener noreferrer' />
          }
        >
          <FileText className='size-4' />
          <span>{t('Docs')}</span>
        </Button>
      )
    }
    return (
      <Button
        variant='outline'
        className='border-border hover:bg-muted/50 inline-flex h-11 items-center gap-2 rounded-md px-5 text-sm font-medium'
        render={<Link to={docsUrl} />}
      >
        <FileText className='size-4' />
        <span>{t('Docs')}</span>
      </Button>
    )
  }

  return (
    <section className='border-border/50 bg-background relative z-10 border-b px-4 pt-24 pb-12 md:px-8 md:pt-28 md:pb-16'>
      <div className='mx-auto grid max-w-7xl grid-cols-1 items-center gap-10 lg:grid-cols-[1fr_0.92fr] lg:gap-16'>
        <div className='flex flex-col items-start text-left'>
          <div
            className='landing-animate-fade-up border-border bg-muted/40 text-muted-foreground mb-6 inline-flex items-center gap-2 rounded-full border px-3 py-1.5 text-xs opacity-0'
            style={{ animationDelay: '0ms' }}
          >
            <span className='bg-brand size-1.5 rounded-full' />
            <span>{t('AI application infrastructure layer')}</span>
          </div>

          <h1
            className='landing-animate-fade-up max-w-3xl text-[clamp(2.625rem,7vw,4.25rem)] leading-[1.08] font-extrabold tracking-tight opacity-0'
            style={{ animationDelay: '60ms' }}
          >
            {t('A unified API for')}
            <br />
            <span className='text-brand'>{t('massive AI models')}</span>
          </h1>
          <p
            className='landing-animate-fade-up text-muted-foreground mt-7 max-w-2xl text-sm leading-7 opacity-0 md:text-base'
            style={{ animationDelay: '120ms' }}
          >
            {t(
              'Access OpenAI, Claude, Gemini, DeepSeek and 40+ providers through one standardized protocol.'
            )}
          </p>

          <div
            className='landing-animate-fade-up mt-8 flex flex-wrap items-center gap-3 opacity-0'
            style={{ animationDelay: '180ms' }}
          >
            {props.isAuthenticated ? (
              <>
                <Button
                  className='group h-11 rounded-md px-5 text-sm font-semibold'
                  render={<Link to='/dashboard' />}
                >
                  {t('Go to Dashboard')}
                  <ArrowRight className='ml-1.5 size-4 transition-transform duration-200 group-hover:translate-x-0.5' />
                </Button>
                {renderDocsButton()}
              </>
            ) : (
              <>
                <Button
                  className='group h-11 rounded-md px-5 text-sm font-semibold'
                  render={<Link to='/sign-up' />}
                >
                  {t('Get Started')}
                  <ArrowRight className='ml-1.5 size-4 transition-transform duration-200 group-hover:translate-x-0.5' />
                </Button>
                <Button
                  variant='outline'
                  className='border-border hover:bg-muted/50 h-11 rounded-md px-5 text-sm font-medium'
                  render={<Link to='/pricing' />}
                >
                  {t('View Pricing')}
                </Button>
                {renderDocsButton()}
              </>
            )}
          </div>

          <div
            className='landing-animate-fade-up mt-10 flex flex-wrap items-center gap-x-5 gap-y-2 opacity-0'
            style={{ animationDelay: '210ms' }}
          >
            <span className='text-muted-foreground text-[11px] font-semibold tracking-[0.2em] uppercase'>
              {t('Powered by')}
            </span>
            <div className='text-foreground/70 flex items-center gap-4 text-sm font-semibold'>
              <span>Tokone</span>
              <span>QuantumNous</span>
            </div>
          </div>
        </div>

        <div
          className='landing-animate-fade-up flex w-full justify-center opacity-0 lg:justify-end'
          style={{ animationDelay: '260ms' }}
        >
          <HeroTerminalDemo />
        </div>
      </div>
    </section>
  )
}
