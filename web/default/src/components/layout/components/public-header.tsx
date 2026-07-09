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
import { Link, useNavigate, useRouterState } from '@tanstack/react-router'
import { ArrowRight, MessageCircle, Monitor } from 'lucide-react'
import {
  useCallback,
  useEffect,
  useState,
  type MouseEvent,
  type ReactNode,
} from 'react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { LanguageSwitcher } from '@/components/language-switcher'
import { NotificationPopover } from '@/components/notification-popover'
import { ProfileDropdown } from '@/components/profile-dropdown'
import { ThemeSwitch } from '@/components/theme-switch'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { useNotifications } from '@/hooks/use-notifications'
import { useSystemConfig } from '@/hooks/use-system-config'
import { useTopNavLinks } from '@/hooks/use-top-nav-links'
import { cn } from '@/lib/utils'
import { useAuthStore } from '@/stores/auth-store'

import { defaultTopNavLinks } from '../config/top-nav.config'
import type { TopNavLink } from '../types'
import { HeaderLogo } from './header-logo'

const AUTH_PROMPT_SECONDS = 5

type AuthPromptTarget = {
  title: string
  href: string
}

export interface PublicHeaderProps {
  navLinks?: TopNavLink[]
  mobileLinks?: TopNavLink[]
  navContent?: ReactNode
  showThemeSwitch?: boolean
  showLanguageSwitcher?: boolean
  logo?: ReactNode
  siteName?: string
  homeUrl?: string
  leftContent?: ReactNode
  rightContent?: ReactNode
  showNavigation?: boolean
  showAuthButtons?: boolean
  showNotifications?: boolean
  showContactButton?: boolean
  className?: string
}

