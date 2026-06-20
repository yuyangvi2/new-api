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
import { FilmIcon, SquareIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { ModelGroupSelector } from '@/components/model-group-selector'
import {
  MAX_PROMPT_LENGTH,
  VIDEO_DURATIONS,
  VIDEO_SIZE_PRESETS,
} from '../constants'
import type {
  GroupOption,
  ImageSourceType,
  ModelOption,
  VideoConfig,
} from '../types'
import { ImageSourceInput } from './image-source-input'

interface VideoPanelProps {
  config: VideoConfig
  updateConfig: <K extends keyof VideoConfig>(
    key: K,
    value: VideoConfig[K]
  ) => void
  models: ModelOption[]
  groups: GroupOption[]
  isModelLoading: boolean
  isGenerating: boolean
  availableImages: { id: string; src: string }[]
  onGenerate: () => void
  onCancel: () => void
}

export function VideoPanel({
  config,
  updateConfig,
  models,
  groups,
  isModelLoading,
  isGenerating,
  availableImages,
  onGenerate,
  onCancel,
}: VideoPanelProps) {
  const { t } = useTranslation()

  const canGenerate = !!config.image && !isGenerating

  const handleImageChange = (src: string, sourceType: ImageSourceType) => {
    updateConfig('image', src)
    updateConfig('imageSourceType', sourceType)
  }

  return (
    <div className='flex h-full flex-col'>
      <div className='flex-1 space-y-5 overflow-y-auto p-4'>
        {/* Input image */}
        <div className='space-y-2'>
          <Label className='text-sm font-medium'>{t('Input image')}</Label>
          <ImageSourceInput
            value={config.image}
            sourceType={config.imageSourceType}
            onChange={handleImageChange}
            availableImages={availableImages}
            disabled={isGenerating}
          />
        </div>

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

        {/* Motion prompt (optional) */}
        <div className='space-y-2'>
          <div className='flex items-center justify-between'>
            <Label className='text-sm font-medium'>
              {t('Motion prompt (optional)')}
            </Label>
            <span className='text-muted-foreground text-xs'>
              {config.prompt.length}/{MAX_PROMPT_LENGTH}
            </span>
          </div>
          <Textarea
            value={config.prompt}
            onChange={(e) =>
              updateConfig('prompt', e.target.value.slice(0, MAX_PROMPT_LENGTH))
            }
            placeholder={t('Describe how the image should move...')}
            className='min-h-[80px] resize-y'
            disabled={isGenerating}
          />
        </div>

        {/* Duration */}
        <div className='space-y-2'>
          <Label className='text-sm font-medium'>{t('Duration (sec)')}</Label>
          <div className='grid grid-cols-2 gap-2'>
            {VIDEO_DURATIONS.map((d) => {
              const active = config.duration === d
              return (
                <button
                  key={d}
                  type='button'
                  disabled={isGenerating}
                  onClick={() => updateConfig('duration', d)}
                  className={cn(
                    'rounded-lg border py-2 text-sm transition-colors disabled:opacity-50',
                    active
                      ? 'border-primary bg-primary/5'
                      : 'hover:border-muted-foreground/40'
                  )}
                >
                  {d}s
                </button>
              )
            })}
          </div>
        </div>

        {/* Aspect ratio */}
        <div className='space-y-2'>
          <Label className='text-sm font-medium'>{t('Aspect ratio')}</Label>
          <div className='grid grid-cols-3 gap-2'>
            {VIDEO_SIZE_PRESETS.map((preset) => {
              const sizeValue = `${preset.width}x${preset.height}`
              const active = config.size === sizeValue
              return (
                <button
                  key={sizeValue}
                  type='button'
                  disabled={isGenerating}
                  onClick={() => updateConfig('size', sizeValue)}
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
        </div>
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
            <FilmIcon size={16} />
            {t('Generate video')}
          </Button>
        )}
      </div>
    </div>
  )
}
