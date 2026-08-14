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

import type { Model } from '../types'
import {
  buildModelPricingUpdates,
  hasModelPayloadChanged,
  hasModelPricingInput,
} from './model-update-planning'

const emptySettings = {
  ModelPrice: '',
  ModelRatio: '',
  CacheRatio: '',
  CompletionRatio: '',
  ImageRatio: '',
  AudioRatio: '',
  AudioCompletionRatio: '',
}

describe('model update planning', () => {
  test('does not update settings when empty pricing config is unchanged', () => {
    const updates = buildModelPricingUpdates({
      modelSettings: emptySettings,
      isEditing: true,
      oldModelName: 'gpt-test',
      finalModelName: 'gpt-test',
      pricingMode: 'per-token',
      values: {},
    })

    assert.deepEqual(updates, [])
  })

  test('does not update settings when JSON order changes but values are unchanged', () => {
    const updates = buildModelPricingUpdates({
      modelSettings: {
        ...emptySettings,
        ModelRatio: '{"gpt-test":1,"other":2}',
        CompletionRatio: '{"gpt-test":3,"other":4}',
      },
      isEditing: true,
      oldModelName: 'gpt-test',
      finalModelName: 'gpt-test',
      pricingMode: 'per-token',
      values: {
        ratio: '1',
        completionRatio: '3',
      },
    })

    assert.deepEqual(updates, [])
  })

  test('moves pricing config when a model is renamed', () => {
    const updates = buildModelPricingUpdates({
      modelSettings: {
        ...emptySettings,
        ModelRatio: '{"old-model":1,"other":2}',
      },
      isEditing: true,
      oldModelName: 'old-model',
      finalModelName: 'new-model',
      pricingMode: 'per-token',
      values: {
        ratio: '1',
      },
    })

    assert.deepEqual(updates, [
      {
        key: 'ModelRatio',
        value: '{"new-model":1,"other":2}',
      },
    ])
  })

  test('detects unchanged model payloads', () => {
    const currentModel: Model = {
      id: 1,
      model_name: 'gpt-test',
      description: '',
      icon: '',
      tags: 'chat,fast',
      vendor_id: undefined,
      endpoints: '{"chat":"/v1/chat/completions"}',
      name_rule: 0,
      status: 1,
      sync_official: 1,
      created_time: 0,
      updated_time: 0,
    }

    assert.equal(
      hasModelPayloadChanged(currentModel, {
        model_name: 'gpt-test',
        description: '',
        icon: '',
        tags: 'chat,fast',
        vendor_id: undefined,
        endpoints: '{\n  "chat": "/v1/chat/completions"\n}',
        name_rule: 0,
        status: 1,
        sync_official: 1,
      }),
      false
    )
  })

  test('treats empty vendor id variants as unchanged', () => {
    const currentModel: Model = {
      id: 1,
      model_name: 'gpt-test',
      vendor_id: 0,
      endpoints: '',
      name_rule: 0,
      status: 1,
      sync_official: 1,
      created_time: 0,
      updated_time: 0,
    }

    assert.equal(
      hasModelPayloadChanged(currentModel, {
        model_name: 'gpt-test',
        vendor_id: undefined,
        endpoints: '',
        name_rule: 0,
        status: 1,
        sync_official: 1,
      }),
      false
    )
  })

  test('treats equivalent tag formatting as unchanged', () => {
    const currentModel: Model = {
      id: 1,
      model_name: 'gpt-test',
      tags: 'chat, fast',
      endpoints: '',
      name_rule: 0,
      status: 1,
      sync_official: 1,
      created_time: 0,
      updated_time: 0,
    }

    assert.equal(
      hasModelPayloadChanged(currentModel, {
        model_name: 'gpt-test',
        tags: 'chat,fast',
        endpoints: '',
        name_rule: 0,
        status: 1,
        sync_official: 1,
      }),
      false
    )
  })

  test('repairs invalid pricing JSON instead of treating it as an empty map', () => {
    const updates = buildModelPricingUpdates({
      modelSettings: {
        ...emptySettings,
        ModelRatio: '{invalid',
      },
      isEditing: true,
      oldModelName: 'gpt-test',
      finalModelName: 'gpt-test',
      pricingMode: 'per-token',
      values: {},
    })

    assert.deepEqual(updates, [
      {
        key: 'ModelRatio',
        value: '{}',
      },
    ])
  })

  test('detects explicit pricing input while settings are still loading', () => {
    assert.equal(hasModelPricingInput({ ratio: '' }), false)
    assert.equal(hasModelPricingInput({ ratio: '0' }), true)
  })
})
