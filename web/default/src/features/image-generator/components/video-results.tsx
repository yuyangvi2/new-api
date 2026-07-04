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
import {
  AlertCircleIcon,
  DownloadIcon,
  FilmIcon,
  Trash2Icon,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Spinner } from '@/components/ui/spinner'
import type { VideoBatch } from '../types'

interface VideoResultsProps {
  batches: VideoBatch[]
  onClearHistory: () => void
}

function StatusOverlay({ label }: { label: string }) {
  return (
    <div className='bg-muted/40 text-muted-foreground flex aspect-video w-full flex-col items-center justify-center gap-3 rounded-xl border'>
      <Spinner className='size-6' />
      <span className='text-sm'>{label}</span>
    </div>
  )
}

export function VideoResults({ batches, onClearHistory }: VideoResultsProps) {
  const { t } = useTranslation()

  if (batches.length === 0) {
    return (
      <div className='flex h-full flex-col items-center justify-center p-8 text-center'>
        <div className='bg-muted text-muted-foreground mb-4 flex size-16 items-center justify-center rounded-2xl'>
          <FilmIcon size={28} />
        </div>
        <h3 className='text-lg font-medium'>{t('No videos yet')}</h3>
        <p className='text-muted-foreground mt-1 max-w-sm text-sm'>
          {t(
            'Pick an input image and click Generate video to animate it. This can take a minute or two.'
          )}
        </p>
      </div>
    )
  }

  return (
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
            <div className='flex items-start gap-3'>
              <img
                src={batch.imagePreview}
                alt=''
                className='size-12 shrink-0 rounded-md border object-cover'
              />
              <p className='text-muted-foreground line-clamp-2 flex-1 text-sm'>
                {batch.prompt || t('(no prompt)')}
              </p>
              <Badge variant='secondary' className='shrink-0'>
                {batch.model}
              </Badge>
            </div>

            {batch.status === 'submitting' && (
              <StatusOverlay label={t('Submitting task...')} />
            )}

            {batch.status === 'polling' && (
              <StatusOverlay
                label={
                  batch.progress
                    ? t('Generating video ({{progress}})...', {
                        progress: batch.progress,
                      })
                    : t('Generating video...')
                }
              />
            )}

            {batch.status === 'error' && (
              <div className='border-destructive/30 bg-destructive/5 text-destructive flex items-center gap-2 rounded-lg border p-3 text-sm'>
                <AlertCircleIcon size={16} />
                {batch.errorMessage || t('Video generation failed')}
              </div>
            )}

            {batch.status === 'complete' && batch.videoUrl && (
              <div className='space-y-2'>
                <video
                  src={batch.videoUrl}
                  controls
                  className='aspect-video w-full rounded-xl border bg-black'
                />
                <div className='flex justify-end'>
                  <a
                    href={batch.videoUrl}
                    download={`video-${batch.id}.mp4`}
                    target='_blank'
                    rel='noopener noreferrer'
                    className='text-muted-foreground hover:text-foreground inline-flex items-center gap-1.5 text-sm'
                  >
                    <DownloadIcon size={14} />
                    {t('Download')}
                  </a>
                </div>
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}
