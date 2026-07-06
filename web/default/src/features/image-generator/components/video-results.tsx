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
import { useState, type FormEvent } from 'react'
import {
  AlertCircleIcon,
  DownloadIcon,
  FilmIcon,
  SearchIcon,
  Trash2Icon,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Spinner } from '@/components/ui/spinner'
import { cn } from '@/lib/utils'

import type { VideoBatch } from '../types'

interface VideoResultsProps {
  batches: VideoBatch[]
  onRecoverTask: (taskId: string) => void
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

function DebugResult({ value }: { value?: string }) {
  const { t } = useTranslation()

  if (!value) {
    return null
  }

  return (
    <div className='border-border bg-muted/30 rounded-md border p-2'>
      <div className='text-muted-foreground mb-1 text-xs font-medium'>
        {t('Latest polling result')}
      </div>
      <pre className='text-muted-foreground max-h-40 overflow-auto text-xs leading-relaxed break-words whitespace-pre-wrap'>
        {value}
      </pre>
    </div>
  )
}

interface TaskIdLookupProps {
  className?: string
  onRecoverTask: (taskId: string) => void
}

function TaskIdLookup(props: TaskIdLookupProps) {
  const { t } = useTranslation()
  const [taskId, setTaskId] = useState('')

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const trimmedTaskId = taskId.trim()
    if (!trimmedTaskId) return
    props.onRecoverTask(trimmedTaskId)
    setTaskId('')
  }

  return (
    <form
      className={cn('flex w-full max-w-md items-center gap-2', props.className)}
      onSubmit={handleSubmit}
    >
      <Input
        value={taskId}
        onChange={(event) => setTaskId(event.target.value)}
        placeholder={t('Enter task ID')}
        aria-label={t('Task ID')}
        className='font-mono text-xs'
      />
      <Button type='submit' size='sm' disabled={!taskId.trim()}>
        <SearchIcon size={14} />
        {t('Find')}
      </Button>
    </form>
  )
}

export function VideoResults(props: VideoResultsProps) {
  const { t } = useTranslation()

  if (props.batches.length === 0) {
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
        <TaskIdLookup className='mt-6' onRecoverTask={props.onRecoverTask} />
      </div>
    )
  }

  return (
    <div className='flex h-full flex-col'>
      <div className='flex flex-col gap-3 border-b p-4 md:flex-row md:items-center md:justify-between'>
        <h2 className='text-base font-semibold'>{t('Results')}</h2>
        <div className='flex flex-col gap-2 sm:flex-row sm:items-center'>
          <TaskIdLookup
            className='max-w-none sm:w-80'
            onRecoverTask={props.onRecoverTask}
          />
          <Button variant='ghost' size='sm' onClick={props.onClearHistory}>
            <Trash2Icon size={14} />
            {t('Clear')}
          </Button>
        </div>
      </div>

      <div className='flex-1 space-y-6 overflow-y-auto p-4'>
        {props.batches.map((batch) => (
          <div key={batch.id} className='space-y-2'>
            <div className='flex items-start gap-3'>
              {batch.imagePreview ? (
                <img
                  src={batch.imagePreview}
                  alt=''
                  className='size-12 shrink-0 rounded-md border object-cover'
                />
              ) : (
                <div className='bg-muted text-muted-foreground flex size-12 shrink-0 items-center justify-center rounded-md border'>
                  <FilmIcon size={18} />
                </div>
              )}
              <p className='text-muted-foreground line-clamp-2 flex-1 text-sm'>
                {batch.prompt || batch.taskId || t('(no prompt)')}
              </p>
              <Badge variant='secondary' className='shrink-0'>
                {batch.model || t('Recovered task')}
              </Badge>
            </div>

            {batch.status === 'submitting' && (
              <StatusOverlay label={t('Submitting task...')} />
            )}

            {batch.status === 'polling' && (
              <>
                <StatusOverlay
                  label={
                    batch.progress
                      ? t('Generating video ({{progress}})...', {
                          progress: batch.progress,
                        })
                      : t('Generating video...')
                  }
                />
                <DebugResult value={batch.debugResult} />
              </>
            )}

            {batch.status === 'error' && (
              <>
                <div className='border-destructive/30 bg-destructive/5 text-destructive flex items-center gap-2 rounded-lg border p-3 text-sm'>
                  <AlertCircleIcon size={16} />
                  {batch.errorMessage || t('Video generation failed')}
                </div>
                <DebugResult value={batch.debugResult} />
              </>
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
