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
import type { SystemStatus } from '@/features/auth/types'

type StatusLike =
  | SystemStatus
  | {
      data?: Record<string, unknown>
      [key: string]: unknown
    }
  | null
  | undefined

function normalizeServerAddress(value: unknown): string | undefined {
  if (typeof value !== 'string') return undefined

  const trimmed = value.trim()
  if (!trimmed) return undefined

  return trimmed.replace(/\/$/, '')
}

function readServerAddressFromStatus(status: StatusLike): string | undefined {
  if (!status) return undefined

  const record = status as Record<string, unknown>
  const data = record.data as Record<string, unknown> | undefined

  return (
    normalizeServerAddress(record.server_address) ??
    normalizeServerAddress(record.serverAddress) ??
    normalizeServerAddress(record.ServerAddress) ??
    normalizeServerAddress(data?.server_address) ??
    normalizeServerAddress(data?.serverAddress) ??
    normalizeServerAddress(data?.ServerAddress)
  )
}

function readApiBaseAddressFromValue(value: unknown): string | undefined {
  if (!Array.isArray(value)) return undefined

  for (const item of value) {
    if (!item || typeof item !== 'object') continue
    const url = (item as Record<string, unknown>).url
    const normalized = normalizeServerAddress(url)
    if (normalized) return normalized
  }

  return undefined
}

function readApiBaseAddressFromStatus(status: StatusLike): string | undefined {
  if (!status) return undefined

  const record = status as Record<string, unknown>
  const data = record.data as Record<string, unknown> | undefined

  if (record.api_info_enabled === false || data?.api_info_enabled === false) {
    return undefined
  }

  return (
    readApiBaseAddressFromValue(record.api_info) ??
    readApiBaseAddressFromValue(record.apiInfo) ??
    readApiBaseAddressFromValue(data?.api_info) ??
    readApiBaseAddressFromValue(data?.apiInfo)
  )
}

function readStoredStatus(): StatusLike {
  if (typeof window === 'undefined') return undefined

  try {
    const raw = window.localStorage.getItem('status')
    return raw ? (JSON.parse(raw) as StatusLike) : undefined
  } catch {
    return undefined
  }
}

export function getServerAddress(
  status?: StatusLike,
  fallback = ''
): string {
  const fromStatus = readServerAddressFromStatus(status)
  if (fromStatus) return fromStatus

  const fromStorage = readServerAddressFromStatus(readStoredStatus())
  if (fromStorage) return fromStorage

  if (fallback) return normalizeServerAddress(fallback) ?? fallback

  if (typeof window !== 'undefined') return window.location.origin

  return ''
}

export function getApiBaseAddress(
  status?: StatusLike,
  fallback = ''
): string {
  const fromStatus = readApiBaseAddressFromStatus(status)
  if (fromStatus) return fromStatus

  const fromStorage = readApiBaseAddressFromStatus(readStoredStatus())
  if (fromStorage) return fromStorage

  return getServerAddress(status, fallback)
}
