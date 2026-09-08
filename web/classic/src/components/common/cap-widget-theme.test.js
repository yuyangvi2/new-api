/*
Copyright (C) 2025 QuantumNous

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
import { describe, expect, it } from 'bun:test';

import { applyCapWidgetTheme, classicCapWidgetTheme } from './cap-widget-theme';

describe('classic Cap widget theme', () => {
  it('uses Semi semantic colors and weakens only the attribution', () => {
    const hostProperties = new Map();
    const styleAttributes = new Map();
    let injectedStyle = null;
    const element = {
      style: {
        setProperty: (name, value) => hostProperties.set(name, value),
      },
      shadowRoot: {
        querySelector: () => injectedStyle,
        append: (node) => {
          injectedStyle = node;
        },
      },
      ownerDocument: {
        createElement: () => ({
          textContent: null,
          setAttribute: (name, value) => styleAttributes.set(name, value),
        }),
      },
    };

    applyCapWidgetTheme(element, classicCapWidgetTheme);
    applyCapWidgetTheme(element, classicCapWidgetTheme);

    expect(hostProperties.get('--cap-background')).toBe(
      'var(--semi-color-bg-0)',
    );
    expect(hostProperties.get('--cap-border-color')).toBe(
      'var(--semi-color-border)',
    );
    expect(hostProperties.get('--cap-color')).toBe('var(--semi-color-text-0)');
    expect(hostProperties.get('--cap-focus-ring')).toBe(
      'var(--semi-color-focus-border)',
    );
    expect(hostProperties.get('--cap-widget-width')).toBe('100%');
    expect(styleAttributes.has('data-new-api-cap-theme')).toBe(true);
    expect(injectedStyle?.textContent).toBe(
      '.credits { color: var(--cap-credits-color) !important; }',
    );
  });
});
