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
import { ImageOff, Maximize2 } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Skeleton } from '@/components/ui/skeleton'

import { ImageDialog } from './dialogs/image-dialog'

interface DrawingImagePreviewProps {
  imageUrl: string
  taskId?: string
}

export function DrawingImagePreview({
  imageUrl,
  taskId,
}: DrawingImagePreviewProps) {
  const { t } = useTranslation()
  const [dialogOpen, setDialogOpen] = useState(false)
  const [isLoading, setIsLoading] = useState(true)
  const [hasError, setHasError] = useState(false)

  return (
    <>
      <button
        type='button'
        className='group border-border bg-muted focus-visible:ring-ring relative block size-14 overflow-hidden rounded-md border focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:outline-none'
        onClick={() => setDialogOpen(true)}
        title={t('Click to view image')}
        aria-label={t('Click to view image')}
      >
        {isLoading && !hasError ? (
          <Skeleton className='absolute inset-0 size-full rounded-none' />
        ) : null}
        {hasError ? (
          <span className='text-muted-foreground flex size-full items-center justify-center'>
            <ImageOff className='size-5' aria-hidden='true' />
          </span>
        ) : (
          <img
            src={imageUrl}
            alt={t('Generated image')}
            className='size-full object-cover transition-transform group-hover:scale-105'
            onLoad={() => setIsLoading(false)}
            onError={() => {
              setIsLoading(false)
              setHasError(true)
            }}
            loading='lazy'
            decoding='async'
          />
        )}
        {!hasError ? (
          <span className='bg-background/75 text-foreground absolute inset-0 flex items-center justify-center opacity-0 transition-opacity group-hover:opacity-100 group-focus-visible:opacity-100'>
            <Maximize2 className='size-4' aria-hidden='true' />
          </span>
        ) : null}
      </button>
      <ImageDialog
        imageUrl={imageUrl}
        taskId={taskId}
        open={dialogOpen}
        onOpenChange={setDialogOpen}
      />
    </>
  )
}
