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
  detectImageModelFamily,
  getTencentVODImageOutputConfig,
  getTencentVODImageProfile,
  getTencentVODReferenceImageLimit,
  hasValidImageGenerationInput,
  limitTencentVODReferenceImages,
} from './constants'

describe('detectImageModelFamily', () => {
  it('keeps official Gemini image models on the synchronous path by default', () => {
    expect(detectImageModelFamily('gemini-2.5-flash-image')).toBe(
      'generic-image'
    )
  })

  it('uses the Tencent VOD family when the selected group prefers image tasks', () => {
    expect(detectImageModelFamily('gemini-2.5-flash-image', true)).toBe(
      'tencent-vod-image'
    )
  })

  it('uses the Tencent VOD family for Vidu image models', () => {
    expect(detectImageModelFamily('vidu-q2', true)).toBe('tencent-vod-image')
  })

  it('uses the Tencent VOD family for custom public aliases', () => {
    expect(detectImageModelFamily('my-image-alias', true)).toBe(
      'tencent-vod-image'
    )
  })
})

describe('getTencentVODReferenceImageLimit', () => {
  it('uses the upstream model identity for Vidu aliases', () => {
    expect(getTencentVODReferenceImageLimit('Vidu:q2')).toBe(7)
    expect(getTencentVODReferenceImageLimit('GG:2.5')).toBe(3)
    expect(getTencentVODReferenceImageLimit('GG:3.1')).toBe(14)
    expect(getTencentVODReferenceImageLimit('Kling:3.0')).toBe(1)
    expect(getTencentVODReferenceImageLimit('Hunyuan:3.0')).toBe(3)
  })

  it('uses prompt-only safe defaults when the upstream model is ambiguous', () => {
    const profile = getTencentVODImageProfile(undefined)
    expect(profile.aspectRatios.length).toBe(0)
    expect(profile.resolutions.length).toBe(0)
    expect(profile.maxReferenceImages).toBe(0)
    expect(
      JSON.stringify(getTencentVODImageOutputConfig(undefined, '1:1', '1K'))
    ).toBe('{}')
  })

  it('normalizes stale output selections for the selected upstream model', () => {
    expect(
      JSON.stringify(getTencentVODImageOutputConfig('Vidu:q2', '4:5', '1K'))
    ).toBe(JSON.stringify({ aspectRatio: '1:1', resolution: '1080p' }))
    expect(
      JSON.stringify(getTencentVODImageOutputConfig('Hunyuan:3.0', '1:1', '1K'))
    ).toBe('{}')
  })

  it('trims references when switching from a larger model to Vidu', () => {
    const images = Array.from({ length: 10 }, (_, index) => `image-${index}`)
    expect(limitTencentVODReferenceImages(images, 'Vidu:q2').join(',')).toBe(
      images.slice(0, 7).join(',')
    )
  })
})

describe('hasValidImageGenerationInput', () => {
  it('allows Tencent VOD reference-only requests', () => {
    expect(hasValidImageGenerationInput('tencent-vod-image', '', 1)).toBe(true)
  })

  it('still requires prompts for AIART task models', () => {
    expect(hasValidImageGenerationInput('image-gi', '', 1)).toBe(false)
    expect(hasValidImageGenerationInput('hunyuan-image', '', 1)).toBe(false)
  })
})
