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
import { SparklesIcon, SquareIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
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
import { Switch } from '@/components/ui/switch'
import { Slider } from '@/components/ui/slider'
import { Textarea } from '@/components/ui/textarea'
import { ModelSelector } from '@/components/model-group-selector'
import {
  AIART_ASPECT_RATIOS,
  COUNT_OPTIONS,
  detectImageModelFamily,
  GPT_IMAGE_QUALITY_OPTIONS,
  GPT_IMAGE_SIZE_PRESETS,
  HUNYUAN_IMAGE_RESOLUTIONS,
  IMAGE_FAMILY_PARAMS,
  MAX_PROMPT_LENGTH,
  QUALITY_OPTIONS,
  SIZE_PRESETS,
  supportsReferenceImages,
} from '../constants'
import type { GeneratorConfig, ModelOption } from '../types'
import { ReferenceImagesInput } from './reference-images-input'

interface GeneratorPanelProps {
  config: GeneratorConfig
  updateConfig: <K extends keyof GeneratorConfig>(
    key: K,
    value: GeneratorConfig[K]
  ) => void
  onModelChange: (value: string) => void
  models: ModelOption[]
  isModelLoading: boolean
  isGenerating: boolean
  onGenerate: () => void
  onCancel: () => void
}

export function GeneratorPanel({
  config,
  updateConfig,
  onModelChange,
  models,
  isModelLoading,
  isGenerating,
  onGenerate,
  onCancel,
}: GeneratorPanelProps) {
  const { t } = useTranslation()

  const family = useMemo(() => detectImageModelFamily(config.model), [config.model])
  const familyParams = useMemo(() => IMAGE_FAMILY_PARAMS[family], [family])
  const isDallE3 = family === 'dall-e'
  const isGptImage = family === 'gpt-image'
  const isImageGI = family === 'image-gi' || family === 'image-gi2'
  const isHunyuanImage = family === 'hunyuan-image'
  const hasRefImages = supportsReferenceImages(family)
  const canGenerate = !!config.prompt.trim() && !isGenerating

  const updateMeta = (key: string, value: unknown) => {
    updateConfig('metadata', { ...config.metadata, [key]: value })
  }

  const getMetaValue = (key: string, defaultValue: unknown) =>
    config.metadata[key] ?? defaultValue

  let sizeOptions: { label: string; value: string }[] = SIZE_PRESETS.map((p) => ({
    label: `${p.ratioLabel} (${p.value})`,
    value: p.value,
  }))
  if (isHunyuanImage) {
    sizeOptions = HUNYUAN_IMAGE_RESOLUTIONS.map((p) => ({
      label: p.label,
      value: p.value,
    }))
  } else if (isImageGI) {
    sizeOptions = [...AIART_ASPECT_RATIOS]
  } else if (isGptImage) {
    sizeOptions = GPT_IMAGE_SIZE_PRESETS.map((p) => ({
      label: 'ratioLabel' in p ? `${p.ratioLabel} (${p.value})` : p.label,
      value: p.value,
    }))
  }

  // Quality options vary by family
  const qualityOptions = isGptImage ? GPT_IMAGE_QUALITY_OPTIONS : QUALITY_OPTIONS

  // Show count selector only for models that support n > 1
  const showCount = !isDallE3 && !isImageGI && !isHunyuanImage
  let sizeLabel = t('Size')
  if (isImageGI) {
    sizeLabel = t('Aspect ratio')
  } else if (isHunyuanImage) {
    sizeLabel = t('Resolution')
  }

  return (
    <div className='flex h-full flex-col'>
      {/* Title */}
      <div className='border-b px-4 py-3'>
        <h2 className='text-sm font-semibold'>{t('Image Generation')}</h2>
      </div>

      <div className='flex-1 space-y-5 overflow-y-auto p-4'>
        {/* Model */}
        <div>
          <ModelSelector
            className='w-full'
            selectedModel={config.model}
            models={models}
            onModelChange={onModelChange}
            disabled={isModelLoading || isGenerating}
          />
        </div>

        {/* Prompt */}
        <div className='space-y-2'>
          <Label className='text-sm font-medium'>{t('Prompt')}</Label>
          <Textarea
            value={config.prompt}
            onChange={(e) =>
              updateConfig('prompt', e.target.value.slice(0, MAX_PROMPT_LENGTH))
            }
            placeholder={t('Describe the image you want to create')}
            className='min-h-[120px] resize-y'
            disabled={isGenerating}
          />
        </div>

        {/* Reference images (image-gi / image-gi2 only) */}
        {hasRefImages && (
          <div className='space-y-2'>
            <Label className='text-sm font-medium'>
              {t('Reference images')}
            </Label>
            <ReferenceImagesInput
              images={config.images}
              onChange={(imgs) => updateConfig('images', imgs)}
              disabled={isGenerating}
            />
          </div>
        )}

        {/* Aspect ratio / size */}
        <div className='space-y-2'>
          <Label className='text-sm font-medium'>{sizeLabel}</Label>
          <Select
            items={sizeOptions.map((p) => ({
              value: p.value,
              label: p.label,
            }))}
            onValueChange={(v) => {
              if (v) updateConfig('size', v)
            }}
            value={config.size}
            disabled={isGenerating}
          >
            <SelectTrigger className='w-full'>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                {sizeOptions.map((p) => (
                  <SelectItem key={p.value} value={p.value}>
                    {p.label}
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
        </div>

        {/* Quality (dall-e-3 and gpt-image) */}
        {(isDallE3 || isGptImage) && (
          <div className='space-y-2'>
            <Label className='text-sm font-medium'>{t('Quality')}</Label>
            <div className='grid grid-cols-2 gap-2'>
              {qualityOptions.map((opt) => {
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

        {/* Number of images */}
        {showCount && (
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
                      onValueChange={(next) => {
                        const v = Array.isArray(next) ? next[0] : next
                        updateMeta(param.key, v)
                      }}
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
                {param.type === 'switch' && (
                  <Switch
                    checked={!!getMetaValue(param.key, param.default)}
                    onCheckedChange={(v) => updateMeta(param.key, v)}
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
            <SparklesIcon size={16} />
            {t('Generate')}
          </Button>
        )}
      </div>
    </div>
  )
}
