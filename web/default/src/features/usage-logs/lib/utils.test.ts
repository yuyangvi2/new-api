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
import { describe, expect, it } from 'bun:test'

import { buildTimeRangeParams, getDefaultTimeRange } from './time-range'

describe('usage log default time ranges', () => {
  it('loads the last 30 days for drawing logs', () => {
    const { start, end } = getDefaultTimeRange('drawing')
    const expectedStart = new Date(end.getTime() - 3600 * 1000)
    expectedStart.setDate(expectedStart.getDate() - 30)
    const params = buildTimeRangeParams(
      { startTime: 1_777_777_777_123, endTime: 1_888_888_888_456 },
      'drawing'
    )

    expect(start.getTime()).toBe(expectedStart.getTime())
    expect(params.start_timestamp).toBe(1_777_777_777_123)
    expect(params.end_timestamp).toBe(1_888_888_888_456)
  })

  it('keeps task logs scoped to today with second timestamps', () => {
    const { start } = getDefaultTimeRange('task')
    const params = buildTimeRangeParams(
      { startTime: 1_777_777_777_123, endTime: 1_888_888_888_456 },
      'task'
    )

    expect(start.getHours()).toBe(0)
    expect(start.getMinutes()).toBe(0)
    expect(start.getSeconds()).toBe(0)
    expect(start.getMilliseconds()).toBe(0)
    expect(params.start_timestamp).toBe(1_777_777_777)
    expect(params.end_timestamp).toBe(1_888_888_888)
  })
})
