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

import {
  CHANNEL_FORM_DEFAULT_VALUES,
  channelFormSchema,
  isValidTencentVODModelMapping,
  transformChannelToFormDefaults,
  transformFormDataToUpdatePayload,
} from './channel-form'

describe('channel Responses compatibility setting', () => {
  it('preserves responses_via_chat_completions when editing a channel', () => {
    const channel = {
      id: 37,
      name: 'FW - DeepSeek',
      type: 1,
      models: 'deepseek-v4',
      group: 'default',
      status: 1,
      setting: JSON.stringify({ responses_via_chat_completions: true }),
      settings: '{}',
      channel_info: {},
    } as Parameters<typeof transformChannelToFormDefaults>[0]

    const formValues = transformChannelToFormDefaults(channel)
    expect(formValues.responses_via_chat_completions).toBe(true)

    const payload = transformFormDataToUpdatePayload(formValues, channel.id)
    expect(
      JSON.parse(String(payload.setting)).responses_via_chat_completions
    ).toBe(true)
  })
})

describe('Tencent VOD Image settings', () => {
  it('validates resolved Tencent model names and versions', () => {
    expect(
      isValidTencentVODModelMapping(
        'gemini-image',
        { 'gemini-image': 'alias', alias: 'Vidu:q2' },
        'GG'
      )
    ).toBe(true)
    expect(
      isValidTencentVODModelMapping(
        'gemini-image',
        { 'gemini-image': '3.1' },
        'GG'
      )
    ).toBe(true)
    expect(
      isValidTencentVODModelMapping(
        'gemini-image',
        { 'gemini-image': 'Unknown:1' },
        'GG'
      )
    ).toBe(false)
    expect(
      isValidTencentVODModelMapping(
        'gemini-image',
        { 'gemini-image': 'GG:' },
        'GG'
      )
    ).toBe(false)
  })

  it('requires a mapping for every configured public model', () => {
    const result = channelFormSchema.safeParse({
      ...CHANNEL_FORM_DEFAULT_VALUES,
      name: 'Tencent image',
      type: 9007,
      key: 'secret-id|secret-key',
      models: 'gemini-3-pro-image,vidu-q2',
      group: ['default'],
      model_mapping: JSON.stringify({
        'gemini-3-pro-image': 'GG:3.0',
      }),
    })

    expect(result.success).toBe(false)
    if (!result.success) {
      expect(
        result.error.issues.some(
          (issue) => issue.path.join('.') === 'model_mapping'
        )
      ).toBe(true)
    }
  })

  it('preserves SubAppId and model defaults when editing a channel', () => {
    const channel = {
      id: 9007,
      name: 'Tencent image',
      type: 9007,
      models: 'vidu-q2',
      group: 'default',
      status: 1,
      setting: '{}',
      settings: JSON.stringify({
        vod_aigc: {
          sub_app_id: 123456,
          input_region: 'Oversea',
          default_model_name: 'Vidu',
        },
      }),
      channel_info: {},
    } as Parameters<typeof transformChannelToFormDefaults>[0]

    const formValues = transformChannelToFormDefaults(channel)
    expect(formValues.vod_sub_app_id).toBe('123456')
    expect(formValues.vod_input_region).toBe('Oversea')
    expect(formValues.vod_default_model_name).toBe('Vidu')

    const payload = transformFormDataToUpdatePayload(formValues, channel.id)
    expect(JSON.stringify(JSON.parse(String(payload.settings)).vod_aigc)).toBe(
      JSON.stringify({
        sub_app_id: 123456,
        input_region: 'Oversea',
        default_model_name: 'Vidu',
      })
    )
  })
})
