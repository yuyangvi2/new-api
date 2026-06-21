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
import { useMemo } from 'react'
import { FilmIcon, SquareIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Slider } from '@/components/ui/slider'
import { Textarea } from '@/components/ui/textarea'
import { ModelSelector } from '@/components/model-group-selector'
import {
  detectModelFamily,
  FAMILY_PARAMS,
  MAX_PROMPT_LENGTH,
  VIDEO_DURATIONS,
  VIDEO_SIZE_PRESETS,
} from '../constants'
import type { ImageSourceType, ModelOption, VideoConfig } from '../types'
import { ImageSourceInput } from './image-source-input'

interface VideoPanelProps {
  config: VideoConfig
  updateConfig: <K extends keyof VideoConfig>(
    key: K,
    value: VideoConfig[K]
  ) => void
  onModelChange: (value: string) => void
  models: ModelOption[]
  isModelLoading: boolean
  isGenerating: boolean
  availableImages: { id: string; src: string }[]
  onGenerate: () => void
  onCancel: () => void
}

export function VideoPanel({
  config,
  updateConfig,
  onModelChange,
  models,
  isModelLoading,
  isGenerating,
  availableImages,
  onGenerate,
  onCancel,
}: VideoPanelProps) {
  const { t } = useTranslation()

  const family = useMemo(() => detectModelFamily(config.model), [config.model])
  const familyParams = useMemo(() => FAMILY_PARAMS[family], [family])

  const canGenerate = !!config.image && !isGenerating

  const updateMeta = (key: string, value: unknown) => {
    updateConfig('metadata', { ...config.metadata, [key]: value })
  }

  const getMetaValue = (key: string, defaultValue: unknown) =>
    config.metadata[key] ?? defaultValue

  const handleImageChange = (src: string, sourceType: ImageSourceType) => {
    updateConfig('image', src)
    updateConfig('imageSourceType', sourceType)
  }

  return (
    <div className='flex h-full flex-col'>
      {/* Title */}
      <div className='border-b px-4 py-3'>
        <h2 className='text-sm font-semibold'>{t('Video Generation')}</h2>
      </div>

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

        {/* Model */}
        <div className='space-y-2'>
          <Label className='text-sm font-medium'>{t('Model')}</Label>
          <ModelSelector
            className='w-full'
            selectedModel={config.model}
            models={models}
            onModelChange={onModelChange}
            disabled={isModelLoading}
          />
        </div>

        {/* Motion prompt (optional) */}
        <div className='space-y-2'>
          <Label className='text-sm font-medium'>
            {t('Motion prompt (optional)')}
          </Label>
          <Textarea
            value={config.prompt}
            onChange={(e) =>
              updateConfig('prompt', e.target.value.slice(0, MAX_PROMPT_LENGTH))
            }
            placeholder={t('Describe how the image should move')}
            className='min-h-[80px] resize-y'
            disabled={isGenerating}
          />
        </div>

        {/* Duration */}
        <div className='space-y-2'>
          <Label className='text-sm font-medium'>{t('Duration (sec)')}</Label>
          <Select
            items={VIDEO_DURATIONS.map((d) => ({
              value: String(d),
              label: `${d}s`,
            }))}
            onValueChange={(v) => updateConfig('duration', Number(v))}
            value={String(config.duration)}
            disabled={isGenerating}
          >
            <SelectTrigger className='w-full'>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                {VIDEO_DURATIONS.map((d) => (
                  <SelectItem key={d} value={String(d)}>
                    {d}s
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
        </div>

        {/* Aspect ratio */}
        <div className='space-y-2'>
          <Label className='text-sm font-medium'>{t('Aspect ratio')}</Label>
          <Select
            items={VIDEO_SIZE_PRESETS.map((p) => ({
              value: `${p.width}x${p.height}`,
              label: `${p.ratioLabel} (${p.width}x${p.height})`,
            }))}
            onValueChange={(v) => updateConfig('size', v)}
            value={config.size}
            disabled={isGenerating}
          >
            <SelectTrigger className='w-full'>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                {VIDEO_SIZE_PRESETS.map((p) => {
                  const val = `${p.width}x${p.height}`
                  return (
                    <SelectItem key={val} value={val}>
                      {p.ratioLabel} ({val})
                    </SelectItem>
                  )
                })}
              </SelectGroup>
            </SelectContent>
          </Select>
        </div>

        {/* Model-family specific parameters */}
        {familyParams.length > 0 && (
          <>
            <div className='border-muted-foreground/20 border-t pt-4'>
              <Label className='text-muted-foreground text-xs font-medium uppercase tracking-wide'>
                {t('Advanced')}
              </Label>
            </div>
            {familyParams.map((param) => (
              <div key={param.key} className='space-y-2'>
                <Label className='text-sm font-medium'>{t(param.label)}</Label>
                {param.type === 'select' && param.options && (
                  <Select
                    items={param.options.map((o) => ({
                      value: o.value,
                      label: t(o.label),
                    }))}
                    onValueChange={(v) => updateMeta(param.key, v)}
                    value={String(getMetaValue(param.key, param.default))}
                    disabled={isGenerating}
                  >
                    <SelectTrigger className='w-full'>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectGroup>
                        {param.options.map((o) => (
                          <SelectItem key={o.value} value={o.value}>
                            {t(o.label)}
                          </SelectItem>
                        ))}
                      </SelectGroup>
                    </SelectContent>
                  </Select>
                )}
                {param.type === 'slider' && (
                  <div className='flex items-center gap-3'>
                    <Slider
                      min={param.min ?? 0}
                      max={param.max ?? 1}
                      step={param.step ?? 0.1}
                      value={[Number(getMetaValue(param.key, param.default))]}
                      onValueChange={([v]) => updateMeta(param.key, v)}
                      disabled={isGenerating}
                      className='flex-1'
                    />
                    <span className='text-muted-foreground w-8 text-right text-xs'>
                      {Number(getMetaValue(param.key, param.default)).toFixed(1)}
                    </span>
                  </div>
                )}
                {param.type === 'text' && (
                  <Textarea
                    value={String(getMetaValue(param.key, param.default))}
                    onChange={(e) => updateMeta(param.key, e.target.value)}
                    placeholder={t(param.label)}
                    className='min-h-[60px] resize-y'
                    disabled={isGenerating}
                  />
                )}
              </div>
            ))}
          </>
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
            <FilmIcon size={16} />
            {t('Generate video')}
          </Button>
        )}
      </div>
    </div>
  )
}
