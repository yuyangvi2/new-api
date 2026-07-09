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
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'

import { ApiKeysDialogs } from './components/api-keys-dialogs'
import { ApiKeysPrimaryButtons } from './components/api-keys-primary-buttons'
import { ApiKeysProvider } from './components/api-keys-provider'
import { ApiKeysTable } from './components/api-keys-table'

export function ApiKeys() {
  const { t } = useTranslation()
  return (
    <ApiKeysProvider>
      <SectionPageLayout fixedContent>
        <SectionPageLayout.Title>{t('API Keys')}</SectionPageLayout.Title>
        <SectionPageLayout.Description>
          {t(
            'Create scoped keys, inspect usage, rotate secrets, and keep production traffic isolated by group and quota.'
          )}
        </SectionPageLayout.Description>
        <SectionPageLayout.Actions>
          <ApiKeysPrimaryButtons />
        </SectionPageLayout.Actions>
        <SectionPageLayout.Content>
          <div className='flex h-full min-h-0 flex-col gap-4'>
            <Card className='tokone-cta-band shrink-0'>
              <CardContent className='grid gap-4 p-0 lg:grid-cols-[minmax(0,1fr)_minmax(320px,0.72fr)] lg:items-center'>
                <div className='space-y-2'>
                  <div className='flex flex-wrap items-center gap-2'>
                    <Badge variant='secondary'>{t('Production ready')}</Badge>
                    <Badge variant='outline'>{t('Bearer token')}</Badge>
                  </div>
                  <h3 className='text-lg font-semibold tracking-tight'>
                    {t('Use your key with the OpenAI-compatible API')}
                  </h3>
                  <p className='text-muted-foreground max-w-2xl text-sm leading-relaxed'>
                    {t(
                      'Keys are shown once for security. Store them in your server environment and rotate them from this page when needed.'
                    )}
                  </p>
                </div>
                <pre className='bg-foreground/[0.045] text-muted-foreground overflow-hidden rounded-lg border p-3 font-mono text-xs leading-5'>
                  <code>{`Authorization: Bearer sk-...
POST /v1/chat/completions
model: gpt-4o-mini`}</code>
                </pre>
              </CardContent>
            </Card>
            <div className='min-h-0 flex-1'>
              <ApiKeysTable />
            </div>
          </div>
        </SectionPageLayout.Content>
      </SectionPageLayout>

      <ApiKeysDialogs />
    </ApiKeysProvider>
  )
}
