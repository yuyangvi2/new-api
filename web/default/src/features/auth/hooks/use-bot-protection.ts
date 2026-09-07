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
import i18next from 'i18next'
import { useCallback, useMemo, useState } from 'react'
import { toast } from 'sonner'

import { useStatus } from '@/hooks/use-status'

import { resolveBotProtectionConfig } from '../lib/bot-protection'
import type { BotProtectionScene } from '../types'

export function useBotProtection(scene: BotProtectionScene) {
  const { status } = useStatus()
  const [token, setToken] = useState('')
  const [widgetKey, setWidgetKey] = useState(0)
  const config = useMemo(
    () => resolveBotProtectionConfig(status, scene),
    [scene, status]
  )

  const validate = useCallback((): boolean => {
    if (config.enabled && !token) {
      toast.info(
        i18next.t('Please wait a moment, human check is initializing...')
      )
      return false
    }
    return true
  }, [config.enabled, token])

  const reset = useCallback(() => {
    setToken('')
    setWidgetKey((value) => value + 1)
  }, [])

  return {
    config,
    token,
    setToken,
    validate,
    reset,
    widgetKey,
  }
}
