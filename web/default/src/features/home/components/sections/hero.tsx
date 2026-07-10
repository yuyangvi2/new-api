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
import { ArrowRight, ChevronLeft, ChevronRight, Sparkles } from 'lucide-react'
import {
  useEffect,
  useMemo,
  useState,
  type ComponentType,
  type SVGProps,
} from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

import {
  IconAlibabaCloud,
  IconAnthropic,
  IconByteDance,
  IconDeepSeek,
  IconGemini,
  IconOpenAI,
} from '../model-brand-icons'

interface HeroProps {
  className?: string
  isAuthenticated?: boolean
}

const HERO_SLIDES = [
  {
    eyebrow: 'Video',
    title: 'Seedance 2.5',
    description:
      'A long-form video generation model for cinematic stories and product shots.',
    primary: 'View model',
    secondary: 'Coming soon',
    href: '/market',
    tone: 'from-orange-100 via-amber-50 to-stone-100',
    backgroundImage: '/images/hero/tokone-hero-slide-1.webp',
    tags: ['Seedance 2.5', '99.9% SLA', '50+ AI models', '5min migration'],
  },
  {
    eyebrow: 'Reasoning',
    title: 'Gemini 3.5 Flash',
    description:
      'Low-latency multimodal reasoning for chat, vision, and agent workflows.',
    primary: 'View model',
    secondary: 'API docs',
    href: '/market',
    tone: 'from-blue-100 via-cyan-50 to-stone-100',
    backgroundImage: '/images/hero/tokone-hero-slide-2.webp',
    tags: ['Fast response', 'Vision', 'Function calling', 'OpenAI compatible'],
  },
  {
    eyebrow: 'Creative',
    title: 'GPT Image Studio',
    description:
      'High quality image generation and editing through one unified API.',
    primary: 'View model',
    secondary: 'Pricing',
    href: '/market',
    tone: 'from-fuchsia-100 via-rose-50 to-stone-100',
    backgroundImage: '/images/hero/tokone-hero-slide-3.webp',
    tags: ['Text to Image', 'Image edit', 'Batch tasks', 'Stable routing'],
  },
  {
    eyebrow: 'Model Access',
    title: 'Ready to unify access to AI models?',
    description:
      'Use one compatible API to connect models, billing, routing, and operations.',
    primary: 'View models',
    secondary: 'API docs',
    href: '/market',
    tone: 'from-emerald-100 via-cyan-50 to-stone-100',
    backgroundImage: '/images/hero/tokone-hero-slide-4.webp',
    tags: [
      'OpenAI compatible',
      'Transparent Billing',
      'Stable routing',
      'Model Access',
    ],
  },
]

const HERO_BACKGROUND_IMAGES = HERO_SLIDES.map((slide) => slide.backgroundImage)

const PARTNERS: Array<{
  name: string
  logo: ComponentType<SVGProps<SVGSVGElement>>
}> = [
  { name: 'OpenAI', logo: IconOpenAI },
  { name: 'Anthropic', logo: IconAnthropic },
  {
    name: 'Google',
    logo: IconGemini,
  },
  {
    name: 'ByteDance',
    logo: IconByteDance,
  },
  {
    name: 'Alibaba',
    logo: IconAlibabaCloud,
  },
  { name: 'DeepSeek', logo: IconDeepSeek },
]

