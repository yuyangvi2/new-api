import { describe, expect, it } from 'bun:test'

import {
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
