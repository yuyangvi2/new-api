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

import { DEFAULT_CONFIG, DEFAULT_PARAMETER_ENABLED } from '../../constants'
import type { Message } from '../../types'
import { buildChatCompletionPayload } from './payload-builder'

const userMessage: Message = {
  key: 'message-1',
  from: 'user',
  versions: [{ id: 'version-1', content: 'hello' }],
}

describe('buildChatCompletionPayload', () => {
  it('omits deprecated sampling parameters for Claude Opus 4.8', () => {
    const payload = buildChatCompletionPayload(
      [userMessage],
      {
        ...DEFAULT_CONFIG,
        model: 'claude-opus-4-8',
        temperature: 0.7,
        top_p: 0.9,
      },
      {
        ...DEFAULT_PARAMETER_ENABLED,
        temperature: true,
        top_p: true,
      }
    )

    expect(payload.temperature).toBeUndefined()
    expect(payload.top_p).toBeUndefined()
  })
})
