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
  buildRequestRuleExpr,
  getRequestRuleMultiplierRange,
  parseTiersFromExpr,
  splitBillingExprAndRequestRules,
  tryParseRequestRuleExpr,
} from './billing-expr'

describe('billing expression parser', () => {
  test('applies top-level conditional multiplier ranges to parsed tier prices', () => {
    const expr =
      'tier("base", p * 0.210084 + c * 0.630252 + cr * 0.007003) * ((((hour("Asia/Shanghai") * 60 + minute("Asia/Shanghai")) >= 540 && (hour("Asia/Shanghai") * 60 + minute("Asia/Shanghai")) < 720) || ((hour("Asia/Shanghai") * 60 + minute("Asia/Shanghai")) >= 840 && (hour("Asia/Shanghai") * 60 + minute("Asia/Shanghai")) < 1080)) ? 2 : 1)'

    const tiers = parseTiersFromExpr(expr)

    assert.equal(tiers.length, 1)
    assert.equal(tiers[0].inputPrice, 0.210084)
    assert.equal(tiers[0].inputPriceMax, 0.420168)
    assert.equal(tiers[0].outputPrice, 0.630252)
    assert.equal(tiers[0].outputPriceMax, 1.260504)
    assert.equal(tiers[0].cacheReadPrice, 0.007003)
    assert.equal(tiers[0].cacheReadPriceMax, 0.014006)
  })

  test('splits minute-of-day conditional multipliers into visual request rules', () => {
    const expr =
      'tier("base", p * 0.210084 + c * 0.630252 + cr * 0.007003) * ((((hour("Asia/Shanghai") * 60 + minute("Asia/Shanghai")) >= 540 && (hour("Asia/Shanghai") * 60 + minute("Asia/Shanghai")) < 720) || ((hour("Asia/Shanghai") * 60 + minute("Asia/Shanghai")) >= 840 && (hour("Asia/Shanghai") * 60 + minute("Asia/Shanghai")) < 1080)) ? 2 : 1)'

    const split = splitBillingExprAndRequestRules(expr)
    const groups = tryParseRequestRuleExpr(split.requestRuleExpr)

    assert.equal(
      split.billingExpr,
      'tier("base", p * 0.210084 + c * 0.630252 + cr * 0.007003)'
    )
    assert.equal(groups?.length, 2)
    assert.deepEqual(groups?.[0], {
      conditions: [
        {
          source: 'time',
          timeFunc: 'hour',
          timezone: 'Asia/Shanghai',
          mode: 'range',
          value: '',
          rangeStart: '9',
          rangeEnd: '12',
        },
      ],
      multiplier: '2',
    })
    assert.deepEqual(groups?.[1], {
      conditions: [
        {
          source: 'time',
          timeFunc: 'hour',
          timezone: 'Asia/Shanghai',
          mode: 'range',
          value: '',
          rangeStart: '14',
          rangeEnd: '18',
        },
      ],
      multiplier: '2',
    })
    assert.equal(
      buildRequestRuleExpr(groups || []),
      '(hour("Asia/Shanghai") >= 9 && hour("Asia/Shanghai") < 12 ? 2 : 1) * (hour("Asia/Shanghai") >= 14 && hour("Asia/Shanghai") < 18 ? 2 : 1)'
    )
    assert.deepEqual(getRequestRuleMultiplierRange(split.requestRuleExpr), {
      min: 1,
      max: 2,
    })
  })

  test('splits weekday-guarded peak hours into visual request rules', () => {
    const expr =
      'tier("base", p * 0.28 + c * 1.12 + cr * 0.028) * ' +
      '((weekday("Asia/Shanghai") >= 1 && weekday("Asia/Shanghai") < 6 && ' +
      '(((hour("Asia/Shanghai") * 60 + minute("Asia/Shanghai")) >= 540 && ' +
      '(hour("Asia/Shanghai") * 60 + minute("Asia/Shanghai")) < 720) || ' +
      '((hour("Asia/Shanghai") * 60 + minute("Asia/Shanghai")) >= 840 && ' +
      '(hour("Asia/Shanghai") * 60 + minute("Asia/Shanghai")) < 1080)) ? 2 : 1))'

    const split = splitBillingExprAndRequestRules(expr)
    const groups = tryParseRequestRuleExpr(split.requestRuleExpr)

    assert.equal(
      split.billingExpr,
      'tier("base", p * 0.28 + c * 1.12 + cr * 0.028)'
    )
    assert.equal(groups?.length, 2)
    assert.deepEqual(groups?.[0], {
      conditions: [
        {
          source: 'time',
          timeFunc: 'weekday',
          timezone: 'Asia/Shanghai',
          mode: 'gte',
          value: '1',
          rangeStart: '',
          rangeEnd: '',
        },
        {
          source: 'time',
          timeFunc: 'weekday',
          timezone: 'Asia/Shanghai',
          mode: 'lt',
          value: '6',
          rangeStart: '',
          rangeEnd: '',
        },
        {
          source: 'time',
          timeFunc: 'hour',
          timezone: 'Asia/Shanghai',
          mode: 'range',
          value: '',
          rangeStart: '9',
          rangeEnd: '12',
        },
      ],
      multiplier: '2',
    })
    assert.deepEqual(groups?.[1]?.conditions[2], {
      source: 'time',
      timeFunc: 'hour',
      timezone: 'Asia/Shanghai',
      mode: 'range',
      value: '',
      rangeStart: '14',
      rangeEnd: '18',
    })
    assert.equal(
      buildRequestRuleExpr(groups || []),
      '(weekday("Asia/Shanghai") >= 1 && weekday("Asia/Shanghai") < 6 && ' +
        'hour("Asia/Shanghai") >= 9 && hour("Asia/Shanghai") < 12 ? 2 : 1) * ' +
        '(weekday("Asia/Shanghai") >= 1 && weekday("Asia/Shanghai") < 6 && ' +
        'hour("Asia/Shanghai") >= 14 && hour("Asia/Shanghai") < 18 ? 2 : 1)'
    )
    assert.deepEqual(getRequestRuleMultiplierRange(split.requestRuleExpr), {
      min: 1,
      max: 2,
    })
  })
})
