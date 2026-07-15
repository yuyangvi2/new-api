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
import {
  CheckIcon,
  CircleSlashIcon,
  GlobeIcon,
  PaperclipIcon,
  Trash2Icon,
} from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  PromptInputButton,
  PromptInputTools,
} from '@/components/ai-elements/prompt-input'
import { ConfirmDialog } from '@/components/confirm-dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

import {
  ATTACHMENT_ACTIONS,
  getAttachmentActionNotice,
  getWebSearchCompatibility,
} from '../../lib'
import { WEB_SEARCH_CONTEXT_OPTIONS } from '../../constants'
import type { WebSearchContextSize } from '../../types'

type PlaygroundInputToolsProps = {
  disabled?: boolean
  groupValue: string
  hasMessages?: boolean
  modelValue: string
  onClearMessages?: () => void
  onWebSearchContextSizeChange: (value: WebSearchContextSize) => void
  onWebSearchEnabledChange: (enabled: boolean) => void
  webSearchContextSize: WebSearchContextSize
  webSearchEnabled: boolean
}

export function PlaygroundInputTools({
  disabled,
  groupValue,
  hasMessages = false,
  modelValue,
  onClearMessages,
  onWebSearchContextSizeChange,
  onWebSearchEnabledChange,
  webSearchContextSize,
  webSearchEnabled,
}: PlaygroundInputToolsProps) {
  const { t } = useTranslation()
  const [clearConfirmOpen, setClearConfirmOpen] = useState(false)
  const webSearchCompatibility = getWebSearchCompatibility(
    groupValue,
    modelValue,
  )
  const canEnableWebSearch = webSearchCompatibility.supported
  const isWebSearchActive = webSearchEnabled && canEnableWebSearch

  const handleFileAction = (action: string) => {
    const notice = getAttachmentActionNotice(action)
    toast.info(t(notice.title), {
      description: notice.description,
    })
  }

  const handleEnableWebSearch = (enabled: boolean) => {
    if (enabled && !webSearchCompatibility.supported) {
      return
    }

    onWebSearchEnabledChange(enabled)
  }

  const handleClearMessages = () => {
    onClearMessages?.()
    setClearConfirmOpen(false)
    toast.success(t('Conversation cleared'))
  }

  return (
    <>
      <PromptInputTools className='bg-background/70 border-border/60 rounded-lg border p-1 shadow-xs'>
        <Tooltip>
          <DropdownMenu>
            <TooltipTrigger
              render={
                <DropdownMenuTrigger
                  render={
                    <PromptInputButton
                      aria-label={t('Attach')}
                      className='text-muted-foreground hover:text-foreground hover:bg-muted/70 font-medium'
                      disabled={disabled}
                      variant='ghost'
                    />
                  }
                >
                  <PaperclipIcon size={16} />
                </DropdownMenuTrigger>
              }
            />
            <TooltipContent>
              <p>{t('Attach')}</p>
            </TooltipContent>
            <DropdownMenuContent align='start'>
              {ATTACHMENT_ACTIONS.map(({ action, icon: Icon, label }) => (
                <DropdownMenuItem
                  key={action}
                  onClick={() => handleFileAction(action)}
                >
                  <Icon className='mr-2' size={16} />
                  {t(label)}
                </DropdownMenuItem>
              ))}
            </DropdownMenuContent>
          </DropdownMenu>
        </Tooltip>

        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <PromptInputButton
                aria-label={t('Web search')}
                className={
                  isWebSearchActive
                    ? 'bg-primary/10 text-primary hover:bg-primary/15 relative font-medium ring-1 ring-primary/15'
                    : 'text-muted-foreground hover:text-foreground hover:bg-muted/70 relative font-medium'
                }
                disabled={disabled}
                title={t('Web search')}
                variant='ghost'
              />
            }
          >
            <GlobeIcon size={16} />
            {isWebSearchActive && (
              <span className='bg-primary absolute top-1.5 right-1.5 size-1.5 rounded-full' />
            )}
          </DropdownMenuTrigger>
          <DropdownMenuContent align='start' className='w-64'>
            <DropdownMenuLabel>{t('Web search')}</DropdownMenuLabel>
            <div className='text-muted-foreground px-1.5 pb-1 text-xs leading-5'>
              {t(webSearchCompatibility.labelKey)}
            </div>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              disabled={!canEnableWebSearch && !webSearchEnabled}
              onClick={() => handleEnableWebSearch(!webSearchEnabled)}
            >
              {webSearchEnabled ? (
                <CheckIcon size={16} />
              ) : (
                <CircleSlashIcon size={16} />
              )}
              {webSearchEnabled ? t('Web search enabled') : t('Off')}
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuLabel>{t('Search depth')}</DropdownMenuLabel>
            {WEB_SEARCH_CONTEXT_OPTIONS.map((option) => (
              <DropdownMenuItem
                disabled={!canEnableWebSearch}
                key={option.value}
                onClick={() => {
                  onWebSearchContextSizeChange(option.value)
                  handleEnableWebSearch(true)
                }}
              >
                {webSearchContextSize === option.value ? (
                  <CheckIcon size={16} />
                ) : (
                  <span className='size-4' />
                )}
                {t(option.label)}
              </DropdownMenuItem>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>

        <Tooltip>
          <TooltipTrigger
            render={
              <PromptInputButton
                aria-label={t('Clear chat history')}
                className='text-muted-foreground hover:text-destructive hover:bg-destructive/10 font-medium'
                disabled={disabled || !hasMessages || !onClearMessages}
                onClick={() => setClearConfirmOpen(true)}
                variant='ghost'
              >
                <Trash2Icon size={16} />
              </PromptInputButton>
            }
          />
          <TooltipContent>
            <p>{t('Clear chat history')}</p>
          </TooltipContent>
        </Tooltip>
      </PromptInputTools>

      <ConfirmDialog
        destructive
        desc={t(
          'All playground messages saved in this browser will be removed. This cannot be undone.',
        )}
        confirmText={t('Clear')}
        handleConfirm={handleClearMessages}
        open={clearConfirmOpen}
        onOpenChange={setClearConfirmOpen}
        title={t('Clear chat history?')}
      />
    </>
  )
}
