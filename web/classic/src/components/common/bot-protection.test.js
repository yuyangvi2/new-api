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

import { buildCapEndpoint, resolveBotProtectionConfig } from './bot-protection';

describe('classic bot protection configuration', () => {
  it('uses the requested Cap scene', () => {
    const config = resolveBotProtectionConfig(
      {
        bot_protection: {
          register: {
            enabled: true,
            provider: 'cap',
            public_endpoint: 'https://captcha.example.com/cap/',
            site_key: 'register-key',
          },
          login: {
            enabled: true,
            provider: 'cap',
            public_endpoint: 'https://captcha.example.com/cap',
            site_key: 'login-key',
          },
        },
      },
      'register',
    );

    expect(config.site_key).toBe('register-key');
    expect(buildCapEndpoint(config)).toBe(
      'https://captcha.example.com/cap/register-key/',
    );
  });

  it('falls back to legacy Turnstile configuration', () => {
    expect(
      resolveBotProtectionConfig(
        { turnstile_check: true, turnstile_site_key: 'legacy-key' },
        'login',
      ),
    ).toEqual({
      enabled: true,
      provider: 'turnstile',
      public_endpoint: '',
      site_key: 'legacy-key',
    });
  });
});
