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
  applyCapWidgetTheme,
  defaultCapWidgetTheme,
} from './cap-widget-theme'

describe('default Cap widget theme', () => {
  it('uses application semantic colors and weakens only the attribution', () => {
    const hostProperties = new Map<string, string>()
    const creditProperties = new Map<string, [string, string | undefined]>()
    const element = {
      style: {
        setProperty: (name: string, value: string) => {
          hostProperties.set(name, value)
        },
      },
      shadowRoot: {
        querySelector: (selector: string) => {
          if (selector !== '.credits') return null
          return {
            style: {
              setProperty: (name: string, value: string, priority?: string) => {
                creditProperties.set(name, [value, priority])
              },
            },
          }
        },
      },
    } as unknown as HTMLElement

    applyCapWidgetTheme(element, defaultCapWidgetTheme)

    expect(hostProperties.get('--cap-background')).toBe('var(--background)')
    expect(hostProperties.get('--cap-border-color')).toBe('var(--border)')
    expect(hostProperties.get('--cap-color')).toBe('var(--foreground)')
    expect(hostProperties.get('--cap-focus-ring')).toBe('var(--ring)')
    expect(hostProperties.get('--cap-widget-width')).toBe('100%')
    expect(creditProperties.get('color')).toEqual([
      'var(--cap-credits-color)',
      'important',
    ])
  })
})
