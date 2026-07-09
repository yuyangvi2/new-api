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
import { useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  CircleHelp,
  FilmIcon,
  LinkIcon,
  Loader2Icon,
  MusicIcon,
  PlusIcon,
  SquareIcon,
  Trash2Icon,
  UploadIcon,
} from 'lucide-react'
import { toast } from 'sonner'

import { ModelSelector } from '@/components/model-group-selector'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
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
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

import { uploadReferenceMedia } from '../api'
import {
  detectModelFamily,
  estimateSeedanceVideoTokens,
  FAMILY_PARAMS,
  getSeedancePricePerMillionCNY,
  getSeedanceOutputSize,
  getSeedanceResolutionOptions,
  getUsableVideoImage,
  getVideoModelVariantState,
  isSeedanceVideoModel,
  MAX_PROMPT_LENGTH,
  MAX_REFERENCE_AUDIO_UPLOAD_BYTES,
  MAX_REFERENCE_VIDEO_UPLOAD_BYTES,
  MAX_IMAGE_UPLOAD_BYTES,
  resolveVideoVariantModel,
  SEEDANCE_REFERENCE_AUDIO_LIMIT,
  SEEDANCE_REFERENCE_IMAGE_LIMIT,
  SEEDANCE_REFERENCE_VIDEO_LIMIT,
  SEEDANCE_REFERENCE_VIDEO_DURATION_LIMIT,
  SEEDANCE_VIDEO_DURATIONS,
  SEEDANCE_VIDEO_RATIO_PRESETS,
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

type ReferenceMediaKind = 'image' | 'video' | 'audio'

interface VideoPanelProps {
  config: VideoConfig
  updateConfig: <K extends keyof VideoConfig>(
    key: K,
    value: VideoConfig[K]
  ) => void
  onModelChange: (value: string) => void
  models: ModelOption[]
  variantModels: ModelOption[]
  groupRatio?: number
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
  groupRatio,
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

  const isSeedanceVideo = isSeedanceVideoModel(config.model)
  const requiresImage = videoModelRequiresImage(config.model)
  const showImageInput = videoModelSupportsImageInput(config.model)
  const allowImageUpload = videoModelAllowsImageUpload(config.model)
  const usableImage = getUsableVideoImage(config.model, config.image)
  const referenceVideoDurations = config.referenceVideoDurations ?? {}
  const updateMeta = (key: string, value: unknown) => {
    updateConfig('metadata', { ...config.metadata, [key]: value })
  }
  const getMetaValue = (key: string, defaultValue: unknown) =>
    config.metadata[key] ?? defaultValue
  const referenceImages = parseURLList(config.referenceImagesText)
  const referenceVideos = parseURLList(config.referenceVideosText)
  const referenceAudios = parseURLList(config.referenceAudiosText)
  const uploadedReferenceVideoDuration = sumVideoDurations(
    normalizeVideoDurationMap(referenceVideoDurations, referenceVideos)
  )
  const effectiveInputVideoDuration = Math.max(
    config.inputVideoDuration,
    uploadedReferenceVideoDuration
  )
  const hasReferenceMedia =
    !!usableImage || referenceImages.length > 0 || referenceVideos.length > 0
  const hasTooManyReferenceImages =
    referenceImages.length > SEEDANCE_REFERENCE_IMAGE_LIMIT
  const hasTooManyReferenceVideos =
    referenceVideos.length > SEEDANCE_REFERENCE_VIDEO_LIMIT
  const hasTooManyReferenceAudios =
    referenceAudios.length > SEEDANCE_REFERENCE_AUDIO_LIMIT
  const hasInvalidAudioOnly = referenceAudios.length > 0 && !hasReferenceMedia
  const hasInvalidInputVideoDuration =
    referenceVideos.length > 0 &&
    (effectiveInputVideoDuration < 0 ||
      effectiveInputVideoDuration > SEEDANCE_REFERENCE_VIDEO_DURATION_LIMIT)
  const referenceImageError = hasTooManyReferenceImages
    ? t('Maximum {{count}} URLs', {
        count: SEEDANCE_REFERENCE_IMAGE_LIMIT,
      })
    : undefined
  const referenceVideoError = hasTooManyReferenceVideos
    ? t('Maximum {{count}} URLs', {
        count: SEEDANCE_REFERENCE_VIDEO_LIMIT,
      })
    : undefined
  let referenceAudioError: string | undefined
  if (hasTooManyReferenceAudios) {
    referenceAudioError = t('Maximum {{count}} URLs', {
      count: SEEDANCE_REFERENCE_AUDIO_LIMIT,
    })
  } else if (hasInvalidAudioOnly) {
    referenceAudioError = t(
      'Audio references require at least one image or video reference.'
    )
  }
  const canGenerate =
    (!requiresImage || !!usableImage) &&
    !hasInvalidAudioOnly &&
    !hasTooManyReferenceImages &&
    !hasTooManyReferenceVideos &&
    !hasTooManyReferenceAudios &&
    !hasInvalidInputVideoDuration &&
    !isGenerating
  const resolutionParam = familyParams.find(
    (param) => param.key === 'resolution' && param.type === 'select'
  )
  const resolutionOptions = useMemo(
    () =>
      isSeedanceVideo
        ? getSeedanceResolutionOptions(config.model)
        : (resolutionParam?.options ?? []),
    [config.model, isSeedanceVideo, resolutionParam?.options]
  )
  const durationOptions = isSeedanceVideo
    ? SEEDANCE_VIDEO_DURATIONS
    : VIDEO_DURATIONS
  const advancedParams = familyParams.filter(
    (param) => param.key !== 'resolution'
  )
  const selectedResolution = String(
    getMetaValue('resolution', resolutionParam?.default ?? '720p')
  )
  const selectedRatio = isSeedanceVideo ? config.size : '16:9'
  const seedanceOutputSize = getSeedanceOutputSize(
    selectedRatio,
    selectedResolution
  )
  const estimatedVideoTokens = isSeedanceVideo
    ? estimateSeedanceVideoTokens({
        inputDuration: effectiveInputVideoDuration,
        outputDuration: config.duration,
        ratio: selectedRatio,
        resolution: selectedResolution,
      })
    : 0
  const pricePerMillionCNY = getSeedancePricePerMillionCNY({
    model: config.model,
    resolution: selectedResolution,
    hasVideoInput: referenceVideos.length > 0,
  })
  const estimatedVideoCostCNY =
    estimatedVideoTokens > 0 && pricePerMillionCNY !== undefined
      ? (estimatedVideoTokens * pricePerMillionCNY * (groupRatio ?? 1)) /
        1_000_000
      : 0
  const hasEstimatedVideoCost =
    estimatedVideoTokens > 0 && pricePerMillionCNY !== undefined
  const generateLabel = hasEstimatedVideoCost
    ? t('Generate video · {{cost}}', {
        cost: formatCNYAmount(estimatedVideoCostCNY),
      })
    : t('Generate video')

  const handleImageChange = (src: string, sourceType: ImageSourceType) => {
    updateConfig('image', src)
    updateConfig('imageSourceType', sourceType)
  }

  const updateReferenceVideoDurations = (
    nextDurations: Record<string, number>,
    nextVideos: string[]
  ) => {
    const normalized = normalizeVideoDurationMap(nextDurations, nextVideos)
    updateConfig('referenceVideoDurations', normalized)
    updateConfig('inputVideoDuration', sumVideoDurations(normalized))
  }

  const handleReferenceVideosChange = (value: string) => {
    const nextVideos = parseURLList(value)
    updateConfig('referenceVideosText', value)
    updateReferenceVideoDurations(referenceVideoDurations, nextVideos)
  }

  const handleUploadedReferenceVideoDuration = (
    url: string,
    duration: number
  ) => {
    updateReferenceVideoDurations(
      { ...referenceVideoDurations, [url]: duration },
      referenceVideos.includes(url)
        ? referenceVideos
        : [...referenceVideos, url]
    )
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
    if (isSeedanceVideo) {
      if (
        !SEEDANCE_VIDEO_RATIO_PRESETS.some(
          (ratio) => ratio.value === config.size
        )
      ) {
        updateConfig('size', '16:9')
      }
      const currentResolution = String(config.metadata.resolution ?? '')
      const nextResolution = resolutionOptions.some(
        (option) => option.value === currentResolution
      )
        ? currentResolution
        : (resolutionOptions[0]?.value ?? '720p')
      if (currentResolution !== nextResolution) {
        updateConfig('metadata', {
          ...config.metadata,
          resolution: nextResolution,
        })
      }
      return
    }
    if (config.size.includes(':')) {
      updateConfig('size', '1280x720')
    }
  }, [
    config.metadata,
    config.size,
    isSeedanceVideo,
    resolutionOptions,
    updateConfig,
  ])

  useEffect(() => {
    if (!config.image) return
    if (getUsableVideoImage(config.model, config.image)) return
    updateConfig('image', '')
    updateConfig('imageSourceType', 'upload')
  }, [config.image, config.model, updateConfig])

  useEffect(() => {
    if (uploadedReferenceVideoDuration <= config.inputVideoDuration) return
    updateConfig('inputVideoDuration', uploadedReferenceVideoDuration)
  }, [config.inputVideoDuration, uploadedReferenceVideoDuration, updateConfig])

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

        {isSeedanceVideo && (
          <>
            <ReferenceMediaField
              label={t('Reference images')}
              kind='image'
              value={config.referenceImagesText}
              onChange={(value) => updateConfig('referenceImagesText', value)}
              disabled={isGenerating}
              placeholder={t('Paste image URL')}
              accept='image/png,image/jpeg,image/gif,image/bmp,image/webp'
              maxItems={SEEDANCE_REFERENCE_IMAGE_LIMIT}
              maxBytes={MAX_IMAGE_UPLOAD_BYTES}
              error={referenceImageError}
              hint={t(
                'Image URL array (up to 9). Supports png/jpg/jpeg/gif/bmp/webp, minimum 300px width and height. Use 【@图片N】 in the prompt to reference the image at that position.'
              )}
            />
            <ReferenceMediaField
              label={t('Reference videos')}
              kind='video'
              value={config.referenceVideosText}
              onChange={handleReferenceVideosChange}
              disabled={isGenerating}
              placeholder={t('Paste video URL')}
              accept='video/mp4,video/quicktime,video/x-msvideo,video/x-matroska,video/webm,video/x-flv,.mkv,.flv'
              maxItems={SEEDANCE_REFERENCE_VIDEO_LIMIT}
              maxBytes={MAX_REFERENCE_VIDEO_UPLOAD_BYTES}
              videoDurations={referenceVideoDurations}
              maxTotalVideoDuration={SEEDANCE_REFERENCE_VIDEO_DURATION_LIMIT}
              onVideoDurationChange={handleUploadedReferenceVideoDuration}
              error={referenceVideoError}
              hint={t(
                'Reference video URL array (up to 3). Supports mp4/mov/avi/mkv/webm/flv. Each video must be 2-15.5 seconds, total duration up to 15.5 seconds, up to 50MB, 24-60 FPS.'
              )}
            />
            <ReferenceMediaField
              label={t('Reference audios')}
              kind='audio'
              value={config.referenceAudiosText}
              onChange={(value) => updateConfig('referenceAudiosText', value)}
              disabled={isGenerating}
              placeholder={t('Paste audio URL')}
              accept='audio/mpeg,audio/wav,audio/x-wav,.mp3,.wav'
              maxItems={SEEDANCE_REFERENCE_AUDIO_LIMIT}
              maxBytes={MAX_REFERENCE_AUDIO_UPLOAD_BYTES}
              error={referenceAudioError}
              hint={t(
                'Reference audio URL array (up to 3). Supports mp3/wav only. Cannot be used alone; use with images or videos. Total size up to 15MB, each audio 2-15 seconds.'
              )}
            />
            {referenceVideos.length > 0 && (
              <div className='space-y-2'>
                <Label className='text-sm font-medium'>
                  {t('Input video duration (sec)')}
                </Label>
                <Input
                  type='number'
                  min={uploadedReferenceVideoDuration}
                  max={SEEDANCE_REFERENCE_VIDEO_DURATION_LIMIT}
                  step={0.5}
                  value={String(config.inputVideoDuration || '')}
                  onChange={(event) =>
                    updateConfig(
                      'inputVideoDuration',
                      Math.max(
                        Number(event.target.value) || 0,
                        uploadedReferenceVideoDuration
                      )
                    )
                  }
                  disabled={isGenerating}
                />
                {hasInvalidInputVideoDuration && (
                  <p className='text-destructive text-xs'>
                    {t('Input video duration must be 0-15.5 seconds.')}
                  </p>
                )}
              </div>
            )}
          </>
        )}

        {/* Duration */}
        <div className='space-y-2'>
          <Label className='text-sm font-medium'>{t('Duration (sec)')}</Label>
          <Select
            items={durationOptions.map((d) => ({
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
                {durationOptions.map((d) => (
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
              items={
                isSeedanceVideo
                  ? SEEDANCE_VIDEO_RATIO_PRESETS.map((ratio) => ({
                      value: ratio.value,
                      label: ratio.label,
                    }))
                  : VIDEO_SIZE_PRESETS.map((p) => ({
                      value: `${p.width}x${p.height}`,
                      label: `${p.ratioLabel} (${p.width}x${p.height})`,
                    }))
              }
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
                  {isSeedanceVideo
                    ? SEEDANCE_VIDEO_RATIO_PRESETS.map((ratio) => (
                        <SelectItem key={ratio.value} value={ratio.value}>
                          {ratio.label}
                        </SelectItem>
                      ))
                    : VIDEO_SIZE_PRESETS.map((p) => {
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
                onValueChange={(value) =>
                  updateMeta(resolutionParam.key, value)
                }
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

        {isSeedanceVideo && (
          <div className='text-muted-foreground text-xs'>
            {t('Estimated output size')}: {seedanceOutputSize.longEdge}×
            {seedanceOutputSize.shortEdge}
          </div>
        )}

        {/* Model-family specific parameters */}
        {advancedParams.length > 0 && (
          <>
            <div className='border-muted-foreground/20 border-t pt-4'>
              <Label className='text-muted-foreground text-xs font-medium tracking-wide uppercase'>
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
                  <Label className='text-sm font-medium'>
                    {t(param.label)}
                  </Label>
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
                        {Number(getMetaValue(param.key, param.default)).toFixed(
                          1
                        )}
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
            {generateLabel}
          </Button>
        )}
      </div>
    </div>
  )
}

function parseURLList(value: string): string[] {
  const trimmed = value.trim()
  if (!trimmed) return []
  try {
    const parsed = JSON.parse(trimmed) as unknown
    if (Array.isArray(parsed)) {
      return parsed
        .map((item) => String(item).trim())
        .filter((item) => item.length > 0)
    }
  } catch {
    /* use line-based input */
  }
  return trimmed
    .split(/[\n,]+/)
    .map((item) => item.trim())
    .filter((item) => item.length > 0)
}

function formatCNYAmount(value: number): string {
  const formatted = new Intl.NumberFormat(undefined, {
    maximumFractionDigits: 4,
  }).format(value)
  return `¥${formatted}`
}

function ReferenceMediaField({
  label,
  kind,
  value,
  onChange,
  disabled,
  placeholder,
  accept,
  maxItems,
  maxBytes,
  videoDurations,
  maxTotalVideoDuration,
  onVideoDurationChange,
  hint,
  error,
}: {
  label: string
  kind: ReferenceMediaKind
  value: string
  onChange: (value: string) => void
  disabled: boolean
  placeholder: string
  accept: string
  maxItems: number
  maxBytes: number
  videoDurations?: Record<string, number>
  maxTotalVideoDuration?: number
  onVideoDurationChange?: (url: string, duration: number) => void
  hint: string
  error?: string
}) {
  const { t } = useTranslation()
  const fileRef = useRef<HTMLInputElement>(null)
  const [urlText, setUrlText] = useState('')
  const [isUploading, setIsUploading] = useState(false)
  const urls = parseURLList(value)
  const canAdd = urls.length < maxItems && !disabled && !isUploading
  let uploadLabel = t('Upload audio')
  if (kind === 'image') {
    uploadLabel = t('Upload image')
  } else if (kind === 'video') {
    uploadLabel = t('Upload video')
  }

  const updateUrls = (nextUrls: string[]) => {
    onChange(JSON.stringify(nextUrls))
  }

  const addUrl = (url: string) => {
    const nextUrl = url.trim()
    if (!nextUrl) return
    if (!isHttpURL(nextUrl)) {
      toast.error(t('Please enter a valid HTTP URL'))
      return
    }
    if (urls.length >= maxItems) {
      toast.error(t('Maximum {{count}} URLs', { count: maxItems }))
      return
    }
    if (urls.includes(nextUrl)) {
      setUrlText('')
      return
    }
    updateUrls([...urls, nextUrl])
    setUrlText('')
  }

  const removeUrl = (urlToRemove: string) => {
    updateUrls(urls.filter((url) => url !== urlToRemove))
  }

  const handleKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
    if (event.key !== 'Enter') return
    event.preventDefault()
    addUrl(urlText)
  }

  const handleFile = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0]
    event.target.value = ''
    if (!file) return
    if (file.size > maxBytes) {
      toast.error(
        t('File is too large (max {{size}})', { size: formatBytes(maxBytes) })
      )
      return
    }
    if (urls.length >= maxItems) {
      toast.error(t('Maximum {{count}} URLs', { count: maxItems }))
      return
    }

    let videoDuration = 0
    if (kind === 'video') {
      try {
        videoDuration = await readVideoFileDuration(file)
      } catch {
        toast.error(t('Upload failed'))
        return
      }

      const nextTotalDuration =
        sumVideoDurations(videoDurations ?? {}) + videoDuration
      if (
        maxTotalVideoDuration !== undefined &&
        nextTotalDuration > maxTotalVideoDuration
      ) {
        toast.error(t('Input video duration must be 0-15.5 seconds.'))
        return
      }
    }

    setIsUploading(true)
    try {
      const uploadedUrl = await uploadReferenceMedia(file, kind)
      if (!urls.includes(uploadedUrl)) {
        updateUrls([...urls, uploadedUrl])
      }
      if (kind === 'video' && videoDuration > 0) {
        onVideoDurationChange?.(uploadedUrl, videoDuration)
      }
      toast.success(t('Upload complete'))
    } catch (uploadError: unknown) {
      const message =
        uploadError instanceof Error ? uploadError.message : t('Upload failed')
      toast.error(message)
    } finally {
      setIsUploading(false)
    }
  }

  return (
    <div className='space-y-2'>
      <div className='flex items-center gap-2'>
        <Label className='text-sm font-medium'>{label}</Label>
        <Tooltip>
          <TooltipTrigger
            render={
              <button
                type='button'
                className='text-muted-foreground hover:text-foreground ml-auto rounded-md p-1 transition-colors'
                aria-label={t('Show field help')}
              />
            }
          >
            <CircleHelp size={15} />
          </TooltipTrigger>
          <TooltipContent side='top' align='end' className='max-w-80 text-left'>
            {hint}
          </TooltipContent>
        </Tooltip>
      </div>
      {urls.length > 0 && (
        <div className='space-y-1.5'>
          {urls.map((url) => {
            let mediaIcon = (
              <LinkIcon className='text-muted-foreground size-4 shrink-0' />
            )
            if (kind === 'audio') {
              mediaIcon = (
                <MusicIcon className='text-muted-foreground size-4 shrink-0' />
              )
            } else if (kind === 'video') {
              mediaIcon = (
                <FilmIcon className='text-muted-foreground size-4 shrink-0' />
              )
            }

            return (
              <div
                key={url}
                className='bg-muted/30 flex min-h-9 items-center gap-2 rounded-md border px-2 py-1.5'
              >
                {mediaIcon}
                <span className='min-w-0 flex-1 truncate text-xs' title={url}>
                  {url}
                </span>
                {kind === 'video' && videoDurations?.[url] !== undefined && (
                  <span className='text-muted-foreground shrink-0 text-xs'>
                    {formatDurationSeconds(videoDurations[url])}
                  </span>
                )}
                {!disabled && (
                  <Button
                    type='button'
                    variant='ghost'
                    size='icon'
                    className='size-7 shrink-0'
                    onClick={() => removeUrl(url)}
                    aria-label={t('Remove')}
                  >
                    <Trash2Icon size={14} />
                  </Button>
                )}
              </div>
            )
          })}
        </div>
      )}
      {urls.length < maxItems && (
        <div className='flex items-center gap-2'>
          <div className='relative min-w-0 flex-1'>
            <Input
              value={urlText}
              onChange={(event) => setUrlText(event.target.value)}
              onKeyDown={handleKeyDown}
              placeholder={placeholder}
              disabled={disabled || isUploading}
              className='pr-9'
            />
            <Button
              type='button'
              variant='ghost'
              size='icon'
              className='text-muted-foreground hover:text-foreground absolute top-1/2 right-1 size-7 -translate-y-1/2'
              disabled={!canAdd || !urlText.trim()}
              onClick={() => addUrl(urlText)}
              aria-label={t('Add URL')}
            >
              <PlusIcon size={15} />
            </Button>
          </div>
          <Button
            type='button'
            variant='outline'
            size='icon'
            className='shrink-0'
            disabled={!canAdd}
            onClick={() => fileRef.current?.click()}
            aria-label={uploadLabel}
            title={uploadLabel}
          >
            {isUploading ? (
              <Loader2Icon className='animate-spin' size={16} />
            ) : (
              <UploadIcon size={16} />
            )}
          </Button>
          <input
            ref={fileRef}
            type='file'
            accept={accept}
            className='hidden'
            onChange={handleFile}
          />
        </div>
      )}
      {!error && (
        <p className='text-muted-foreground text-xs'>
          {t('{{current}}/{{max}} URLs', {
            current: urls.length,
            max: maxItems,
          })}
        </p>
      )}
      {error && <p className='text-destructive text-xs'>{error}</p>}
    </div>
  )
}

function isHttpURL(value: string): boolean {
  try {
    const url = new URL(value)
    return url.protocol === 'http:' || url.protocol === 'https:'
  } catch {
    return false
  }
}

function formatBytes(bytes: number): string {
  if (bytes >= 1024 * 1024) {
    return `${Math.floor(bytes / (1024 * 1024))}MB`
  }
  if (bytes >= 1024) {
    return `${Math.floor(bytes / 1024)}KB`
  }
  return `${bytes}B`
}

function normalizeVideoDurationMap(
  durations: Record<string, number>,
  videos: string[]
): Record<string, number> {
  const nextDurations: Record<string, number> = {}
  for (const video of videos) {
    const duration = durations[video]
    if (Number.isFinite(duration) && duration > 0) {
      nextDurations[video] = roundVideoDuration(duration)
    }
  }
  return nextDurations
}

function sumVideoDurations(durations: Record<string, number>): number {
  const total = Object.values(durations).reduce((sum, duration) => {
    if (!Number.isFinite(duration) || duration <= 0) return sum
    return sum + duration
  }, 0)
  return roundVideoDuration(total)
}

function roundVideoDuration(duration: number): number {
  return Math.round(duration * 100) / 100
}

function formatDurationSeconds(duration: number): string {
  return `${roundVideoDuration(duration)}s`
}

function readVideoFileDuration(file: File): Promise<number> {
  return new Promise((resolve, reject) => {
    const url = URL.createObjectURL(file)
    const video = document.createElement('video')

    const cleanup = () => {
      video.removeAttribute('src')
      video.load()
      URL.revokeObjectURL(url)
    }

    video.preload = 'metadata'
    video.addEventListener(
      'loadedmetadata',
      () => {
        const duration = video.duration
        cleanup()
        if (!Number.isFinite(duration) || duration <= 0) {
          reject(new Error('invalid video duration'))
          return
        }
        resolve(roundVideoDuration(duration))
      },
      { once: true }
    )
    video.addEventListener(
      'error',
      () => {
        cleanup()
        reject(new Error('failed to read video duration'))
      },
      { once: true }
    )
    video.src = url
  })
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
            className='data-active:bg-muted h-7 min-w-0 flex-1 rounded-none border-r px-1.5 text-[11px] leading-none shadow-none last:border-r-0 data-active:shadow-none'
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
    <Tabs
      value={value}
      onValueChange={onValueChange}
      className='!flex-row gap-0'
    >
      <TabsList className='bg-background h-9 overflow-hidden rounded-lg border p-0'>
        {options.map((option) => (
          <TabsTrigger
            key={option.value}
            value={option.value}
            disabled={disabled}
            className='data-active:bg-muted h-9 min-w-20 rounded-none border-r px-4 text-sm shadow-none last:border-r-0 data-active:shadow-none'
          >
            {param.key === 'resolution' ? option.label : t(option.label)}
          </TabsTrigger>
        ))}
      </TabsList>
    </Tabs>
  )
}
