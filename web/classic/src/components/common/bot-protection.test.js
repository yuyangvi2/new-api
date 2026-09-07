import { describe, expect, it } from 'bun:test';

import {
  buildCapEndpoint,
  resolveBotProtectionConfig,
} from './bot-protection';

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
