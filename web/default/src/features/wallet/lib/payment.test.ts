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

import { PAYMENT_TYPES } from '../constants'
import {
  isSafeHttpPaymentUrl,
  isStripePayment,
  isWaffoPancakePayment,
} from './payment'

describe('payment type classification', () => {
  test('keeps Waffo Pancake and Stripe on their dedicated flows', () => {
    assert.equal(isWaffoPancakePayment(PAYMENT_TYPES.WAFFO_PANCAKE), true)
    assert.equal(isWaffoPancakePayment(PAYMENT_TYPES.WAFFO), false)
    assert.equal(isStripePayment(PAYMENT_TYPES.STRIPE), true)
  })
})

describe('payment checkout URL safety', () => {
  test('allows only absolute http(s) URLs', () => {
    assert.equal(isSafeHttpPaymentUrl('https://pay.example.com/checkout'), true)
    assert.equal(isSafeHttpPaymentUrl('http://pay.example.com/checkout'), true)
    assert.equal(isSafeHttpPaymentUrl('javascript:alert(1)'), false)
    assert.equal(isSafeHttpPaymentUrl('data:text/html,hi'), false)
    assert.equal(isSafeHttpPaymentUrl('/relative/path'), false)
    assert.equal(isSafeHttpPaymentUrl(''), false)
  })
})
