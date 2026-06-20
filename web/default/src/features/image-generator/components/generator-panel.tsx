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
import { SparklesIcon, SquareIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { ModelGroupSelector } from '@/components/model-group-selector'
import {
  COUNT_OPTIONS,
  EXAMPLE_PROMPTS,
  MAX_PROMPT_LENGTH,
  QUALITY_OPTIONS,
  SIZE_PRESETS,
} from '../constants'
import type { GeneratorConfig, GroupOption, ModelOption } from '../types'

interface GeneratorPanelProps {
  config: GeneratorConfig
  updateConfig: <K extends keyof GeneratorConfig>(
    key: K,
    value: GeneratorConfig[K]
  ) => void
  models: ModelOption[]
  groups: GroupOption[]
  isModelLoading: boolean
  isGenerating: boolean
  onGenerate: () => void
  onCancel: () => void
}

export function GeneratorPanel({
  config,
  updateConfig,
  models,
  groups,
  isModelLoading,
  isGenerating,
  onGenerate,
  onCancel,
}: GeneratorPanelProps) {
  const { t } = useTranslation()

  const isDallE3 = config.model.startsWith('dall-e-3')
  const canGenerate = !!config.prompt.trim() && !isGenerating

  return (
    <div className='flex h-full flex-col'>
      <div className='flex-1 space-y-5 overflow-y-auto p-4'>
        {/* Model + group */}
        <div className='space-y-2'>
          <Label className='text-sm font-medium'>{t('Model')}</Label>
          <ModelGroupSelector
            className='w-full'
            selectedModel={config.model}
            models={models}
            onModelChange={(value) => updateConfig('model', value)}
            selectedGroup={config.group}
            groups={groups}
            onGroupChange={(value) => updateConfig('group', value)}
            disabled={isModelLoading}
          />
        </div>

        {/* Prompt */}
        <div className='space-y-2'>
          <div className='flex items-center justify-between'>
            <Label className='text-sm font-medium'>{t('Prompt')}</Label>
            <span className='text-muted-foreground text-xs'>
              {config.prompt.length}/{MAX_PROMPT_LENGTH}
            </span>
          </div>
          <Textarea
            value={config.prompt}
            onChange={(e) =>
              updateConfig('prompt', e.target.value.slice(0, MAX_PROMPT_LENGTH))
            }
            placeholder={t('Describe the image you want to create...')}
            className='min-h-[120px] resize-y'
            disabled={isGenerating}
          />
          <div className='flex flex-wrap gap-1.5'>
            {EXAMPLE_PROMPTS.map((example) => (
              <button
                key={example}
                type='button'
                disabled={isGenerating}
                onClick={() => updateConfig('prompt', example)}
                className='text-muted-foreground hover:bg-accent hover:text-foreground rounded-full border px-2.5 py-1 text-xs transition-colors disabled:opacity-50'
              >
                {example.length > 28 ? `${example.slice(0, 28)}…` : example}
              </button>
            ))}
          </div>
        </div>

        {/* Aspect ratio / size */}
        <div className='space-y-2'>
          <Label className='text-sm font-medium'>{t('Aspect ratio')}</Label>
          <div className='grid grid-cols-4 gap-2'>
            {SIZE_PRESETS.map((preset) => {
              const active = config.size === preset.value
              return (
                <button
                  key={preset.value}
                  type='button'
                  disabled={isGenerating}
                  onClick={() => updateConfig('size', preset.value)}
                  className={cn(
                    'flex flex-col items-center gap-1.5 rounded-lg border p-2 transition-colors disabled:opacity-50',
                    active
                      ? 'border-primary bg-primary/5'
                      : 'hover:border-muted-foreground/40'
                  )}
                >
                  <span
                    className={cn(
                      'rounded-sm border',
                      active ? 'border-primary' : 'border-muted-foreground/50'
                    )}
                    style={{
                      width: preset.ratio >= 1 ? 24 : 24 * preset.ratio,
                      height: preset.ratio >= 1 ? 24 / preset.ratio : 24,
                    }}
                  />
                  <span className='text-[11px] leading-none'>
                    {preset.ratioLabel}
                  </span>
                </button>
              )
            })}
          </div>
          <p className='text-muted-foreground text-xs'>{config.size}</p>
        </div>

        {/* Quality (dall-e-3 only) */}
        {isDallE3 && (
          <div className='space-y-2'>
            <Label className='text-sm font-medium'>{t('Quality')}</Label>
            <div className='grid grid-cols-2 gap-2'>
              {QUALITY_OPTIONS.map((opt) => {
                const active = config.quality === opt.value
                return (
                  <button
                    key={opt.value}
                    type='button'
                    disabled={isGenerating}
                    onClick={() => updateConfig('quality', opt.value)}
                    className={cn(
                      'rounded-lg border py-2 text-sm transition-colors disabled:opacity-50',
                      active
                        ? 'border-primary bg-primary/5'
                        : 'hover:border-muted-foreground/40'
                    )}
                  >
                    {t(opt.label)}
                  </button>
                )
              })}
            </div>
          </div>
        )}

        {/* Number of images (hidden for dall-e-3 which only supports 1) */}
        {!isDallE3 && (
          <div className='space-y-2'>
            <Label className='text-sm font-medium'>
              {t('Number of images')}
            </Label>
            <div className='grid grid-cols-4 gap-2'>
              {COUNT_OPTIONS.map((count) => {
                const active = config.n === count
                return (
                  <button
                    key={count}
                    type='button'
                    disabled={isGenerating}
                    onClick={() => updateConfig('n', count)}
                    className={cn(
                      'rounded-lg border py-2 text-sm transition-colors disabled:opacity-50',
                      active
                        ? 'border-primary bg-primary/5'
                        : 'hover:border-muted-foreground/40'
                    )}
                  >
                    {count}
                  </button>
                )
              })}
            </div>
          </div>
        )}
      </div>

      {/* Sticky generate button */}
      <div className='border-t p-4'>
        {isGenerating ? (
          <Button
            variant='secondary'
            className='w-full'
            onClick={onCancel}
            size='lg'
          >
            <SquareIcon className='fill-current' size={16} />
            {t('Stop')}
          </Button>
        ) : (
          <Button
            className='w-full'
            onClick={onGenerate}
            disabled={!canGenerate}
            size='lg'
          >
            <SparklesIcon size={16} />
            {t('Generate')}
          </Button>
        )}
      </div>
    </div>
  )
}
