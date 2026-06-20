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
import { useState } from 'react'
import { AlertCircleIcon, ImageIcon, Trash2Icon } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { Dialog } from '@/components/dialog'
import { ImageResultCard } from './image-result-card'
import type { GeneratedImage, GenerationBatch } from '../types'

interface ResultGalleryProps {
  batches: GenerationBatch[]
  onClearHistory: () => void
}

function LoadingGrid({ count }: { count: number }) {
  return (
    <div className='grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4'>
      {Array.from({ length: count }).map((_, i) => (
        <Skeleton key={i} className='aspect-square w-full rounded-xl' />
      ))}
    </div>
  )
}

export function ResultGallery({ batches, onClearHistory }: ResultGalleryProps) {
  const { t } = useTranslation()
  const [preview, setPreview] = useState<GeneratedImage | null>(null)

  if (batches.length === 0) {
    return (
      <div className='flex h-full flex-col items-center justify-center p-8 text-center'>
        <div className='bg-muted text-muted-foreground mb-4 flex size-16 items-center justify-center rounded-2xl'>
          <ImageIcon size={28} />
        </div>
        <h3 className='text-lg font-medium'>{t('No images yet')}</h3>
        <p className='text-muted-foreground mt-1 max-w-sm text-sm'>
          {t(
            'Enter a prompt and pick a model, then click Generate to create images.'
          )}
        </p>
      </div>
    )
  }

  return (
    <>
      <div className='flex h-full flex-col'>
        <div className='flex items-center justify-between border-b p-4'>
          <h2 className='text-base font-semibold'>{t('Results')}</h2>
          <Button variant='ghost' size='sm' onClick={onClearHistory}>
            <Trash2Icon size={14} />
            {t('Clear')}
          </Button>
        </div>

        <div className='flex-1 space-y-6 overflow-y-auto p-4'>
          {batches.map((batch) => (
            <div key={batch.id} className='space-y-2'>
              <div className='flex items-start justify-between gap-3'>
                <p className='text-muted-foreground line-clamp-2 flex-1 text-sm'>
                  {batch.prompt}
                </p>
                <Badge variant='secondary' className='shrink-0'>
                  {batch.model}
                </Badge>
              </div>

              {batch.status === 'loading' && (
                <LoadingGrid count={batch.count} />
              )}

              {batch.status === 'error' && (
                <div className='border-destructive/30 bg-destructive/5 text-destructive flex items-center gap-2 rounded-lg border p-3 text-sm'>
                  <AlertCircleIcon size={16} />
                  {batch.errorMessage || t('Image generation failed')}
                </div>
              )}

              {batch.status === 'complete' && (
                <div className='grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4'>
                  {batch.images.map((image) => (
                    <ImageResultCard
                      key={image.id}
                      image={image}
                      onPreview={setPreview}
                    />
                  ))}
                </div>
              )}
            </div>
          ))}
        </div>
      </div>

      <Dialog
        open={!!preview}
        onOpenChange={(open) => !open && setPreview(null)}
        title={t('Image Preview')}
        description={preview?.prompt}
        contentClassName='sm:max-w-3xl'
        contentHeight='auto'
      >
        {preview && (
          <div className='flex justify-center py-2'>
            <img
              src={preview.src}
              alt={preview.prompt}
              className='max-h-[70vh] w-auto rounded-lg object-contain'
            />
          </div>
        )}
      </Dialog>
    </>
  )
}
