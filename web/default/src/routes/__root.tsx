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
import type { QueryClient } from '@tanstack/react-query'
import { ReactQueryDevtools } from '@tanstack/react-query-devtools'
import { createRootRouteWithContext, Outlet } from '@tanstack/react-router'
import { TanStackRouterDevtools } from '@tanstack/react-router-devtools'
import { useEffect } from 'react'

import { NavigationProgress } from '@/components/navigation-progress'
import { Toaster } from '@/components/ui/sonner'
import { ThemeCustomizationProvider } from '@/context/theme-customization-provider'
import { saveAffiliateCode } from '@/features/auth/lib/storage'
import { GeneralError } from '@/features/errors/general-error'
import { NotFoundError } from '@/features/errors/not-found-error'
import { getSetupStatus } from '@/features/setup/api'
import { useSystemConfig } from '@/hooks/use-system-config'

function RootComponent() {
  // Load system configuration (logo, system name, etc.) from backend
  useSystemConfig({ autoLoad: true })

  useEffect(() => {
    startSetupStatusCheck()

    const aff = new URLSearchParams(window.location.search).get('aff')?.trim()
    if (aff) {
      saveAffiliateCode(aff)
    }
  }, [])

  return (
    <ThemeCustomizationProvider>
      <NavigationProgress />
      <Outlet />
      <Toaster closeButton duration={5000} position='top-center' richColors />
      {import.meta.env.MODE === 'development' && (
        <>
          <ReactQueryDevtools buttonPosition='bottom-left' />
          <TanStackRouterDevtools position='bottom-right' />
        </>
      )}
    </ThemeCustomizationProvider>
  )
}

// 缓存 setup 状态检查结果，避免每次导航都重复调用 API
// 使用 localStorage 持久化，避免页面刷新后重复检查
const SETUP_CHECKED_KEY = 'setup_status_checked'

function getSetupStatusFromCache(): boolean {
  try {
    if (typeof window !== 'undefined') {
      return window.localStorage.getItem(SETUP_CHECKED_KEY) === 'true'
    }
  } catch {
    /* empty */
  }
  return false
}

function setSetupStatusCache(value: boolean): void {
  try {
    if (typeof window !== 'undefined') {
      if (value) {
        window.localStorage.setItem(SETUP_CHECKED_KEY, 'true')
      } else {
        window.localStorage.removeItem(SETUP_CHECKED_KEY)
      }
    }
  } catch {
    /* empty */
  }
}

// 内存中的标记，避免同一会话中重复检查
let setupStatusChecked = getSetupStatusFromCache()
let setupStatusCheckPromise: Promise<void> | null = null

function getSetupStatusWithTimeout(timeoutMs = 1200) {
  return Promise.race([
    getSetupStatus(),
    new Promise<null>((resolve) => {
      window.setTimeout(() => resolve(null), timeoutMs)
    }),
  ])
}

function startSetupStatusCheck() {
  if (
    setupStatusChecked ||
    setupStatusCheckPromise ||
    typeof window === 'undefined' ||
    window.location.pathname.startsWith('/setup')
  ) {
    return
  }

  setupStatusCheckPromise = getSetupStatusWithTimeout()
    .then((status) => {
      if (!status?.success || !status.data) return

      setupStatusChecked = true
      setSetupStatusCache(true)

      if (
        !status.data.status &&
        !window.location.pathname.startsWith('/setup')
      ) {
        window.location.assign('/setup')
      }
    })
    .catch((error: unknown) => {
      if (import.meta.env.DEV) {
        // eslint-disable-next-line no-console
        console.warn('[root] setup status check failed', error)
      }
    })
    .finally(() => {
      setupStatusCheckPromise = null
    })
}

export const Route = createRootRouteWithContext<{
  queryClient: QueryClient
}>()({
  // 应用初始化与路由解析前统一校验会话
  beforeLoad: ({ location }) => {
    const pathname = location?.pathname || ''

    // 用户信息已通过 auth-store 从 localStorage 恢复
    // 如果 auth.user 存在，说明用户已登录（有缓存的用户数据）
    // 如果 auth.user 为 null，说明用户未登录，直接让 _authenticated 路由处理重定向
    // 不再调用 getSelf() API，避免不必要的网络请求和等待

    // Setup 检查不阻塞首屏；真实未初始化时后台检查会跳转到 /setup。
    if (!pathname.startsWith('/setup')) {
      startSetupStatusCheck()
    }
    // 用户认证状态完全依赖 localStorage 缓存
    // 如果用户有有效 session 但 localStorage 被清空，会被重定向到登录页重新登录
  },
  component: RootComponent,
  notFoundComponent: NotFoundError,
  errorComponent: GeneralError,
})
