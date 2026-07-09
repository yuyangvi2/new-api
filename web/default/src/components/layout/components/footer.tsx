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
import { Fragment, useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { useStatus } from '@/hooks/use-status'
import { useSystemConfig } from '@/hooks/use-system-config'
import { cn } from '@/lib/utils'

interface FooterLink {
  text: string
  href: string
}

interface FooterColumnProps {
  title: string
  links: FooterLink[]
}

interface FooterProps {
  logo?: string
  name?: string
  columns?: FooterColumnProps[]
  copyright?: string
  className?: string
}

function FooterLinkItem(props: { link: FooterLink }) {
  const { t } = useTranslation()
  const isExternal = props.link.href.startsWith('http')
  const label = t(props.link.text)

  if (isExternal) {
    return (
      <a
        href={props.link.href}
        target='_blank'
        rel='noopener noreferrer'
        className='text-muted-foreground hover:text-foreground text-sm leading-7 transition-colors duration-200'
      >
        {label}
      </a>
    )
  }

  return (
    <Link
      to={props.link.href}
      className='text-muted-foreground hover:text-foreground text-sm leading-7 transition-colors duration-200'
    >
      {label}
    </Link>
  )
}

function LegalLinks(props: { leadingSeparator?: boolean }) {
  const { t } = useTranslation()
  const { status } = useStatus()
  const items: { key: string; label: string; href: string }[] = []
  if (status?.user_agreement_enabled) {
    items.push({
      key: 'user-agreement',
      label: t('User Agreement'),
      href: '/user-agreement',
    })
  }
  if (status?.privacy_policy_enabled) {
    items.push({
      key: 'privacy-policy',
      label: t('Privacy Policy'),
      href: '/privacy-policy',
    })
  }
  if (items.length === 0) {
    return null
  }
  return (
    <>
      {items.map((item, index) => (
        <Fragment key={item.key}>
          {(props.leadingSeparator || index > 0) && (
            <span aria-hidden='true' className='text-muted-foreground/30'>
              &middot;
            </span>
          )}
          <Link
            to={item.href}
            className='hover:text-foreground transition-colors duration-200'
          >
            {item.label}
          </Link>
        </Fragment>
      ))}
    </>
  )
}

function FooterHtmlLegalLinks() {
  const { status } = useStatus()
  if (!status?.user_agreement_enabled && !status?.privacy_policy_enabled) {
    return null
  }
  return (
    <div className='border-border/60 text-muted-foreground/45 flex w-full flex-wrap items-center justify-center gap-x-3 gap-y-1 border-t pt-4 text-xs sm:w-auto sm:justify-end sm:border-t-0 sm:border-l sm:pt-0 sm:pl-5'>
      <LegalLinks />
    </div>
  )
}

export function Footer(props: FooterProps) {
  const { t } = useTranslation()
  const { status } = useStatus()
  const {
    systemName,
    logo: systemLogo,
    footerHtml,
    demoSiteEnabled,
  } = useSystemConfig()

  const displayLogo = systemLogo || props.logo || '/logo.png'
  const displayName = systemName || props.name || 'Tokone'
  const currentYear = new Date().getFullYear()
  const docsUrl =
    (status?.docs_link as string | undefined) || 'https://docs.newapi.pro'
  const docsBase = docsUrl.endsWith('/') ? docsUrl.slice(0, -1) : docsUrl

  const fallbackColumns = useMemo<FooterColumnProps[]>(
    () => [
      {
        title: 'Product',
        links: [
          {
            text: 'Model services',
            href: '/market',
          },
          {
            text: 'API Reference',
            href: docsUrl,
          },
          {
            text: 'Pricing',
            href: '/pricing',
          },
        ],
      },
      {
        title: 'Resources',
        links: [
          {
            text: 'Quick Start',
            href: `${docsBase}/getting-started/`,
          },
          {
            text: 'Documentation',
            href: docsUrl,
          },
          {
            text: 'Help Center',
            href: `${docsBase}/support/faq/`,
          },
        ],
      },
      {
        title: 'Legal',
        links: [
          {
            text: 'User Agreement',
            href: '/user-agreement',
          },
          {
            text: 'Privacy Policy',
            href: '/privacy-policy',
          },
          {
            text: 'Contact us',
            href: '/about',
          },
        ],
      },
    ],
    [docsBase, docsUrl]
  )

  const displayColumns = props.columns ?? fallbackColumns

  if (footerHtml) {
    return (
      <footer
        className={cn(
          'relative z-10 border-t border-orange-100/70 bg-[#fbf7ef] dark:border-border/40 dark:bg-background',
          props.className
        )}
      >
        <div className='mx-auto w-full max-w-6xl px-6 py-5'>
          <div className='bg-background/80 dark:border-border/50 dark:bg-card/80 flex flex-col items-center justify-between gap-4 rounded-2xl border border-orange-100/70 px-4 py-4 shadow-sm backdrop-blur-sm sm:flex-row sm:px-5'>
            <div
              className='custom-footer text-muted-foreground min-w-0 text-center text-sm sm:text-left'
              dangerouslySetInnerHTML={{ __html: footerHtml }}
            />
            <FooterHtmlLegalLinks />
          </div>
        </div>
      </footer>
    )
  }

  return (
    <footer
      className={cn(
        'relative z-10 border-t border-orange-100/70 bg-[#fbf7ef] dark:border-border/40 dark:bg-background',
        props.className
      )}
    >
      <div className='mx-auto max-w-7xl px-4 py-16 md:px-6 md:py-20'>
        <div className='dark:border-border/50 grid gap-12 border-b border-orange-100/80 pb-12 md:grid-cols-[minmax(260px,1.35fr)_minmax(0,2fr)] md:items-start'>
          <div className='max-w-sm'>
            <Link to='/' className='group flex items-center gap-2.5'>
              <img
                src={displayLogo}
                alt={displayName}
                className='size-8 rounded-lg object-contain shadow-sm'
              />
              <span className='text-base font-bold tracking-tight'>
                {displayName}
              </span>
            </Link>
            <p className='text-muted-foreground mt-6 max-w-xs text-sm leading-7'>
              {t('Built for unified AI model access, billing, and operations.')}
            </p>
            {demoSiteEnabled && (
              <div className='text-brand mt-5 text-xs font-semibold tracking-wide'>
                {t('Demo site')}
              </div>
            )}
          </div>

          <div className='grid gap-10 sm:grid-cols-3 md:justify-self-end md:pl-8 lg:min-w-[620px] lg:pl-16'>
            {displayColumns.map((column) => (
              <div key={column.title}>
                <p className='text-foreground mb-5 text-xs font-semibold tracking-[0.18em] uppercase'>
                  {t(column.title)}
                </p>
                <ul className='space-y-3.5'>
                  {column.links.map((link) => (
                    <li key={`${link.text}:${link.href}`}>
                      <FooterLinkItem link={link} />
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </div>
        </div>

        <div className='text-muted-foreground/65 flex items-center justify-center pt-8 text-center text-xs'>
          <span>
            &copy; {currentYear} {displayName}.{' '}
            {props.copyright ?? t('footer.defaultCopyright')}
          </span>
        </div>
      </div>
    </footer>
  )
}
