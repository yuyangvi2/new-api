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
import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { buildCapEndpoint } from '@/features/auth/lib/bot-protection'
import type { BotProtectionConfig } from '@/features/auth/types'

import { Turnstile } from './turnstile'

declare global {
  interface Window {
    CAP_CUSTOM_WASM_URL?: string
  }
}

type CapElement = HTMLElement & { reset: () => void }

type BotProtectionProps = {
  config: BotProtectionConfig
  onVerify: (token: string) => void
  onExpire?: () => void
  className?: string
}

export function BotProtection(props: BotProtectionProps) {
  if (!props.config.enabled) return null

  if (props.config.provider === 'turnstile') {
    return (
      <Turnstile
        siteKey={props.config.site_key}
        onVerify={props.onVerify}
        onExpire={props.onExpire}
        className={props.className}
      />
    )
  }

  if (props.config.provider === 'cap') {
    return <CapProtection {...props} />
  }

  return null
}

function CapProtection(props: BotProtectionProps) {
  const { t } = useTranslation()
  const containerRef = useRef<HTMLDivElement | null>(null)
  const onVerifyRef = useRef(props.onVerify)
  const onExpireRef = useRef(props.onExpire)
  const [loadFailed, setLoadFailed] = useState(false)
  const publicEndpoint = props.config.public_endpoint.replace(/\/+$/, '')
  const capEndpoint = buildCapEndpoint(props.config)

  useEffect(() => {
    onVerifyRef.current = props.onVerify
    onExpireRef.current = props.onExpire
  }, [props.onExpire, props.onVerify])

  useEffect(() => {
    const container = containerRef.current
    if (!container) return

    window.CAP_CUSTOM_WASM_URL = `${publicEndpoint}/assets/cap_wasm_bg.wasm`
    let element: CapElement | null = null
    let disposed = false

    const handleSolve = (event: Event) => {
      const token = (event as CustomEvent<{ token?: string }>).detail?.token
      if (token) onVerifyRef.current(token)
    }
    const handleReset = () => onExpireRef.current?.()

    setLoadFailed(false)
    void import('cap-widget')
      .then(() => {
        if (disposed) return
        element = document.createElement('cap-widget') as CapElement
        element.setAttribute('data-cap-api-endpoint', capEndpoint)
        element.setAttribute('required', '')
        element.addEventListener('solve', handleSolve)
        element.addEventListener('reset', handleReset)
        element.addEventListener('error', handleReset)
        container.replaceChildren(element)
      })
      .catch(() => {
        if (!disposed) setLoadFailed(true)
      })

    return () => {
      disposed = true
      if (element) {
        element.removeEventListener('solve', handleSolve)
        element.removeEventListener('reset', handleReset)
        element.removeEventListener('error', handleReset)
        element.remove()
      }
    }
  }, [capEndpoint, publicEndpoint])

  if (loadFailed) {
    return (
      <div className={props.className} role='alert'>
        {t('Human verification failed to load. Please refresh and try again.')}
      </div>
    )
  }

  return <div ref={containerRef} className={props.className} />
}
