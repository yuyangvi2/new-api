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
type LogCategory = 'common' | 'drawing' | 'task'

export function getDefaultTimeRange(logCategory: LogCategory = 'common'): {
  start: Date
  end: Date
} {
  const now = new Date()
  const start = new Date(now)
  if (logCategory === 'drawing') {
    start.setDate(start.getDate() - 30)
  } else {
    start.setHours(0, 0, 0, 0)
  }
  const end = new Date(now.getTime() + 3600 * 1000)

  return { start, end }
}

export function buildTimeRangeParams(
  searchParams: Record<string, unknown>,
  logCategory: LogCategory
): { start_timestamp?: number; end_timestamp?: number } {
  const hasTimeParams = searchParams.startTime ?? searchParams.endTime
  const defaultTimeRange = !hasTimeParams
    ? getDefaultTimeRange(logCategory)
    : null

  const getTimestamp = (paramTime?: unknown, defaultTime?: Date) => {
    const time = (paramTime as number) || defaultTime?.getTime()
    if (!time) return undefined
    return logCategory === 'drawing' ? time : Math.floor(time / 1000)
  }

  return {
    start_timestamp: getTimestamp(
      searchParams.startTime,
      defaultTimeRange?.start
    ),
    end_timestamp: getTimestamp(searchParams.endTime, defaultTimeRange?.end),
  }
}
