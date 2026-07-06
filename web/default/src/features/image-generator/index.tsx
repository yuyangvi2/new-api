import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { ArrowLeft, ImageIcon, FilmIcon } from 'lucide-react'
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
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { useSidebarPortalTarget } from '@/context/sidebar-portal'
import { cn } from '@/lib/utils'

import { getUserGroups, getUserModels } from './api'
import { GeneratorPanel } from './components/generator-panel'
import { ResultGallery } from './components/result-gallery'
import { VideoPanel } from './components/video-panel'
import { VideoResults } from './components/video-results'
import {
  detectImageModelFamily,
  getVideoVariantDisplayName,
  isHiddenVideoVariantModel,
} from './constants'
import { useImageGenerator, useVideoGenerator } from './hooks'
import type { GeneratorMode, GroupOption, ModelOption } from './types'

const IMAGE_MODEL_RE =
  /dall-e|image|flux|sd|stable|gpt-image|midjourney|ideogram/i
const VIDEO_MODEL_RE =
  /kling|jimeng|sora|vidu|cogvideo|video|hailuo|minimax|wan|seedance|runway|luma|pika|veo|i2v|t2v|s2v/i
const NON_VIDEO_MODEL_RE =
  /embedding|seedream|seed-|suno|lyrics|music|image|dall-e|gpt-image|flux|stable|midjourney|ideogram/i

function isVideoGenerationModel(model: ModelOption): boolean {
  const name = model.value
  if (NON_VIDEO_MODEL_RE.test(name) && !/seedance/i.test(name)) return false
  return VIDEO_MODEL_RE.test(name)
}

function selectGroupForModel(model: ModelOption, currentGroup: string): string {
  if (!model.groups || model.groups.length === 0) return currentGroup
  if (model.groups.includes(currentGroup)) return currentGroup
  return model.groups[0]
}

function getDisplayVideoModels(models: ModelOption[]): ModelOption[] {
  const availableModelNames = models.map((m) => m.value)
  return models
    .filter((m) => !isHiddenVideoVariantModel(m.value, availableModelNames))
    .map((m) => ({
      ...m,
      label: getVideoVariantDisplayName(m.value) ?? m.label,
    }))
}

interface ImageGeneratorProps {
  initialVideoTaskId?: string
}

function mergeGroupOptions(
  groups: GroupOption[],
  models: ModelOption[],
  currentGroups: string[]
): GroupOption[] {
  const merged = new Map<string, GroupOption>()

  for (const group of groups) {
    merged.set(group.value, group)
  }

  for (const model of models) {
    for (const group of model.groups ?? []) {
      if (!merged.has(group)) {
        merged.set(group, { label: group, value: group })
      }
    }
  }

  if (merged.size === 0) {
    for (const group of currentGroups) {
      if (group && !merged.has(group)) {
        merged.set(group, { label: group, value: group })
      }
    }
  }

  return [...merged.values()]
}

