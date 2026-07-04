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
import i18n, {
  type BackendModule,
  type ReadCallback,
  type ResourceKey,
} from 'i18next'
import LanguageDetector from 'i18next-browser-languagedetector'
import { initReactI18next } from 'react-i18next'

const localeLoaders = {
  en: () => import('./locales/en.json'),
  zh: () => import('./locales/zh.json'),
  fr: () => import('./locales/fr.json'),
  ru: () => import('./locales/ru.json'),
  ja: () => import('./locales/ja.json'),
  vi: () => import('./locales/vi.json'),
} as const

type SupportedLanguage = keyof typeof localeLoaders

function normalizeLanguage(language?: string): SupportedLanguage {
  const normalized = language?.trim().replace(/_/g, '-').toLowerCase()
  if (normalized?.startsWith('zh')) return 'zh'

  if (
    normalized &&
    Object.prototype.hasOwnProperty.call(localeLoaders, normalized)
  ) {
    return normalized as SupportedLanguage
  }

  return 'en'
}

const dynamicResourceBackend: BackendModule = {
  type: 'backend',
  init() {
    /* no backend options */
  },
  read(language: string, namespace: string, callback: ReadCallback) {
    if (namespace !== 'translation') {
      callback(null, {})
      return
    }

    const normalized = normalizeLanguage(language)
    localeLoaders[normalized]()
      .then((module) => callback(null, module.default.translation))
      .catch((error: unknown) =>
        callback(
          error instanceof Error ? error : new Error(String(error)),
          null
        )
      )
  },
}

let initPromise: Promise<typeof i18n> | null = null

export function initI18n() {
  if (i18n.isInitialized) return Promise.resolve(i18n)
  if (initPromise) return initPromise

  initPromise = i18n
    .use(dynamicResourceBackend)
    .use(LanguageDetector)
    .use(initReactI18next)
    .init({
      fallbackLng: 'en',
      supportedLngs: ['en', 'zh', 'fr', 'ru', 'ja', 'vi'],
      load: 'languageOnly', // Convert zh-CN -> zh
      nsSeparator: false, // Allow literal colons in keys (e.g., URLs, labels)
      defaultNS: 'translation',
      ns: ['translation'],
      debug: import.meta.env.DEV,
      interpolation: {
        escapeValue: false, // not needed for react as it escapes by default
      },
      detection: {
        order: ['localStorage', 'navigator'],
        caches: ['localStorage'],
      },
      react: {
        useSuspense: false,
      },
    })
    .then(() => i18n)

  return initPromise
}

export async function loadLanguage(language: string) {
  await initI18n()
  const normalized = normalizeLanguage(language)
  if (!i18n.hasResourceBundle(normalized, 'translation')) {
    const module = await localeLoaders[normalized]()
    i18n.addResourceBundle(
      normalized,
      'translation',
      module.default.translation as ResourceKey,
      true,
      true
    )
  }
  await i18n.changeLanguage(normalized)
}

export default i18n