export function Hero(props: HeroProps) {
  const { t } = useTranslation()
  const [activeIndex, setActiveIndex] = useState(0)
  const activeSlide = HERO_SLIDES[activeIndex]

  useEffect(() => {
    const interval = window.setInterval(() => {
      setActiveIndex((index) => (index + 1) % HERO_SLIDES.length)
    }, 6000)

    return () => window.clearInterval(interval)
  }, [])

  const heroAction = useMemo(
    () => (props.isAuthenticated ? '/dashboard' : activeSlide.href),
    [activeSlide.href, props.isAuthenticated]
  )

  const goToPrevious = () => {
    setActiveIndex(
      (index) => (index - 1 + HERO_SLIDES.length) % HERO_SLIDES.length
    )
  }

  const goToNext = () => {
    setActiveIndex((index) => (index + 1) % HERO_SLIDES.length)
  }

  return (
    <section
      className={cn(
        'relative overflow-hidden bg-[#fbf7ef] dark:bg-background',
        props.className
      )}
    >
      <div className='dark:border-border/70 dark:bg-background relative overflow-hidden border-b border-orange-100/80 bg-[#fbf7ef] transition-colors duration-500'>
        <div className='dark:bg-background pointer-events-none absolute inset-0 bg-[#fbf7ef]' />
        {HERO_BACKGROUND_IMAGES.map((image, index) => (
          <img
            key={image}
            src={image}
            alt=''
            aria-hidden='true'
            fetchPriority={index === 0 ? 'high' : 'auto'}
            decoding='async'
            style={{ opacity: index === activeIndex ? 1 : 0 }}
            className='pointer-events-none absolute inset-0 size-full object-cover object-center contrast-[1.08] saturate-[1.08] transition-opacity duration-700'
          />
        ))}
        <div className='pointer-events-none absolute inset-0 bg-[linear-gradient(180deg,rgb(251_247_239/0.5)_0%,rgb(251_247_239/0.28)_42%,rgb(251_247_239/0.74)_100%)] dark:bg-[linear-gradient(180deg,rgb(9_9_11/0.66)_0%,rgb(9_9_11/0.44)_42%,rgb(9_9_11/0.86)_100%)]' />
        <div className='pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_50%_38%,rgb(255_171_45/0.24),transparent_42%),linear-gradient(90deg,rgb(251_247_239/0.42)_0%,rgb(251_247_239/0.06)_30%,rgb(251_247_239/0.04)_70%,rgb(251_247_239/0.36)_100%)] dark:bg-[radial-gradient(circle_at_50%_38%,rgb(234_117_20/0.28),transparent_42%),linear-gradient(90deg,rgb(9_9_11/0.64)_0%,rgb(9_9_11/0.18)_32%,rgb(9_9_11/0.16)_68%,rgb(9_9_11/0.58)_100%)]' />
        <div className='absolute inset-x-0 bottom-0 h-1.5 bg-gradient-to-r from-orange-500 via-orange-400 to-amber-300 dark:from-orange-500/70 dark:via-amber-400/55 dark:to-cyan-300/45' />

        <button
          type='button'
          onClick={goToPrevious}
          className='bg-card/80 text-muted-foreground hover:text-foreground absolute top-1/2 left-4 z-10 hidden size-10 -translate-y-1/2 items-center justify-center rounded-full border shadow-sm transition-colors md:flex'
          aria-label={t('Previous')}
        >
          <ChevronLeft className='size-5' />
        </button>
        <button
          type='button'
          onClick={goToNext}
          className='bg-card/80 text-muted-foreground hover:text-foreground absolute top-1/2 right-4 z-10 hidden size-10 -translate-y-1/2 items-center justify-center rounded-full border shadow-sm transition-colors md:flex'
          aria-label={t('Next')}
        >
          <ChevronRight className='size-5' />
        </button>

        <div className='relative z-10 mx-auto flex min-h-[540px] max-w-7xl flex-col items-center justify-center px-4 pt-20 text-center sm:min-h-[560px] md:min-h-[620px] md:px-6'>
          <div className='bg-card/80 text-brand inline-flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-xs font-semibold shadow-sm'>
            <Sparkles className='size-3.5' />
            {t(activeSlide.eyebrow)}
          </div>

          <h1 className='text-foreground mt-5 max-w-4xl [font-family:var(--font-playfair-display),Georgia,serif] text-4xl leading-[1.05] font-semibold tracking-normal sm:text-5xl lg:text-6xl'>
            {t(activeSlide.title)}
          </h1>
          <p className='text-muted-foreground mt-4 max-w-3xl text-base leading-7 md:text-lg'>
            {t(activeSlide.description)}
          </p>

          <div className='mt-7 flex flex-wrap items-center justify-center gap-3'>
            <Button
              className='bg-brand hover:bg-brand-hover h-11 min-w-40 rounded-full px-6 text-sm font-semibold text-white'
              render={<Link to={heroAction} />}
            >
              {props.isAuthenticated
                ? t('Go to Dashboard')
                : t(activeSlide.primary)}
              <ArrowRight className='size-4' />
            </Button>
            <Button
              variant='outline'
              className='bg-card/55 h-11 min-w-40 rounded-full px-6 text-sm font-semibold'
              render={<Link to='/pricing' />}
            >
              {t(activeSlide.secondary)}
            </Button>
          </div>

          <div className='mt-6 flex flex-wrap justify-center gap-2'>
            {activeSlide.tags.map((tag) => (
              <span
                key={tag}
                className='bg-card/75 text-muted-foreground rounded-full border px-3 py-1.5 text-xs font-medium shadow-sm'
              >
                {t(tag)}
              </span>
            ))}
          </div>

          <div className='mt-10 flex items-center justify-center gap-2 sm:mt-16'>
            {HERO_SLIDES.map((slide, index) => (
              <button
                key={slide.title}
                type='button'
                onClick={() => setActiveIndex(index)}
                className={cn(
                  'h-2 rounded-full transition-all',
                  index === activeIndex
                    ? 'w-8 bg-brand'
                    : 'w-2 bg-foreground/20 hover:bg-foreground/35'
                )}
                aria-label={t('Go to slide {{index}}', { index: index + 1 })}
              />
            ))}
          </div>
        </div>
      </div>

      <div className='mx-auto max-w-7xl px-4 py-12 md:px-6'>
        <div className='text-center'>
          <div className='text-muted-foreground text-xs font-bold tracking-[0.24em] uppercase'>
            {t('Authorized Partners')}
          </div>
          <h2 className='mt-3 [font-family:var(--font-playfair-display),Georgia,serif] text-2xl leading-[1.08] font-semibold tracking-normal sm:text-3xl'>
            {t('Official model authorization')}
          </h2>
          <p className='text-muted-foreground mt-2 text-sm'>
            {t('Direct source access, real-time sync, stable and reliable')}
          </p>
        </div>

        <div className='mt-8 grid gap-3 sm:grid-cols-2 lg:grid-cols-6'>
          {PARTNERS.map((partner) => {
            const Logo = partner.logo
            return (
              <div
                key={partner.name}
                className='bg-card/90 rounded-2xl border p-4 text-center shadow-sm transition-all hover:-translate-y-0.5 hover:shadow-md sm:p-5'
              >
                <div className='mx-auto flex h-12 w-full max-w-24 items-center justify-center rounded-xl bg-white/70 px-3 ring-1 ring-black/5 dark:bg-white/10 dark:ring-white/10'>
                  <Logo className='max-h-8 w-auto' aria-hidden />
                </div>
                <div className='mt-4 text-sm font-bold'>{partner.name}</div>
              </div>
            )
          })}
        </div>

        <div className='mt-8 text-center'>
          <div className='text-muted-foreground flex items-center justify-center gap-4 text-xs'>
            <div className='bg-border h-px flex-1' />
            <Link to='/market' className='hover:text-foreground font-medium'>
              {t('More access')}
            </Link>
            <div className='bg-border h-px flex-1' />
          </div>
          <p className='text-muted-foreground mx-auto mt-3 max-w-3xl text-xs leading-6'>
            {t('Moonshot, MiniMax, Kling, Vidu, Grok, Doubao and Qwen')}
          </p>
        </div>
      </div>
    </section>
  )
}
