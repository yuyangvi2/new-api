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
import { useQuery } from '@tanstack/react-query'
import { getRouteApi } from '@tanstack/react-router'
import { Gauge, GaugeCircle, WalletCards } from 'lucide-react'
import type { ComponentType } from 'react'
import { useTranslation } from 'react-i18next'

import { Skeleton } from '@/components/ui/skeleton'
import { useIsAdmin } from '@/hooks/use-admin'
import { formatLogQuota } from '@/lib/format'
import { cn } from '@/lib/utils'

import { getLogStats, getUserLogStats } from '../api'
import { DEFAULT_LOG_STATS } from '../constants'
import { buildApiParams } from '../lib/utils'
import { useUsageLogsContext } from './usage-logs-provider'

const route = getRouteApi('/_authenticated/usage-logs/$section')

function StatBadge(props: {
  label: string
  value: string | number
  accent: string
  icon: ComponentType<{ className?: string }>
}) {
  const Icon = props.icon

  return (
    <div className='border-border/70 bg-background/70 flex min-h-20 min-w-0 items-center gap-3 rounded-xl border px-4 py-3 shadow-xs dark:bg-muted/20'>
      <div
        className={cn(
          'flex size-10 shrink-0 items-center justify-center rounded-xl text-white shadow-sm',
          props.accent
        )}
      >
        <Icon className='size-4.5' />
      </div>
      <div className='min-w-0'>
        <p className='text-muted-foreground text-xs font-medium'>
          {props.label}
        </p>
        <p className='text-foreground mt-1 truncate font-mono text-xl font-bold tabular-nums'>
          {props.value}
        </p>
      </div>
    </div>
  )
}

export function CommonLogsStats() {
  const { t } = useTranslation()
  const isAdmin = useIsAdmin()
  const searchParams = route.useSearch()
  const { sensitiveVisible } = useUsageLogsContext()

  const { data: stats, isLoading } = useQuery({
    queryKey: ['usage-logs-stats', isAdmin, searchParams],
    queryFn: async () => {
      const params = buildApiParams({
        page: 1,
        pageSize: 1,
        searchParams,
        columnFilters: [],
        isAdmin,
      })

      const result = isAdmin
        ? await getLogStats(params)
        : await getUserLogStats(params)

      return result.success
        ? result.data || DEFAULT_LOG_STATS
        : DEFAULT_LOG_STATS
    },
    placeholderData: (previousData) => previousData,
  })

  if (isLoading) {
    return (
      <div className='grid gap-3 sm:grid-cols-3'>
        <Skeleton className='h-20 rounded-xl' />
        <Skeleton className='h-20 rounded-xl' />
        <Skeleton className='h-20 rounded-xl' />
      </div>
    )
  }

  return (
    <div className='grid gap-3 sm:grid-cols-3'>
      <StatBadge
        label={t('Usage')}
        value={sensitiveVisible ? formatLogQuota(stats?.quota || 0) : '••••'}
        accent='bg-sky-600'
        icon={WalletCards}
      />
      <StatBadge
        label={t('RPM')}
        value={stats?.rpm || 0}
        accent='bg-rose-600'
        icon={Gauge}
      />
      <StatBadge
        label={t('TPM')}
        value={stats?.tpm || 0}
        accent='bg-teal-700'
        icon={GaugeCircle}
      />
    </div>
  )
}