export function PublicHeader(props: PublicHeaderProps) {
  const {
    navLinks = defaultTopNavLinks,
    showThemeSwitch = true,
    showLanguageSwitcher = true,
    logo: customLogo,
    siteName: customSiteName,
    homeUrl = '/',
    showAuthButtons = true,
    showNotifications = true,
    showContactButton = true,
  } = props

  const { t } = useTranslation()
  const navigate = useNavigate()
  const [scrolled, setScrolled] = useState(false)
  const [mobileOpen, setMobileOpen] = useState(false)
  const [authPromptTarget, setAuthPromptTarget] =
    useState<AuthPromptTarget | null>(null)
  const [authPromptSecondsLeft, setAuthPromptSecondsLeft] =
    useState(AUTH_PROMPT_SECONDS)
  const { auth } = useAuthStore()
  const {
    systemName,
    logo: systemLogo,
    loading,
    logoLoaded,
  } = useSystemConfig()
  const dynamicLinks = useTopNavLinks()
  const notifications = useNotifications()
  const routerState = useRouterState()
  const pathname = routerState.location.pathname

  const user = auth.user
  const isAuthenticated = !!user
  const displaySiteName = customSiteName || systemName
  const links = dynamicLinks.length > 0 ? dynamicLinks : navLinks

  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 20)
    onScroll()
    window.addEventListener('scroll', onScroll, { passive: true })
    return () => window.removeEventListener('scroll', onScroll)
  }, [])

  useEffect(() => {
    document.body.style.overflow = mobileOpen ? 'hidden' : ''
    return () => {
      document.body.style.overflow = ''
    }
  }, [mobileOpen])

  useEffect(() => {
    if (!authPromptTarget) return

    const intervalId = window.setInterval(() => {
      setAuthPromptSecondsLeft((seconds) => Math.max(seconds - 1, 0))
    }, 1000)

    const timeoutId = window.setTimeout(() => {
      const redirect = authPromptTarget.href
      setAuthPromptTarget(null)
      navigate({ to: '/sign-in', search: { redirect } })
    }, AUTH_PROMPT_SECONDS * 1000)

    return () => {
      window.clearInterval(intervalId)
      window.clearTimeout(timeoutId)
    }
  }, [authPromptTarget, navigate])

  const closeAuthPrompt = useCallback(() => {
    setAuthPromptTarget(null)
    setAuthPromptSecondsLeft(AUTH_PROMPT_SECONDS)
  }, [])

  const navigateToSignIn = useCallback(() => {
    const redirect = authPromptTarget?.href || '/'
    setAuthPromptTarget(null)
    navigate({ to: '/sign-in', search: { redirect } })
  }, [authPromptTarget?.href, navigate])

  const handleNavLinkClick = useCallback(
    (
      event: MouseEvent<HTMLAnchorElement>,
      link: TopNavLink,
      closeMobile = false
    ) => {
      if (link.disabled) {
        event.preventDefault()
        return
      }

      if (link.requiresAuth) {
        event.preventDefault()
        if (closeMobile) {
          setMobileOpen(false)
        }
        setAuthPromptSecondsLeft(AUTH_PROMPT_SECONDS)
        setAuthPromptTarget({
          title: t(link.title),
          href: link.href,
        })
        return
      }

      if (closeMobile) {
        setMobileOpen(false)
      }
    },
    [t]
  )

  let brandLogo: ReactNode = customLogo
  if (loading) {
    brandLogo = <Skeleton className='size-full rounded-full' />
  } else if (!brandLogo) {
    brandLogo = (
      <HeaderLogo
        src={systemLogo}
        loading={loading}
        logoLoaded={logoLoaded}
        className='size-full rounded-full object-contain'
      />
    )
  }

  let authControls: ReactNode = null
  if (showAuthButtons) {
    if (loading) {
      authControls = <Skeleton className='h-8 w-28 rounded-full' />
    } else if (isAuthenticated) {
      authControls = (
        <>
          <Button
            variant='ghost'
            size='sm'
            className='hover:bg-background/70 h-9 gap-1.5 rounded-full px-3 text-xs'
            render={<Link to='/dashboard' />}
          >
            <Monitor className='size-3.5' />
            {t('Console')}
          </Button>
          <ProfileDropdown />
        </>
      )
    } else {
      authControls = (
        <>
          <Link
            to='/sign-in'
            className='text-muted-foreground hover:text-foreground text-sm font-medium transition-colors'
          >
            {t('Sign in')}
          </Link>
          <Button
            size='sm'
            className='group bg-foreground text-background hover:bg-foreground/90 relative h-9 overflow-hidden rounded-full px-5 text-xs font-semibold'
            render={<Link to='/sign-up' />}
          >
            <span className='absolute inset-0 -translate-x-full bg-[linear-gradient(110deg,transparent_0%,rgb(255_255_255/0.22)_45%,transparent_100%)] transition-transform duration-700 group-hover:translate-x-full' />
            <span className='relative flex items-center gap-1.5'>
              {t('Get Started')}
              <ArrowRight className='size-3.5 transition-transform duration-200 group-hover:translate-x-0.5' />
            </span>
          </Button>
        </>
      )
    }
  }

  return (
    <>
      <header
        className={cn(
          'fixed inset-x-0 top-0 z-50 overflow-hidden transition-all duration-500 ease-[cubic-bezier(0.16,1,0.3,1)]',
          scrolled
            ? 'border-b border-white/55 bg-[#fbf7ef]/68 shadow-[0_12px_36px_rgb(15_23_42/0.08),inset_0_1px_0_rgb(255_255_255/0.58)] backdrop-blur-2xl supports-[backdrop-filter]:bg-[#fbf7ef]/62 dark:border-white/10 dark:bg-background/70 dark:shadow-[0_12px_36px_rgb(0_0_0/0.24),inset_0_1px_0_rgb(255_255_255/0.08)] dark:supports-[backdrop-filter]:bg-background/58'
            : 'border-b border-transparent bg-transparent'
        )}
      >
        <div
          className={cn(
            'relative mx-auto max-w-7xl px-4 transition-all duration-500 ease-[cubic-bezier(0.16,1,0.3,1)] md:px-6',
            scrolled ? 'h-[3.75rem]' : 'h-[4.5rem]'
          )}
        >
          <nav className='flex h-full items-center justify-between gap-4 transition-all duration-500 ease-[cubic-bezier(0.16,1,0.3,1)]'>
            <div className='flex min-w-0 items-center gap-6 lg:gap-8'>
              {/* Logo */}
              <Link
                to={homeUrl}
                className='group flex shrink-0 items-center gap-2.5'
              >
                <div className='flex size-9 shrink-0 items-center justify-center rounded-xl shadow-[0_10px_26px_rgb(234_117_20/0.18)] transition-all duration-300 group-hover:scale-105 group-hover:shadow-[0_14px_34px_rgb(234_117_20/0.24)]'>
                  {brandLogo}
                </div>
                <span className='max-w-[10rem] truncate text-lg font-semibold tracking-tight sm:max-w-none sm:text-xl md:text-[1.6rem]'>
                  {loading ? (
                    <Skeleton className='h-4 w-16' />
                  ) : (
                    displaySiteName
                  )}
                </span>
              </Link>

              {/* Desktop nav */}
              <div className='hidden items-center gap-6 lg:flex'>
                {links.map((link) => {
                  const isActive = pathname === link.href
                  const key = `${link.href}:${link.title}`
                  const linkClassName = cn(
                    'group/link relative flex h-[3.75rem] items-center overflow-visible px-0.5 text-sm font-medium transition-colors duration-300',
                    isActive
                      ? 'text-foreground'
                      : 'text-muted-foreground hover:text-foreground',
                    link.disabled && 'pointer-events-none opacity-50'
                  )
                  const linkContent = (
                    <>
                      <span className='relative z-10 transition-transform duration-300 group-hover/link:-translate-y-px'>
                        {t(link.title)}
                      </span>
                      <span className='absolute inset-x-[-0.8rem] top-1/2 h-8 -translate-y-1/2 rounded-full bg-orange-200/0 blur-xl transition-colors duration-300 group-hover/link:bg-orange-200/28' />
                      <span
                        className={cn(
                          'absolute bottom-3 left-0 h-[2px] rounded-full bg-gradient-to-r from-orange-500 via-amber-400 to-foreground transition-all duration-300',
                          isActive
                            ? 'w-full opacity-100'
                            : 'w-0 opacity-0 group-hover/link:w-full group-hover/link:opacity-100'
                        )}
                      />
                    </>
                  )
                  if (link.external) {
                    return (
                      <a
                        key={key}
                        href={link.href}
                        target='_blank'
                        rel='noopener noreferrer'
                        aria-disabled={link.disabled}
                        tabIndex={link.disabled ? -1 : undefined}
                        onClick={(event) => handleNavLinkClick(event, link)}
                        className={linkClassName}
                      >
                        {linkContent}
                      </a>
                    )
                  }
                  return (
                    <Link
                      key={key}
                      to={link.href}
                      disabled={link.disabled}
                      onClick={(event) => handleNavLinkClick(event, link)}
                      className={linkClassName}
                    >
                      {linkContent}
                    </Link>
                  )
                })}
              </div>
            </div>

            <div className='hidden items-center gap-3 lg:flex'>
              {showContactButton && (
                <Button
                  variant='outline'
                  size='sm'
                  className='border-border/70 bg-card/78 hover:bg-background h-9 gap-1.5 rounded-full px-3 text-xs shadow-sm backdrop-blur transition-all hover:-translate-y-0.5 hover:border-orange-200'
                  render={<Link to='/about' />}
                >
                  <MessageCircle className='size-3.5' />
                  {t('Contact us')}
                </Button>
              )}
              {(showLanguageSwitcher ||
                showThemeSwitch ||
                showNotifications) && <div className='bg-border/50 h-4 w-px' />}

              {showLanguageSwitcher && <LanguageSwitcher />}
              {showThemeSwitch && <ThemeSwitch />}
              {showNotifications && (
                <NotificationPopover
                  open={notifications.popoverOpen}
                  onOpenChange={notifications.setPopoverOpen}
                  unreadCount={notifications.unreadCount}
                  activeTab={notifications.activeTab}
                  onTabChange={notifications.setActiveTab}
                  notice={notifications.notice}
                  announcements={notifications.announcements}
                  loading={notifications.loading}
                />
              )}

              {authControls}
            </div>

            {/* Mobile: compact actions + hamburger */}
            <div className='flex items-center gap-2 lg:hidden'>
              {showAuthButtons && !loading && isAuthenticated && (
                <ProfileDropdown />
              )}
              <Button
                type='button'
                variant='ghost'
                size='icon'
                className='border-border/50 bg-background/58 size-10 rounded-full border shadow-sm backdrop-blur'
                onClick={() => setMobileOpen((v) => !v)}
                aria-label={t('Toggle navigation menu')}
              >
                <div className='relative size-4'>
                  <span
                    className={cn(
                      'absolute inset-x-0 block h-[1.5px] origin-center rounded-full bg-current transition-all duration-300',
                      mobileOpen ? 'top-[7px] rotate-45' : 'top-[3px]'
                    )}
                  />
                  <span
                    className={cn(
                      'absolute inset-x-0 top-[7px] block h-[1.5px] rounded-full bg-current transition-all duration-300',
                      mobileOpen ? 'scale-x-0 opacity-0' : 'opacity-100'
                    )}
                  />
                  <span
                    className={cn(
                      'absolute inset-x-0 block h-[1.5px] origin-center rounded-full bg-current transition-all duration-300',
                      mobileOpen ? 'top-[7px] -rotate-45' : 'top-[11px]'
                    )}
                  />
                </div>
              </Button>
            </div>
          </nav>
        </div>
      </header>

      {/* Mobile full-screen overlay */}
      <div
        className={cn(
          'fixed inset-0 z-40 overflow-hidden bg-[#fbf7ef]/94 backdrop-blur-2xl transition-all duration-500 ease-[cubic-bezier(0.16,1,0.3,1)] lg:pointer-events-none lg:hidden dark:bg-background/96',
          mobileOpen
            ? 'pointer-events-auto opacity-100'
            : 'pointer-events-none opacity-0'
        )}
      >
        <div className='pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_50%_14%,rgb(255_171_45/0.22),transparent_34%),linear-gradient(180deg,rgb(255_255_255/0.26),transparent_56%)]' />
        <div className='relative flex h-full flex-col justify-between px-4 pt-24 pb-7 sm:px-5'>
          <nav className='flex flex-col gap-2'>
            {links.map((link, i) => {
              const isActive = pathname === link.href
              const key = `${link.href}:${link.title}`
              const linkClassName = cn(
                'flex min-h-[3.25rem] items-center justify-between rounded-2xl border px-4 text-base font-semibold tracking-tight shadow-sm transition-all duration-500 ease-[cubic-bezier(0.16,1,0.3,1)]',
                mobileOpen
                  ? 'translate-y-0 opacity-100'
                  : 'translate-y-4 opacity-0',
                isActive
                  ? 'border-foreground bg-foreground text-background'
                  : 'border-orange-100/80 bg-background/72 text-foreground',
                link.disabled && 'pointer-events-none opacity-50'
              )
              const transitionStyle = {
                transitionDelay: mobileOpen ? `${100 + i * 50}ms` : '0ms',
              }
              if (link.external) {
                return (
                  <a
                    key={key}
                    href={link.href}
                    target='_blank'
                    rel='noopener noreferrer'
                    aria-disabled={link.disabled}
                    tabIndex={link.disabled ? -1 : undefined}
                    onClick={(event) => handleNavLinkClick(event, link, true)}
                    className={linkClassName}
                    style={transitionStyle}
                  >
                    <span>{t(link.title)}</span>
                    <ArrowRight className='size-4 opacity-60' />
                  </a>
                )
              }
              return (
                <Link
                  key={key}
                  to={link.href}
                  disabled={link.disabled}
                  onClick={(event) => handleNavLinkClick(event, link, true)}
                  className={linkClassName}
                  style={transitionStyle}
                >
                  <span>{t(link.title)}</span>
                  <ArrowRight className='size-4 opacity-60' />
                </Link>
              )
            })}
          </nav>

          <div
            className={cn(
              'flex flex-col gap-3 transition-all duration-500',
              mobileOpen
                ? 'translate-y-0 opacity-100'
                : 'translate-y-4 opacity-0'
            )}
            style={{ transitionDelay: mobileOpen ? '250ms' : '0ms' }}
          >
            <div className='bg-background/70 flex items-center justify-center gap-3 rounded-2xl border border-orange-100/80 p-3 shadow-sm'>
              {showLanguageSwitcher && <LanguageSwitcher />}
              {showThemeSwitch && <ThemeSwitch />}
              {showNotifications && (
                <NotificationPopover
                  open={notifications.popoverOpen}
                  onOpenChange={notifications.setPopoverOpen}
                  unreadCount={notifications.unreadCount}
                  activeTab={notifications.activeTab}
                  onTabChange={notifications.setActiveTab}
                  notice={notifications.notice}
                  announcements={notifications.announcements}
                  loading={notifications.loading}
                />
              )}
            </div>
            {showAuthButtons && (
              <div className='grid grid-cols-1 gap-3 sm:grid-cols-2'>
                <Link
                  to={isAuthenticated ? '/dashboard' : '/sign-in'}
                  onClick={() => setMobileOpen(false)}
                  className='border-border bg-background/78 inline-flex h-12 items-center justify-center rounded-full border text-sm font-semibold shadow-sm transition-opacity hover:opacity-90 active:opacity-80'
                >
                  {isAuthenticated ? t('Go to Dashboard') : t('Sign in')}
                </Link>
                <Link
                  to={isAuthenticated ? '/dashboard' : '/sign-up'}
                  onClick={() => setMobileOpen(false)}
                  className='bg-foreground text-background inline-flex h-12 items-center justify-center rounded-full text-sm font-semibold transition-opacity hover:opacity-90 active:opacity-80'
                >
                  {isAuthenticated ? t('Console') : t('Get Started')}
                </Link>
              </div>
            )}
          </div>
        </div>
      </div>

      <Dialog
        open={!!authPromptTarget}
        onOpenChange={(open) => {
          if (!open) {
            closeAuthPrompt()
          }
        }}
        title={t('Sign in required')}
        description={t('Please sign in to view {{module}}.', {
          module: authPromptTarget?.title || '',
        })}
        contentClassName='sm:max-w-md'
        contentHeight='auto'
        footer={
          <>
            <Button variant='outline' onClick={closeAuthPrompt}>
              {t('Cancel')}
            </Button>
            <Button onClick={navigateToSignIn}>{t('Sign in now')}</Button>
          </>
        }
      >
        <div className='bg-muted/40 text-muted-foreground rounded-lg px-3 py-2 text-sm'>
          {t('Redirecting to sign in in {{seconds}} seconds.', {
            seconds: authPromptSecondsLeft,
          })}
        </div>
      </Dialog>
    </>
  )
}
