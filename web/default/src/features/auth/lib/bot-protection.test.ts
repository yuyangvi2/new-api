import { describe, expect, it } from 'bun:test'

import { buildCapEndpoint, resolveBotProtectionConfig } from './bot-protection'

describe('bot protection configuration', () => {
  it('uses scene-specific Cap configuration', () => {
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
      'login'
    )

    expect(config.site_key).toBe('login-key')
    expect(buildCapEndpoint(config)).toBe(
      'https://captcha.example.com/cap/login-key/'
    )
  })

  it('keeps legacy Turnstile status compatible', () => {
    const config = resolveBotProtectionConfig(
      { turnstile_check: true, turnstile_site_key: 'legacy-key' },
      'register'
    )

    expect(JSON.stringify(config)).toBe(
      JSON.stringify({
        enabled: true,
        provider: 'turnstile',
        public_endpoint: '',
        site_key: 'legacy-key',
      })
    )
  })
})
