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
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import {
  generateExprFromVisualConfig,
  tryParseVisualConfig,
  type VisualConfig,
} from './tier-expr'

describe('tier expression visual config', () => {
  test('generates and parses time period tier conditions', () => {
    const config: VisualConfig = {
      tiers: [
        {
          label: 'peak_morning',
          conditions: [
            {
              type: 'time_period',
              timezone: 'Asia/Shanghai',
              start: '09:00',
              end: '12:00',
            },
          ],
          input_unit_cost: 1.260504,
          output_unit_cost: 3.781513,
          cache_mode: 'generic',
          cache_read_unit_cost: 0.042017,
        },
        {
          label: 'peak_afternoon',
          conditions: [
            {
              type: 'time_period',
              timezone: 'Asia/Shanghai',
              start: '14:00',
              end: '18:00',
            },
          ],
          input_unit_cost: 1.260504,
          output_unit_cost: 3.781513,
          cache_mode: 'generic',
          cache_read_unit_cost: 0.042017,
        },
        {
          label: 'off_peak',
          conditions: [],
          input_unit_cost: 0.630252,
          output_unit_cost: 1.890756,
          cache_mode: 'generic',
          cache_read_unit_cost: 0.021008,
        },
      ],
    }

    const expr = generateExprFromVisualConfig(config)

    assert.match(
      expr,
      /\(hour\("Asia\/Shanghai"\) \* 60 \+ minute\("Asia\/Shanghai"\)\) >= 540/
    )
    assert.match(
      expr,
      /\(hour\("Asia\/Shanghai"\) \* 60 \+ minute\("Asia\/Shanghai"\)\) < 720/
    )

    const parsed = tryParseVisualConfig(expr)

    assert.deepEqual(parsed?.tiers[0].conditions, config.tiers[0].conditions)
    assert.equal(parsed?.tiers[0].label, 'peak_morning')
    assert.deepEqual(parsed?.tiers[1].conditions, config.tiers[1].conditions)
    assert.equal(parsed?.tiers[1].label, 'peak_afternoon')
    assert.equal(parsed?.tiers[2].label, 'off_peak')
  })

  test('generates cross-midnight time period conditions', () => {
    const expr = generateExprFromVisualConfig({
      tiers: [
        {
          label: 'off_peak',
          conditions: [
            {
              type: 'time_period',
              timezone: 'Asia/Shanghai',
              start: '18:00',
              end: '09:00',
            },
          ],
          input_unit_cost: 0.630252,
          output_unit_cost: 1.890756,
          cache_mode: 'generic',
        },
      ],
    })

    assert.match(expr, />= 1080 \|\|/)
    assert.match(expr, /< 540/)
    assert.deepEqual(tryParseVisualConfig(expr)?.tiers[0].conditions, [
      {
        type: 'time_period',
        timezone: 'Asia/Shanghai',
        start: '18:00',
        end: '09:00',
      },
    ])
  })
})
