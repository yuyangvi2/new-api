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
import { useTranslation } from 'react-i18next'

import type { SidebarData } from '@/components/layout/types'
import { ROLE } from '@/lib/roles'
import { SidebarIcons } from '@/components/layout/config/sidebar-icons'

/**
 * Root navigation groups for the application sidebar.
 *
 * These are shown when the URL does not match any nested sidebar view
 * registered in `layout/lib/sidebar-view-registry.ts`.
 */
export function useSidebarData(): SidebarData {
  const { t } = useTranslation()

  return {
    navGroups: [
      {
        id: 'chat',
        title: t('Chat'),
        items: [
          {
            title: t('Playground'),
            url: '/playground',
            icon: SidebarIcons.playground,
          },
          {
            title: t('Experience Center'),
            url: '/image-generator',
            icon: SidebarIcons.experienceCenter,
            activeUrls: ['/image-generator'],
          },
          {
            title: t('Chat'),
            icon: SidebarIcons.chat,
            type: 'chat-presets',
          },
        ],
      },
      {
        id: 'general',
        title: t('General'),
        items: [
          {
            title: t('Overview'),
            url: '/dashboard/overview',
            icon: SidebarIcons.overview,
          },
          {
            title: t('Dashboard'),
            url: '/dashboard/models',
            icon: SidebarIcons.dashboard,
          },
          {
            title: t('API Keys'),
            url: '/keys',
            icon: SidebarIcons.apiKeys,
          },
          {
            title: t('Usage Logs'),
            url: '/usage-logs/common',
            icon: SidebarIcons.usageLogs,
          },
          {
            title: t('Task Logs'),
            url: '/usage-logs/task',
            activeUrls: ['/usage-logs/drawing'],
            configUrls: ['/usage-logs/drawing', '/usage-logs/task'],
            icon: SidebarIcons.taskLogs,
          },
        ],
      },
      {
        id: 'personal',
        title: t('Personal'),
        items: [
          {
            title: t('Wallet'),
            url: '/wallet',
            icon: SidebarIcons.wallet,
          },
          {
            title: t('Profile'),
            url: '/profile',
            icon: SidebarIcons.profile,
          },
        ],
      },
      {
        id: 'admin',
        title: t('Admin'),
        items: [
          {
            title: t('Channels'),
            url: '/channels',
            icon: SidebarIcons.channels,
          },
          {
            title: t('Models'),
            url: '/models/metadata',
            icon: SidebarIcons.models,
          },
          {
            title: t('Users'),
            url: '/users',
            icon: SidebarIcons.users,
          },
          {
            title: t('Redemption Codes'),
            url: '/redemption-codes',
            icon: SidebarIcons.redemptionCodes,
          },
          {
            title: t('Subscriptions'),
            url: '/subscriptions',
            icon: SidebarIcons.subscription,
          },
          {
            title: t('System Info'),
            url: '/system-info',
            icon: SidebarIcons.systemInfo,
            requiredRole: ROLE.SUPER_ADMIN,
          },
          {
            title: t('System Settings'),
            url: '/system-settings/site',
            activeUrls: ['/system-settings'],
            icon: SidebarIcons.systemSettings,
          },
        ],
      },
    ],
  }
}
