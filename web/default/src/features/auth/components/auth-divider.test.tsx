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

import { createInstance } from 'i18next'
import { renderToStaticMarkup } from 'react-dom/server'
import { I18nextProvider, initReactI18next } from 'react-i18next'

import { AuthDivider } from './auth-divider'

describe('AuthDivider', () => {
  it('renders one semantic divider with the localized alternative-login label', async () => {
    const i18n = createInstance()
    await i18n.use(initReactI18next).init({
      lng: 'en',
      resources: {
        en: {
          translation: {
            'Or continue with': 'Continue another way',
          },
        },
      },
    })
    const markup = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <AuthDivider />
      </I18nextProvider>
    )

    expect(markup.match(/data-slot="separator"/g)?.length).toBe(1)
    expect(markup.includes('Continue another way')).toBe(true)
  })
})
