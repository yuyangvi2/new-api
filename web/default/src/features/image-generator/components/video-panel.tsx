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
import { useEffect, useMemo } from 'react'
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
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { ModelSelector } from '@/components/model-group-selector'
import {
  detectModelFamily,
  FAMILY_PARAMS,
  getUsableVideoImage,
  getVideoModelVariantState,
  MAX_PROMPT_LENGTH,
  resolveVideoVariantModel,
  VIDEO_DURATIONS,
  videoModelAllowsImageUpload,
  videoModelRequiresImage,
  videoModelSupportsImageInput,
  VIDEO_SIZE_PRESETS,
} from '../constants'
import type {
  FamilyParam,
  ImageSourceType,
  ModelOption,
  VideoConfig,
  VideoModelVariantAxisState,
} from '../types'
import { ImageSourceInput } from './image-source-input'

interface VideoPanelProps {
  config: VideoConfig
  updateConfig: <K extends keyof VideoConfig>(
    key: K,
    value: VideoConfig[K]
  ) => void
  onModelChange: (value: string) => void
  models: ModelOption[]
  variantModels: ModelOption[]
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
  variantModels,
  isModelLoading,
  isGenerating,
  availableImages,
  onGenerate,
  onCancel,
}: VideoPanelProps) {
  const { t } = useTranslation()

  const family = useMemo(() => detectModelFamily(config.model), [config.model])
  const familyParams = useMemo(() => FAMILY_PARAMS[family], [family])
  const availableVariantModels = useMemo(
    () => variantModels.map((m) => m.value),
    [variantModels]
  )
  const variantState = useMemo(
    () => getVideoModelVariantState(config.model, availableVariantModels),
    [config.model, availableVariantModels]
  )
  const displayModel = variantState?.set.defaultModel ?? config.model

  const requiresImage = videoModelRequiresImage(config.model)
  const showImageInput = videoModelSupportsImageInput(config.model)
  const allowImageUpload = videoModelAllowsImageUpload(config.model)
  const usableImage = getUsableVideoImage(config.model, config.image)
  const canGenerate = (!requiresImage || !!usableImage) && !isGenerating
  const resolutionParam = familyParams.find(
    (param) => param.key === 'resolution' && param.type === 'select'
  )
  const resolutionOptions =
    resolutionParam?.options?.filter(
      (option) =>
        !/fast/i.test(config.model) ||
        option.value === '480p' ||
        option.value === '720p'
    ) ?? []
  const advancedParams = familyParams.filter(
    (param) => param.key !== 'resolution'
  )

  const updateMeta = (key: string, value: unknown) => {
    updateConfig('metadata', { ...config.metadata, [key]: value })
  }

  const getMetaValue = (key: string, defaultValue: unknown) =>
    config.metadata[key] ?? defaultValue

  const handleImageChange = (src: string, sourceType: ImageSourceType) => {
    updateConfig('image', src)
    updateConfig('imageSourceType', sourceType)
  }

  const handleVariantChange = (axisId: string, value: string) => {
    const nextModel = resolveVideoVariantModel(
      config.model,
      axisId,
      value,
      availableVariantModels
    )
    if (nextModel && nextModel !== config.model) {
      onModelChange(nextModel)
    }
  }

  useEffect(() => {
    if (!/seedance/i.test(config.model) || !/fast/i.test(config.model)) return
    const resolution = String(config.metadata.resolution ?? '720p')
    if (resolution !== '480p' && resolution !== '720p') {
      updateConfig('metadata', { ...config.metadata, resolution: '720p' })
    }
  }, [config.metadata, config.model, updateConfig])

  useEffect(() => {
    if (!config.image) return
    if (getUsableVideoImage(config.model, config.image)) return
    updateConfig('image', '')
    updateConfig('imageSourceType', 'upload')
  }, [config.image, config.model, updateConfig])

  return (
    <div className='flex h-full flex-col'>
      {/* Title */}
      <div className='border-b px-4 py-3'>
        <h2 className='text-sm font-semibold'>{t('Video Generation')}</h2>
      </div>

      <div className='flex-1 space-y-5 overflow-y-auto p-4'>
        {/* Model */}
        <div>
          <ModelSelector
            className='w-full'
            selectedModel={displayModel}
            models={models}
            onModelChange={onModelChange}
            disabled={isModelLoading || isGenerating}
          />
        </div>

        {variantState && (
          <div className='grid grid-cols-2 items-center gap-1'>
            {variantState.axes.map((axis) => (
              <SegmentedTabs
                key={axis.id}
                axis={axis}
                disabled={isGenerating}
                onValueChange={(value) => handleVariantChange(axis.id, value)}
              />
            ))}
          </div>
        )}

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

        {showImageInput && (
          <div className='space-y-2'>
            <Label className='text-sm font-medium'>{t('Input image')}</Label>
            <ImageSourceInput
              value={config.image}
              sourceType={config.imageSourceType}
              onChange={handleImageChange}
              availableImages={availableImages}
              disabled={isGenerating}
              allowUpload={allowImageUpload}
            />
          </div>
        )}

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

        <div className='grid grid-cols-2 gap-2'>
          <div className='min-w-0 space-y-2'>
            <Label className='text-sm font-medium'>{t('Aspect ratio')}</Label>
            <Select
              items={VIDEO_SIZE_PRESETS.map((p) => ({
                value: `${p.width}x${p.height}`,
                label: `${p.ratioLabel} (${p.width}x${p.height})`,
              }))}
              onValueChange={(v) => {
                if (v) updateConfig('size', v)
              }}
              value={config.size}
              disabled={isGenerating}
            >
              <SelectTrigger className='w-full min-w-0 [&>span]:truncate'>
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

          {resolutionParam && (
            <div className='min-w-0 space-y-2'>
              <Label className='text-sm font-medium'>
                {t(resolutionParam.label)}
              </Label>
              <Select
                items={resolutionOptions.map((option) => ({
                  value: option.value,
                  label: option.label,
                }))}
                onValueChange={(value) => updateMeta(resolutionParam.key, value)}
                value={String(
                  getMetaValue(resolutionParam.key, resolutionParam.default)
                )}
                disabled={isGenerating}
              >
                <SelectTrigger className='w-full min-w-0 [&>span]:truncate'>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    {resolutionOptions.map((option) => (
                      <SelectItem key={option.value} value={option.value}>
                        {option.label}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
            </div>
          )}
        </div>

        {/* Model-family specific parameters */}
        {advancedParams.length > 0 && (
          <>
            <div className='border-muted-foreground/20 border-t pt-4'>
              <Label className='text-muted-foreground text-xs font-medium uppercase tracking-wide'>
                {t('Advanced')}
              </Label>
            </div>
            <div className='flex flex-wrap items-center gap-2'>
              {advancedParams
                .filter((param) => param.type === 'select' && param.options)
                .map((param) => (
                  <SegmentedParam
                    key={param.key}
                    param={param}
                    model={config.model}
                    value={String(getMetaValue(param.key, param.default))}
                    disabled={isGenerating}
                    onValueChange={(value) => updateMeta(param.key, value)}
                  />
                ))}
            </div>
            {advancedParams
              .filter((param) => param.type !== 'select')
              .map((param) => (
                <div key={param.key} className='space-y-2'>
                  <Label className='text-sm font-medium'>{t(param.label)}</Label>
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

function SegmentedTabs({
  axis,
  disabled,
  onValueChange,
}: {
  axis: VideoModelVariantAxisState
  disabled: boolean
  onValueChange: (value: string) => void
}) {
  const { t } = useTranslation()

  return (
    <Tabs
      value={axis.value}
      onValueChange={onValueChange}
      className='min-w-0 !flex-row gap-0'
    >
      <TabsList className='bg-background h-7 w-full overflow-hidden rounded-none border p-0'>
        {axis.options.map((option, index) => (
          <TabsTrigger
            key={option.value}
            value={option.value}
            disabled={disabled}
            className='h-7 min-w-0 flex-1 rounded-none border-r px-1.5 text-[11px] leading-none shadow-none last:border-r-0 data-active:bg-muted data-active:shadow-none'
            data-index={index}
          >
            {axis.translateLabels === false ? option.label : t(option.label)}
          </TabsTrigger>
        ))}
      </TabsList>
    </Tabs>
  )
}

function SegmentedParam({
  param,
  model,
  value,
  disabled,
  onValueChange,
}: {
  param: FamilyParam
  model: string
  value: string
  disabled: boolean
  onValueChange: (value: string) => void
}) {
  const { t } = useTranslation()
  if (!param.options || param.options.length === 0) return null
  const options =
    param.key === 'resolution' && /fast/i.test(model)
      ? param.options.filter(
          (option) => option.value === '480p' || option.value === '720p'
        )
      : param.options

  return (
    <Tabs value={value} onValueChange={onValueChange} className='!flex-row gap-0'>
      <TabsList className='bg-background h-9 overflow-hidden rounded-lg border p-0'>
        {options.map((option) => (
          <TabsTrigger
            key={option.value}
            value={option.value}
            disabled={disabled}
            className='h-9 min-w-20 rounded-none border-r px-4 text-sm shadow-none last:border-r-0 data-active:bg-muted data-active:shadow-none'
          >
            {param.key === 'resolution' ? option.label : t(option.label)}
          </TabsTrigger>
        ))}
      </TabsList>
    </Tabs>
  )
}
