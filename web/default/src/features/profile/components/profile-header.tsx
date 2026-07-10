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
  Activity,
  BarChart3,
  CalendarDays,
  Mail,
  ShieldCheck,
  UserRound,
  Users,
  WalletCards,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { StatusBadge } from '@/components/status-badge'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Card, CardContent } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { getUserAvatarFallback, getUserAvatarStyle } from '@/lib/avatar'
import dayjs from '@/lib/dayjs'
import { formatCompactNumber, formatQuota } from '@/lib/format'
import { getRoleLabel } from '@/lib/roles'
import { cn } from '@/lib/utils'

import { getDisplayName } from '../lib'
import type { UserProfile } from '../types'

// ============================================================================
// Profile Header Component
// ============================================================================

interface ProfileHeaderProps {
  profile: UserProfile | null
  loading: boolean
}

export function ProfileHeader(props: ProfileHeaderProps) {
  const { t } = useTranslation()

  if (props.loading) {
    return (
      <Card
        data-card-hover='false'
        className='gap-0 overflow-hidden rounded-2xl py-0 shadow-sm'
      >
        <CardContent className='p-4 sm:p-5 lg:p-6'>
          <div className='grid gap-5 lg:grid-cols-[minmax(0,1fr)_minmax(34rem,0.95fr)] lg:items-stretch'>
            <div className='flex min-w-0 gap-4'>
              <Skeleton className='h-16 w-16 rounded-2xl' />
              <div className='min-w-0 flex-1 space-y-3'>
                <Skeleton className='h-8 w-48' />
                <Skeleton className='h-4 w-72 max-w-full' />
                <div className='grid gap-2 sm:grid-cols-2'>
                  <Skeleton className='h-11 w-full rounded-xl' />
                  <Skeleton className='h-11 w-full rounded-xl' />
                </div>
              </div>
            </div>
            <div className='grid gap-3 sm:grid-cols-3'>
              {['balance', 'usage', 'requests'].map((item) => (
                <div key={item} className='rounded-xl border p-4'>
                  <Skeleton className='h-4 w-20' />
                  <Skeleton className='mt-4 h-8 w-28' />
                  <Skeleton className='mt-3 h-3.5 w-24' />
                </div>
              ))}
            </div>
          </div>
        </CardContent>
      </Card>
    )
  }

  if (!props.profile) return null

  const profile = props.profile
  const displayName = getDisplayName(profile)
  const avatarName = profile.username || displayName
  const avatarFallback = getUserAvatarFallback(avatarName)
  const avatarFallbackStyle = getUserAvatarStyle(avatarName)
  const roleLabel = getRoleLabel(profile.role)
  const createdAt =
    profile.created_time > 0
      ? dayjs.unix(profile.created_time).format('YYYY-MM-DD')
      : '-'
  const statusLabel = profile.status === 1 ? t('Enabled') : t('Disabled')
  const stats = [
    {
      label: t('Current Balance'),
      value: formatQuota(profile.quota),
      description: t('Remaining quota'),
      icon: WalletCards,
      className: 'text-emerald-600 bg-emerald-500/10',
    },
    {
      label: t('Total Usage'),
      value: formatQuota(profile.used_quota),
      description: t('Total consumed quota'),
      icon: BarChart3,
      className: 'text-blue-600 bg-blue-500/10',
    },
    {
      label: t('API Requests'),
      value: formatCompactNumber(profile.request_count),
      description: t('Total requests made'),
      icon: Activity,
      className: 'text-orange-600 bg-orange-500/10',
    },
  ]
  const details = [
    {
      label: t('Email'),
      value: profile.email || '-',
      icon: Mail,
    },
    {
      label: t('User Group'),
      value: profile.group || '-',
      icon: Users,
    },
    {
      label: t('Created At'),
      value: createdAt,
      icon: CalendarDays,
    },
    {
      label: t('Status'),
      value: statusLabel,
      icon: ShieldCheck,
    },
  ]

  return (
    <Card
      data-card-hover='false'
      className='gap-0 overflow-hidden rounded-2xl py-0 shadow-sm'
    >
      <CardContent className='p-0'>
        <div className='grid gap-0 lg:grid-cols-[minmax(0,1fr)_minmax(34rem,0.95fr)]'>
          <div className='border-border/70 min-w-0 border-b p-4 sm:p-5 lg:border-r lg:border-b-0 lg:p-6'>
            <div className='flex min-w-0 items-start gap-4'>
              <Avatar className='ring-background h-16 w-16 rounded-2xl text-lg ring-4 sm:h-20 sm:w-20 sm:text-2xl'>
                <AvatarFallback
                  className='rounded-2xl font-semibold text-white'
                  style={avatarFallbackStyle}
                >
                  {avatarFallback}
                </AvatarFallback>
              </Avatar>

              <div className='min-w-0 flex-1'>
                <div className='flex min-w-0 flex-wrap items-center gap-2'>
                  <h1 className='min-w-0 truncate text-2xl font-semibold tracking-tight sm:text-3xl'>
                    {displayName}
                  </h1>
                  <StatusBadge
                    label={roleLabel}
                    variant='neutral'
                    copyable={false}
                  />
                  <StatusBadge
                    label={`${t('User ID')} ${profile.id}`}
                    variant='info'
                    copyText={String(profile.id)}
                  />
                </div>

                <div className='text-muted-foreground mt-2 flex min-w-0 items-center gap-2 text-sm'>
                  <UserRound className='size-4 shrink-0' />
                  <span className='truncate'>@{profile.username}</span>
                </div>

                <div className='mt-5 grid gap-2 sm:grid-cols-2'>
                  {details.map((item) => (
                    <div
                      key={item.label}
                      className='bg-muted/35 flex min-w-0 items-center gap-3 rounded-xl border px-3 py-2.5'
                    >
                      <item.icon className='text-muted-foreground size-4 shrink-0' />
                      <div className='min-w-0'>
                        <div className='text-muted-foreground text-[11px] font-medium tracking-wider uppercase'>
                          {item.label}
                        </div>
                        <div className='truncate text-sm font-medium'>
                          {item.value}
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </div>

            <div className='mt-5 grid gap-3 sm:grid-cols-3 lg:hidden'>
              {stats.map((item) => (
                <ProfileStat key={item.label} item={item} />
              ))}
            </div>
          </div>

          <div className='bg-muted/25 hidden p-4 sm:p-5 lg:grid lg:grid-cols-3 lg:gap-3 lg:p-6'>
            {stats.map((item) => (
              <ProfileStat key={item.label} item={item} />
            ))}
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

type ProfileStatItem = {
  label: string
  value: string
  description: string
  icon: typeof Activity
  className: string
}

function ProfileStat(props: { item: ProfileStatItem }) {
  const item = props.item

  return (
    <div className='bg-card min-w-0 rounded-xl border p-3 shadow-sm sm:p-4'>
      <div className='flex items-center justify-between gap-3'>
        <div className='text-muted-foreground min-w-0 truncate text-[11px] font-medium tracking-wider uppercase'>
          {item.label}
        </div>
        <div
          className={cn(
            'flex size-8 shrink-0 items-center justify-center rounded-lg',
            item.className
          )}
        >
          <item.icon className='size-4' />
        </div>
      </div>

      <div className='text-foreground mt-4 truncate font-mono text-2xl font-bold tracking-tight tabular-nums'>
        {item.value}
      </div>
      <div className='text-muted-foreground/70 mt-1 truncate text-xs'>
        {item.description}
      </div>
    </div>
  )
}
