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
export const classicCapWidgetTheme = {
  '--cap-widget-width': '100%',
  '--cap-widget-height': '3rem',
  '--cap-background': 'var(--semi-color-bg-0)',
  '--cap-border-color': 'var(--semi-color-border)',
  '--cap-border-radius': '0.75rem',
  '--cap-color': 'var(--semi-color-text-0)',
  '--cap-font': 'inherit',
  '--cap-checkbox-size': '1.25rem',
  '--cap-checkbox-border': '1px solid var(--semi-color-border)',
  '--cap-checkbox-border-radius': '0.375rem',
  '--cap-checkbox-background': 'var(--semi-color-fill-0)',
  '--cap-focus-ring': 'var(--semi-color-focus-border)',
  '--cap-spinner-background-color': 'var(--semi-color-fill-1)',
  '--cap-spinner-color': 'var(--semi-color-text-0)',
  '--cap-troubleshoot-color': 'var(--semi-color-text-2)',
  '--cap-credits-color': 'var(--semi-color-text-2)',
};

const capThemeStyleAttribute = 'data-new-api-cap-theme';
const capCreditsRule =
  '.credits { color: var(--cap-credits-color) !important; }';

export function applyCapWidgetTheme(element, theme) {
  element.style.setProperty('display', 'block');
  element.style.setProperty('width', '100%');

  for (const [name, value] of Object.entries(theme)) {
    element.style.setProperty(name, value);
  }

  const shadowRoot = element.shadowRoot;
  if (!shadowRoot || shadowRoot.querySelector(`[${capThemeStyleAttribute}]`)) {
    return;
  }

  const style = element.ownerDocument.createElement('style');
  style.setAttribute(capThemeStyleAttribute, '');
  if (typeof window !== 'undefined' && window.CAP_CSS_NONCE) {
    style.setAttribute('nonce', window.CAP_CSS_NONCE);
  }
  style.textContent = capCreditsRule;
  shadowRoot.append(style);
}
