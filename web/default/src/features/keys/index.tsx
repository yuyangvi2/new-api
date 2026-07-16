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
import { Check, Copy, Gauge } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { openExternalSpeedTest } from '@/features/dashboard/lib'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'
import { getApiBaseAddress } from '@/lib/server-address'
import { cn } from '@/lib/utils'

import { ApiKeysDialogs } from './components/api-keys-dialogs'
import { ApiKeysPrimaryButtons } from './components/api-keys-primary-buttons'
import { ApiKeysProvider } from './components/api-keys-provider'
import { ApiKeysTable } from './components/api-keys-table'

export function ApiKeys() {
  const { t } = useTranslation()
  const { copiedText, copyToClipboard } = useCopyToClipboard({ notify: false })
  const apiBaseAddress = getApiBaseAddress()
  const endpointUrl = apiBaseAddress.endsWith('/v1')
    ? apiBaseAddress
    : `${apiBaseAddress}/v1`
  const isCopied = copiedText === endpointUrl
  const copyLabel = isCopied ? t('Copied!') : t('Click to copy')

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
            <div className='border-border/60 bg-muted/40 text-muted-foreground inline-flex max-w-full flex-wrap items-center gap-1.5 self-start rounded-md border px-2 py-1 text-xs'>
              <span className='font-medium'>{t('API Endpoint')}</span>
              <span
                aria-hidden
                className='bg-border/70 mx-0.5 h-3 w-px shrink-0'
              />
              <Tooltip>
                <TooltipTrigger
                  render={
                    <button
                      type='button'
                      onClick={() => copyToClipboard(endpointUrl)}
                      aria-label={copyLabel}
                      className={cn(
                        'text-muted-foreground hover:text-foreground focus-visible:ring-ring/50 focus-visible:text-foreground max-w-[min(68vw,420px)] truncate rounded-sm font-mono outline-none decoration-dashed underline-offset-2 hover:underline focus-visible:ring-2',
                        isCopied && 'text-foreground'
                      )}
                    >
                      {endpointUrl}
                    </button>
                  }
                />
                <TooltipContent>{copyLabel}</TooltipContent>
              </Tooltip>
              <Tooltip>
                <TooltipTrigger
                  render={
                    <Button
                      variant='ghost'
                      size='icon'
                      onClick={() => copyToClipboard(endpointUrl)}
                      aria-label={copyLabel}
                      className='text-muted-foreground hover:text-foreground -my-0.5 size-5'
                    >
                      {isCopied ? (
                        <Check className='text-success size-3' />
                      ) : (
                        <Copy className='size-3' />
                      )}
                    </Button>
                  }
                />
                <TooltipContent>{copyLabel}</TooltipContent>
              </Tooltip>
              <Tooltip>
                <TooltipTrigger
                  render={
                    <Button
                      variant='ghost'
                      size='icon'
                      onClick={() => openExternalSpeedTest(endpointUrl)}
                      aria-label={t('External Speed Test')}
                      className='text-muted-foreground hover:text-foreground -my-0.5 size-5'
                    >
                      <Gauge className='size-3' />
                    </Button>
                  }
                />
                <TooltipContent>{t('External Speed Test')}</TooltipContent>
              </Tooltip>
            </div>
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
