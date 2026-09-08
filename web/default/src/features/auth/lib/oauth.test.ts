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

import { hasOAuthProviders } from './oauth'

describe('hasOAuthProviders', () => {
  it('recognizes a custom OAuth provider as an available login method', () => {
    const available = hasOAuthProviders({
      custom_oauth_providers: [
        {
          id: 1,
          name: 'Example',
          slug: 'example',
          icon: '',
          client_id: 'client-id',
          authorization_endpoint: 'https://example.com/oauth/authorize',
          scopes: 'openid',
        },
      ],
    })

    expect(available).toBe(true)
  })

  it('returns false when no OAuth login method is enabled', () => {
    expect(hasOAuthProviders({})).toBe(false)
  })
})
