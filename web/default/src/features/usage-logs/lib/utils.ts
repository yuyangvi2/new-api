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
/**
 * Utility functions for usage logs feature
 */
import {
  getAllLogs,
  getUserLogs,
  getAllTaskLogs,
  getUserTaskLogs,
} from '../api'
import {
  LOG_TYPES,
  DISPLAYABLE_LOG_TYPES,
  TIMING_LOG_TYPES,
} from '../constants'
import type {
  GetLogsParams,
  GetLogsResponse,
  FetchLogsConfig,
  GetTaskLogsParams,
} from '../types'
import { buildTimeRangeParams } from './time-range'

export { getDefaultTimeRange } from './time-range'

// ============================================================================
// Type Checkers & Utilities
// ============================================================================

/**
 * Check if log type is displayable (has detailed info)
 */
export function isDisplayableLogType(type: number): boolean {
  return (DISPLAYABLE_LOG_TYPES as readonly number[]).includes(type)
}

/**
 * Check if log type shows timing info
 */
export function isTimingLogType(type: number): boolean {
  return (TIMING_LOG_TYPES as readonly number[]).includes(type)
}

/**
 * Get log type configuration by type number
 */
export function getLogTypeConfig(type: number) {
  return LOG_TYPES.find((t) => t.value === type) || LOG_TYPES[0]
}

/**
 * Check if log uses per-call billing
 */
export function isPerCallBilling(modelPrice?: number): boolean {
  return (modelPrice ?? 0) > 0
}

/**
 * Build base parameters with time range (for drawing and task logs)
 * Drawing logs use millisecond timestamps and default to the last 30 days.
 * Other task logs use second timestamps and default to today.
 */
export function buildBaseParams(config: {
  page: number
  pageSize: number
  searchParams: Record<string, unknown>
  logCategory: 'drawing' | 'task'
}): {
  p: number
  page_size: number
  channel_id?: string
  start_timestamp?: number
  end_timestamp?: number
} {
  const { page, pageSize, searchParams, logCategory } = config

  return {
    p: page,
    page_size: pageSize,
    ...(searchParams.channel
      ? {
          channel_id: String(searchParams.channel),
        }
      : {}),
    ...buildTimeRangeParams(searchParams, logCategory),
  }
}

/**
 * Build API params from search params and column filters (for common logs)
 */
export function buildApiParams(config: {
  page: number
  pageSize: number
  searchParams: Record<string, unknown>
  columnFilters?: Array<{ id: string; value: unknown }>
  isAdmin: boolean
}): GetLogsParams {
  const { page, pageSize, searchParams, columnFilters = [], isAdmin } = config

  // Helper to process type parameter (single value from array)
  const processType = (value: unknown): number | undefined => {
    const parseType = (raw: unknown): number | undefined => {
      const type = Number(raw)
      return Number.isFinite(type) ? type : undefined
    }

    if (Array.isArray(value) && value.length === 1) {
      return parseType(value[0])
    }
    if (typeof value === 'string' && value !== '') {
      return parseType(value)
    }
    return undefined
  }

  // Build base params from search params
  const params: GetLogsParams = {
    p: page,
    page_size: pageSize,
    ...(searchParams.type ? { type: processType(searchParams.type) } : {}),
    ...(searchParams.model ? { model_name: String(searchParams.model) } : {}),
    ...(searchParams.token ? { token_name: String(searchParams.token) } : {}),
    ...(searchParams.group ? { group: String(searchParams.group) } : {}),
    ...(isAdmin && searchParams.channel
      ? { channel: Number(searchParams.channel) || 0 }
      : {}),
    ...(isAdmin && searchParams.username
      ? { username: String(searchParams.username) }
      : {}),
    ...(searchParams.requestId
      ? { request_id: String(searchParams.requestId) }
      : {}),
    ...(searchParams.upstreamRequestId
      ? { upstream_request_id: String(searchParams.upstreamRequestId) }
      : {}),
    ...buildTimeRangeParams(searchParams, 'common'),
  }

  // Override with column filters if present
  if (columnFilters.length > 0) {
    columnFilters.forEach(({ id, value }) => {
      if (value === undefined || value === null || value === '') return

      switch (id) {
        case 'type':
          params.type = processType(value)
          break
        case 'model_name':
          params.model_name = String(value)
          break
        case 'token_name':
          params.token_name = String(value)
          break
        case 'group':
          params.group = String(value)
          break
        case 'channel':
          if (isAdmin) params.channel = Number(value) || 0
          break
        case 'username':
          if (isAdmin) params.username = String(value)
          break
      }
    })
  }

  return params
}

// ============================================================================
// Data Fetching
// ============================================================================

/**
 * Fetch logs based on category type
 */
export async function fetchLogsByCategory(
  config: FetchLogsConfig
): Promise<GetLogsResponse> {
  const { logCategory, isAdmin, page, pageSize, searchParams, columnFilters } =
    config

  if (logCategory === 'common' || logCategory === 'drawing') {
    const params = buildApiParams({
      page,
      pageSize,
      searchParams:
        logCategory === 'drawing'
          ? { ...searchParams, model: searchParams.filter }
          : searchParams,
      columnFilters,
      isAdmin,
    })
    if (logCategory === 'drawing') {
      params.type = 2
      params.request_type = 'image'
    }
    return isAdmin ? await getAllLogs(params) : await getUserLogs(params)
  }

  const baseParams = buildBaseParams({
    page,
    pageSize,
    searchParams,
    logCategory: 'task',
  })

  const paramsWithFilter = {
    ...baseParams,
    task_id: searchParams.filter as string | undefined,
  }

  return isAdmin
    ? await getAllTaskLogs(paramsWithFilter as GetTaskLogsParams)
    : await getUserTaskLogs(paramsWithFilter as GetTaskLogsParams)
}