export function ImageGenerator(props: ImageGeneratorProps) {
  const { t } = useTranslation()
  const [mode, setMode] = useState<GeneratorMode>('image')
  const recoveredVideoTaskIdRef = useRef<string | null>(null)
  const portalTarget = useSidebarPortalTarget()

  const imageGen = useImageGenerator()
  const videoGen = useVideoGenerator()
  const imageConfig = imageGen.config
  const videoConfig = videoGen.config
  const updateImageConfig = imageGen.updateConfig
  const updateVideoConfig = videoGen.updateConfig
  const recoverVideoTask = videoGen.recoverTask
  const isVideoGenerating = videoGen.isGenerating

  useEffect(() => {
    const taskId = props.initialVideoTaskId?.trim()
    if (!taskId || recoveredVideoTaskIdRef.current === taskId) return
    if (isVideoGenerating) return

    recoveredVideoTaskIdRef.current = taskId
    setMode('video')
    recoverVideoTask(taskId)
  }, [props.initialVideoTaskId, isVideoGenerating, recoverVideoTask])

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

  const { data: fetchedGroups = [] } = useQuery({
    queryKey: ['image-generator-groups', t],
    queryFn: async () => {
      try {
        return await getUserGroups()
      } catch {
        toast.error(t('Failed to load groups'))
        return []
      }
    },
  })

  const groups = useMemo(
    () =>
      mergeGroupOptions(fetchedGroups, models, [
        imageConfig.group,
        videoConfig.group,
      ]),
    [fetchedGroups, models, imageConfig.group, videoConfig.group]
  )

  // Filter models by tab type so image tab only shows image models, etc.
  const imageModels = useMemo(
    () => models.filter((m) => IMAGE_MODEL_RE.test(m.value)),
    [models]
  )
  const allVideoModels = useMemo(
    () => models.filter(isVideoGenerationModel),
    [models]
  )
  const selectedVideoGroup = useMemo(
    () => groups.find((group) => group.value === videoConfig.group),
    [groups, videoConfig.group]
  )
  // Completed images from Image mode, offered as video input sources.
  const availableImages = useMemo(
    () =>
      imageGen.batches
        .filter((b) => b.status === 'complete')
        .flatMap((b) => b.images.map((img) => ({ id: img.id, src: img.src }))),
    [imageGen.batches]
  )

  const applyImageModelChange = useCallback(
    (value: string) => {
      const oldFamily = detectImageModelFamily(imageConfig.model)
      const newFamily = detectImageModelFamily(value)
      updateImageConfig('model', value)
      // Reset size when switching between families with different size formats
      if (oldFamily !== newFamily) {
        let defaultSize = '1024x1024'
        if (newFamily === 'hunyuan-image') {
          defaultSize = '1024:1024'
        } else if (newFamily === 'image-gi' || newFamily === 'image-gi2') {
          defaultSize = '1:1'
        }
        updateImageConfig('size', defaultSize)
      }
      // Clear metadata when switching families
      if (oldFamily !== newFamily) {
        updateImageConfig('metadata', {})
        updateImageConfig('images', [])
      }
    },
    [imageConfig.model, updateImageConfig]
  )

  const handleImageModelChange = useCallback(
    (value: string) => {
      applyImageModelChange(value)
      const nextModel = imageModels.find((m) => m.value === value)
      if (nextModel) {
        updateImageConfig(
          'group',
          selectGroupForModel(nextModel, imageConfig.group)
        )
      }
    },
    [applyImageModelChange, imageConfig.group, imageModels, updateImageConfig]
  )

  const handleVideoModelChange = useCallback(
    (value: string) => {
      updateVideoConfig('model', value)
      const nextModel = allVideoModels.find((m) => m.value === value)
      if (nextModel) {
        updateVideoConfig(
          'group',
          selectGroupForModel(nextModel, videoConfig.group)
        )
      }
    },
    [allVideoModels, updateVideoConfig, videoConfig.group]
  )

  useEffect(() => {
    if (groups.length === 0) return
    if (!groups.some((g) => g.value === imageConfig.group)) {
      updateImageConfig('group', groups[0].value)
    }
  }, [groups, imageConfig.group, updateImageConfig])

  useEffect(() => {
    if (groups.length === 0) return
    if (!groups.some((g) => g.value === videoConfig.group)) {
      updateVideoConfig('group', groups[0].value)
    }
  }, [groups, videoConfig.group, updateVideoConfig])

  // Auto-select initial model when models load
  useEffect(() => {
    if (imageModels.length === 0) return
    const currentModel = imageModels.find((m) => m.value === imageConfig.model)
    const nextModel = currentModel ?? imageModels[0]
    if (!currentModel) {
      applyImageModelChange(nextModel.value)
    }
    updateImageConfig(
      'group',
      selectGroupForModel(nextModel, imageConfig.group)
    )
  }, [
    applyImageModelChange,
    imageConfig.group,
    imageConfig.model,
    imageModels,
    updateImageConfig,
  ])

  useEffect(() => {
    const displayVideoModels = getDisplayVideoModels(allVideoModels)
    if (displayVideoModels.length === 0) return
    const currentModel = allVideoModels.find(
      (m) => m.value === videoConfig.model
    )
    const nextModel = currentModel ?? displayVideoModels[0]
    if (!currentModel) {
      updateVideoConfig('model', nextModel.value)
    }
    updateVideoConfig(
      'group',
      selectGroupForModel(nextModel, videoConfig.group)
    )
  }, [allVideoModels, videoConfig.model, videoConfig.group, updateVideoConfig])

  // ---- Icon rail tabs (rendered into the 80px sidebar via portal) ----
  const railTabs: {
    mode: GeneratorMode
    label: string
    icon: typeof ImageIcon
  }[] = [
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
              models={getDisplayVideoModels(allVideoModels)}
              variantModels={allVideoModels}
              groupRatio={selectedVideoGroup?.ratio}
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
              onRecoverTask={videoGen.recoverTask}
              onClearHistory={videoGen.clearHistory}
            />
          )}
        </section>
      </div>
    </>
  )
}
