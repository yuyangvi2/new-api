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

import { isVideoGenerationModelName } from './model-classification'

describe('isVideoGenerationModelName', () => {
  it('does not classify MiniMax text models as video models', () => {
    const textModels = [
      'minimax-m2.5',
      'minimax-m2.5-highspeed',
      'minimax-m2.7',
      'minimax-m3',
    ]

    for (const model of textModels) {
      expect(isVideoGenerationModelName(model)).toBe(false)
    }
  })

  it('keeps known MiniMax video models in the video list', () => {
    expect(isVideoGenerationModelName('video-01')).toBe(true)
    expect(isVideoGenerationModelName('I2V-01-Director')).toBe(true)
  })
})
