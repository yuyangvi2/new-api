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

import { HERO_SLIDES } from './hero'

describe('home hero model lineup', () => {
  it('promotes the current flagship model versions', () => {
    expect(HERO_SLIDES.slice(0, 3).map((slide) => slide.title)).toEqual([
      'Fable 5.1',
      'GPT-6 Astra',
      'Grok 4.6',
    ])
  })

  it('does not advertise superseded model versions in hero content', () => {
    const heroContent = JSON.stringify(HERO_SLIDES.slice(0, 3))

    expect(heroContent).not.toContain('Claude Opus 5')
    expect(heroContent).not.toContain('GPT 5.6')
    expect(heroContent).not.toContain('Grok 4.5')
  })
})
