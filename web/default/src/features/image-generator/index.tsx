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
import { useEffect, useMemo, useState } from 'react'
import { createPortal } from 'react-dom'
import { useQuery } from '@tanstack/react-query'
import { ImageIcon, FilmIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { cn } from '@/lib/utils'
import { useSidebarPortalTarget } from '@/context/sidebar-portal'
import { getUserGroups, getUserModels } from './api'
import { GeneratorPanel } from './components/generator-panel'
import { ResultGallery } from './components/result-gallery'
import { VideoPanel } from './components/video-panel'
import { VideoResults } from './components/video-results'
import { useImageGenerator, useVideoGenerator } from './hooks'
import type { GeneratorMode } from './types'

const IMAGE_MODEL_RE = /dall-e|image|flux|sd|stable|gpt-image|midjourney|ideogram/i
const VIDEO_MODEL_RE =
  /kling|jimeng|sora|vidu|cogvideo|video|hailuo|minimax|wan|seedance|runway|luma|pika|veo|doubao/i

export function ImageGenerator() {
  const { t } = useTranslation()
  const [mode, setMode] = useState<GeneratorMode>('image')
  const portalTarget = useSidebarPortalTarget()

  const imageGen = useImageGenerator()
  const videoGen = useVideoGenerator()

  const { data: models = [], isLoading: isModelLoading } = useQuery({
    queryKey: ['image-generator-models'],
    queryFn: async () => {
      try {
        return await getUserModels()
      } catch {
        toast.error(t('Failed to load models'))
        return []
      }
    },
  })

  const { data: groups = [] } = useQuery({
    queryKey: ['image-generator-groups'],
    queryFn: async () => {
      try {
        return await getUserGroups()
      } catch {
        return []
      }
    },
  })

  // Completed images from Image mode, offered as video input sources.
  const availableImages = useMemo(
    () =>
      imageGen.batches
        .filter((b) => b.status === 'complete')
        .flatMap((b) => b.images.map((img) => ({ id: img.id, src: img.src }))),
    [imageGen.batches]
  )

  const imageModel = imageGen.config.model
  const videoModel = videoGen.config.model
  const imageGroup = imageGen.config.group
  const videoGroup = videoGen.config.group
  const updateImageConfig = imageGen.updateConfig
  const updateVideoConfig = videoGen.updateConfig

  // Default the image model to an image-capable one.
  useEffect(() => {
    if (models.length === 0) return
    if (models.some((m) => m.value === imageModel)) return
    const next =
      models.find((m) => IMAGE_MODEL_RE.test(m.value))?.value ?? models[0].value
    updateImageConfig('model', next)
  }, [models, imageModel, updateImageConfig])

  // Default the video model to a video-capable one.
  useEffect(() => {
    if (models.length === 0) return
    if (models.some((m) => m.value === videoModel)) return
    const next =
      models.find((m) => VIDEO_MODEL_RE.test(m.value))?.value ?? models[0].value
    updateVideoConfig('model', next)
  }, [models, videoModel, updateVideoConfig])

  // Default groups for both configs when unavailable.
  useEffect(() => {
    if (groups.length === 0) return
    const fallback =
      groups.find((g) => g.value === 'default')?.value ?? groups[0].value
    if (!groups.some((g) => g.value === imageGroup)) {
      updateImageConfig('group', fallback)
    }
    if (!groups.some((g) => g.value === videoGroup)) {
      updateVideoConfig('group', fallback)
    }
  }, [groups, imageGroup, videoGroup, updateImageConfig, updateVideoConfig])

  const tabs: { mode: GeneratorMode; label: string; icon: typeof ImageIcon }[] =
    [
      { mode: 'image', label: t('Image'), icon: ImageIcon },
      { mode: 'video', label: t('Video'), icon: FilmIcon },
    ]

  // Sidebar content: mode toggle + config panel
  const sidebarContent = (
    <div className='flex h-full flex-col'>
      {/* Mode toggle */}
      <div className='flex items-center gap-2 border-b px-3 py-2'>
        <div className='bg-muted flex w-full gap-1 rounded-lg p-1'>
          {tabs.map((tab) => {
            const Icon = tab.icon
            return (
              <button
                key={tab.mode}
                type='button'
                onClick={() => setMode(tab.mode)}
                className={cn(
                  'flex flex-1 items-center justify-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors',
                  mode === tab.mode
                    ? 'bg-background text-foreground shadow-sm'
                    : 'text-muted-foreground hover:text-foreground'
                )}
              >
                <Icon size={15} />
                {tab.label}
              </button>
            )
          })}
        </div>
      </div>

      {/* Config panel */}
      <div className='min-h-0 flex-1 overflow-hidden'>
        {mode === 'image' ? (
          <GeneratorPanel
            config={imageGen.config}
            updateConfig={imageGen.updateConfig}
            models={models}
            groups={groups}
            isModelLoading={isModelLoading}
            isGenerating={imageGen.isGenerating}
            onGenerate={imageGen.generate}
            onCancel={imageGen.cancel}
          />
        ) : (
          <VideoPanel
            config={videoGen.config}
            updateConfig={videoGen.updateConfig}
            models={models}
            groups={groups}
            isModelLoading={isModelLoading}
            isGenerating={videoGen.isGenerating}
            availableImages={availableImages}
            onGenerate={videoGen.generate}
            onCancel={videoGen.cancel}
          />
        )}
      </div>
    </div>
  )

  return (
    <>
      {/* Inject config panel into sidebar via portal */}
      {portalTarget && createPortal(sidebarContent, portalTarget)}

      {/* Results in main content area */}
      <div className='size-full'>
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
      </div>
    </>
  )
}
