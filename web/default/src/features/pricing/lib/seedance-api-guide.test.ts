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
  buildSeedanceGuideSample,
  getSeedanceGuideFlows,
  type SeedanceGuideFlowId,
} from './seedance-api-guide'

const ctx = {
  baseUrl: 'https://api.example.com',
  apiKeyEnv: 'NEW_API_KEY',
  modelName: 'doubao-seedance-2-0-fast',
  endpointPath: '/v1/videos/generations',
}

describe('Seedance API guide samples', () => {
  it('lists the video, asset, real-person, and workflow guide flows', () => {
    const flows = getSeedanceGuideFlows()

    const flowIds = flows.map((flow) => flow.id)
    const expectedFlowIds: SeedanceGuideFlowId[] = [
      'video',
      'asset',
      'real-person',
      'workflow',
    ]

    expect(JSON.stringify(flowIds)).toBe(JSON.stringify(expectedFlowIds))
  })

  it('builds asset management examples against the Seedance Official resource routes', () => {
    const sample = buildSeedanceGuideSample('asset', 'curl', ctx)

    expect(sample.includes('/v1/seedance-official/asset-groups')).toBe(true)
    expect(sample.includes('/v1/seedance-official/assets')).toBe(true)
    expect(sample.includes('/v1/seedance-official/assets/query')).toBe(true)
    expect(sample.includes('Authorization: Bearer $NEW_API_KEY')).toBe(true)
  })

  it('builds real-person authentication examples with session and token exchange routes', () => {
    const sample = buildSeedanceGuideSample('real-person', 'typescript', ctx)

    expect(
      sample.includes('/v1/seedance-official/real-person-auth/sessions')
    ).toBe(true)
    expect(
      sample.includes(
        '/v1/seedance-official/real-person-auth/asset-group/by-byted-token'
      )
    ).toBe(true)
    expect(sample.includes('H5Link')).toBe(true)
    expect(sample.includes('byted_token')).toBe(true)
  })

  it('keeps the full workflow sample tied to the selected model and video endpoint', () => {
    const sample = buildSeedanceGuideSample('workflow', 'python', ctx)

    expect(sample.includes('"model": "doubao-seedance-2-0-fast"')).toBe(true)
    expect(
      sample.includes('https://api.example.com/v1/videos/generations')
    ).toBe(true)
    expect(sample.includes('/v1/seedance-official/assets')).toBe(true)
  })
})
