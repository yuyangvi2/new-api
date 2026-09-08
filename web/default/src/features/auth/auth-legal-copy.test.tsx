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
import * as bunTest from 'bun:test'

import type { ReactNode } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'

const { describe, expect, it } = bunTest
const mock = (
  bunTest as unknown as {
    mock: { module: (specifier: string, factory: () => unknown) => void }
  }
).mock

mock.module('@tanstack/react-router', () => ({
  Link: (props: { children: ReactNode }) => <a>{props.children}</a>,
  useSearch: () => ({ redirect: undefined }),
}))

mock.module('react-i18next', () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}))

mock.module('@/hooks/use-status', () => ({
  useStatus: () => ({
    status: {
      user_agreement_enabled: true,
      privacy_policy_enabled: true,
    },
  }),
}))

mock.module('./auth-layout', () => ({
  AuthLayout: (props: { children: ReactNode }) => <main>{props.children}</main>,
}))

mock.module('./sign-in/components/user-auth-form', () => ({
  UserAuthForm: () => <form>Sign-in form</form>,
}))

mock.module('./sign-up/components/sign-up-form', () => ({
  SignUpForm: () => <form>Sign-up form</form>,
}))

const { SignIn } = await import('./sign-in')
const { SignUp } = await import('./sign-up')
const mockedReactI18next = await import('react-i18next')

describe('authentication legal copy', () => {
  it('preserves unrelated react-i18next exports for other tests', () => {
    expect(typeof mockedReactI18next.I18nextProvider).toBe('function')
  })

  it('does not repeat the agreement notice below the sign-in form', () => {
    const markup = renderToStaticMarkup(<SignIn />)

    expect(markup.includes('Sign-in form')).toBe(true)
    expect(markup.includes('By clicking sign in')).toBe(false)
  })

  it('does not repeat the agreement notice below the sign-up form', () => {
    const markup = renderToStaticMarkup(<SignUp />)

    expect(markup.includes('Sign-up form')).toBe(true)
    expect(markup.includes('By creating an account')).toBe(false)
  })
})
