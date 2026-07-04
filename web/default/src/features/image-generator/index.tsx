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
import { useCallback, useEffect, useMemo, useState } from 'react'
import { createPortal } from 'react-dom'
import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { ArrowLeft, ImageIcon, FilmIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { cn } from '@/lib/utils'
import { useSidebarPortalTarget } from '@/context/sidebar-portal'
import { getUserModels } from './api'
import { detectImageModelFamily, isHiddenVideoVariantModel } from './constants'
import { GeneratorPanel } from './components/generator-panel'
import { ResultGallery } from './components/result-gallery'
import { VideoPanel } from './components/video-panel'
import { VideoResults } from './components/video-results'
import { useImageGenerator, useVideoGenerator } from './hooks'
import type { GeneratorMode, ModelOption } from './types'

const IMAGE_MODEL_RE = /dall-e|image|flux|sd|stable|gpt-image|midjourney|ideogram/i
const VIDEO_MODEL_RE =
  /kling|jimeng|sora|vidu|cogvideo|video|hailuo|minimax|wan|seedance|runway|luma|pika|veo|doubao/i

function resolveModelGroup(
  model: ModelOption | undefined,
  currentGroup: string
): string | undefined {
  if (!model?.groups || model.groups.length === 0) return undefined
  if (model.groups.includes(currentGroup)) return currentGroup
  return model.groups[0]
}

export function ImageGenerator() {
  const { t } = useTranslation()
  const [mode, setMode] = useState<GeneratorMode>('image')
  const portalTarget = useSidebarPortalTarget()

  const imageGen = useImageGenerator()
  const videoGen = useVideoGenerator()

  const { data: models = [], isLoading: isModelLoading } = useQuery({
    queryKey: ['image-generator-models', t],
    queryFn: async () => {
      try {
        return await getUserModels()
      } catch {
        toast.error(t('Failed to load models'))
        return []
      }
    },
  })

  // Filter models by tab type so image tab only shows image models, etc.
  const imageModels = useMemo(
    () => models.filter((m) => IMAGE_MODEL_RE.test(m.value)),
    [models]
  )
  const allVideoModels = useMemo(
    // Exclude image models that also match a video keyword (e.g. wan2.7-image-pro
    // contains "wan") so the video tab only lists real video models.
    () =>
      models.filter(
        (m) => VIDEO_MODEL_RE.test(m.value) && !IMAGE_MODEL_RE.test(m.value)
      ),
    [models]
  )
  const videoModels = useMemo(() => {
    const availableModelNames = allVideoModels.map((m) => m.value)
    return allVideoModels.filter(
      (m) => !isHiddenVideoVariantModel(m.value, availableModelNames)
    )
  }, [allVideoModels])

  // Completed images from Image mode, offered as video input sources.
  const availableImages = useMemo(
    () =>
      imageGen.batches
        .filter((b) => b.status === 'complete')
        .flatMap((b) => b.images.map((img) => ({ id: img.id, src: img.src }))),
    [imageGen.batches]
  )

  const handleImageModelChange = useCallback(
    (value: string) => {
      const oldFamily = detectImageModelFamily(imageGen.config.model)
      const newFamily = detectImageModelFamily(value)
      const nextGroup = resolveModelGroup(
        imageModels.find((m) => m.value === value),
        imageGen.config.group
      )
      imageGen.updateConfig('model', value)
      if (nextGroup && nextGroup !== imageGen.config.group) {
        imageGen.updateConfig('group', nextGroup)
      }
      // Reset size when switching between families with different size formats
      if (oldFamily !== newFamily) {
        const defaultSize = newFamily === 'hunyuan-image'
          ? '1024:1024'
          : newFamily === 'image-gi' || newFamily === 'image-gi2'
            ? '1:1'
            : '1024x1024'
        imageGen.updateConfig('size', defaultSize)
      }
      // Clear metadata when switching families
      if (oldFamily !== newFamily) {
        imageGen.updateConfig('metadata', {})
        imageGen.updateConfig('images', [])
      }
    },
    [
      imageGen.updateConfig,
      imageGen.config.model,
      imageGen.config.group,
      imageModels,
    ]
  )

  const handleVideoModelChange = useCallback(
    (value: string) => {
      const nextGroup = resolveModelGroup(
        allVideoModels.find((m) => m.value === value),
        videoGen.config.group
      )
      videoGen.updateConfig('model', value)
      if (nextGroup && nextGroup !== videoGen.config.group) {
        videoGen.updateConfig('group', nextGroup)
      }
    },
    [videoGen.updateConfig, videoGen.config.group, allVideoModels]
  )

  // Auto-select initial model when models load
  useEffect(() => {
    if (imageModels.length === 0) return
    const currentModel = imageModels.find((m) => m.value === imageGen.config.model)
    const nextModel = currentModel ?? imageModels[0]
    if (!currentModel) {
      imageGen.updateConfig('model', nextModel.value)
    }
    const nextGroup = resolveModelGroup(nextModel, imageGen.config.group)
    if (nextGroup && nextGroup !== imageGen.config.group) {
      imageGen.updateConfig('group', nextGroup)
    }
  }, [
    imageModels,
    imageGen.config.model,
    imageGen.config.group,
    imageGen.updateConfig,
  ])

  useEffect(() => {
    if (videoModels.length === 0) return
    const currentModel = allVideoModels.find(
      (m) => m.value === videoGen.config.model
    )
    const nextModel = currentModel ?? videoModels[0]
    if (!currentModel) {
      videoGen.updateConfig('model', nextModel.value)
    }
    const nextGroup = resolveModelGroup(nextModel, videoGen.config.group)
    if (nextGroup && nextGroup !== videoGen.config.group) {
      videoGen.updateConfig('group', nextGroup)
    }
  }, [
    videoModels,
    allVideoModels,
    videoGen.config.model,
    videoGen.config.group,
    videoGen.updateConfig,
  ])

  // ---- Icon rail tabs (rendered into the 80px sidebar via portal) ----
  const railTabs: { mode: GeneratorMode; label: string; icon: typeof ImageIcon }[] = [
    { mode: 'image', label: t('Image'), icon: ImageIcon },
    { mode: 'video', label: t('Video'), icon: FilmIcon },
  ]

  const iconRail = (
    <nav className='flex h-full flex-col items-center'>
      {/* Back */}
      <Link
        to='/dashboard/$section'
        params={{ section: 'overview' }}
        className='text-muted-foreground hover:text-foreground hover:bg-accent flex w-full flex-col items-center gap-1 border-b px-2 py-3 text-xs transition-colors'
      >
        <ArrowLeft size={20} />
        <span>{t('Back')}</span>
      </Link>

      {/* Mode tabs */}
      <div className='flex w-full flex-1 flex-col items-center gap-1 pt-2'>
        {railTabs.map((tab) => {
          const Icon = tab.icon
          const active = mode === tab.mode
          return (
            <button
              key={tab.mode}
              type='button'
              onClick={() => setMode(tab.mode)}
              className={cn(
                'flex w-full flex-col items-center gap-1 rounded-md px-2 py-2.5 text-xs transition-colors',
                active
                  ? 'bg-primary/10 text-primary font-medium'
                  : 'text-muted-foreground hover:bg-accent hover:text-foreground'
              )}
            >
              <Icon size={20} />
              <span>{tab.label}</span>
            </button>
          )
        })}
      </div>
    </nav>
  )

  return (
    <>
      {/* Icon rail → sidebar (80px) via portal */}
      {portalTarget && createPortal(iconRail, portalTarget)}

      {/* Main content: 320px config aside + results */}
      <div className='flex size-full min-h-0'>
        {/* Config panel */}
        <aside className='bg-card flex w-[320px] shrink-0 flex-col border-r'>
          {mode === 'image' ? (
            <GeneratorPanel
              config={imageGen.config}
              updateConfig={imageGen.updateConfig}
              onModelChange={handleImageModelChange}
              models={imageModels}
              isModelLoading={isModelLoading}
              isGenerating={imageGen.isGenerating}
              onGenerate={imageGen.generate}
              onCancel={imageGen.cancel}
            />
          ) : (
            <VideoPanel
              config={videoGen.config}
              updateConfig={videoGen.updateConfig}
              onModelChange={handleVideoModelChange}
          models={videoModels}
          variantModels={allVideoModels}
          isModelLoading={isModelLoading}
              isGenerating={videoGen.isGenerating}
              availableImages={availableImages}
              onGenerate={videoGen.generate}
              onCancel={videoGen.cancel}
            />
          )}
        </aside>

        {/* Results gallery */}
        <section className='min-w-0 flex-1'>
          {mode === 'image' ? (
            <ResultGallery
              batches={imageGen.batches}
              onClearHistory={imageGen.clearHistory}
            />
          ) : (
            <VideoResults
              batches={videoGen.batches}
              onClearHistory={videoGen.clearHistory}
            />
          )}
        </section>
      </div>
    </>
  )
}
