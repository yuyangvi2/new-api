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
import { useTranslation } from 'react-i18next'

import { Skeleton } from '@/components/ui/skeleton'
import { useSystemConfig } from '@/hooks/use-system-config'

type AuthLayoutProps = {
  children: React.ReactNode
}

export function AuthLayout({ children }: AuthLayoutProps) {
  const { t } = useTranslation()
  const { systemName, logo, loading } = useSystemConfig()
  const currentYear = new Date().getFullYear()

  return (
    <div className='relative flex min-h-svh flex-col items-center justify-center px-4 py-10'>
      <Link
        to='/'
        className='mb-7 flex flex-col items-center gap-2 transition-opacity hover:opacity-80'
      >
        <div className='relative h-10 w-10'>
          {loading ? (
            <Skeleton className='absolute inset-0 rounded-xl' />
          ) : (
            <img
              src={logo}
              alt={t('Logo')}
              className='h-10 w-10 rounded-xl object-cover shadow-sm'
            />
          )}
        </div>
        {loading ? (
          <Skeleton className='h-6 w-24' />
        ) : (
          <div className='text-center'>
            <h1 className='text-xl font-bold tracking-tight'>{systemName}</h1>
            <p className='text-muted-foreground mt-1 text-sm'>
              {t('Intelligent API gateway console')}
            </p>
          </div>
        )}
      </Link>
      <div className='w-full max-w-[440px]'>{children}</div>
      <div className='text-muted-foreground mt-7 text-center text-xs'>
        {t('© {{year}} {{name}}. All rights reserved.', {
          year: currentYear,
          name: systemName || 'Tokone',
        })}
      </div>
    </div>
  )
}
