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
import { DownloadIcon, MaximizeIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import type { GeneratedImage } from '../types'

interface ImageResultCardProps {
  image: GeneratedImage
  onPreview: (image: GeneratedImage) => void
}

export function ImageResultCard({ image, onPreview }: ImageResultCardProps) {
  const { t } = useTranslation()
  const [loaded, setLoaded] = useState(false)

  const handleDownload = async () => {
    try {
      let href = image.src
      // Remote URLs: fetch as blob so the browser saves rather than navigates.
      if (image.src.startsWith('http')) {
        const res = await fetch(image.src)
        const blob = await res.blob()
        href = URL.createObjectURL(blob)
      }
      const a = document.createElement('a')
      a.href = href
      a.download = `image-${image.id}.png`
      document.body.appendChild(a)
      a.click()
      a.remove()
      if (href !== image.src) URL.revokeObjectURL(href)
    } catch {
      // Fallback: open in a new tab if direct download fails (e.g. CORS).
      window.open(image.src, '_blank', 'noopener')
      toast.info(t('Opened image in a new tab'))
    }
  }

  return (
    <div className='group bg-muted/40 relative overflow-hidden rounded-xl border'>
      {!loaded && <Skeleton className='absolute inset-0 h-full w-full' />}
      <img
        src={image.src}
        alt={image.prompt}
        loading='lazy'
        onLoad={() => setLoaded(true)}
        className='aspect-square w-full cursor-zoom-in object-cover'
        onClick={() => onPreview(image)}
      />
      <div className='pointer-events-none absolute inset-x-0 bottom-0 flex items-center justify-end gap-1.5 bg-gradient-to-t from-black/60 to-transparent p-2 opacity-0 transition-opacity group-hover:opacity-100'>
        <Button
          size='icon-sm'
          variant='secondary'
          className='pointer-events-auto'
          onClick={() => onPreview(image)}
          aria-label={t('Preview')}
        >
          <MaximizeIcon size={14} />
        </Button>
        <Button
          size='icon-sm'
          variant='secondary'
          className='pointer-events-auto'
          onClick={handleDownload}
          aria-label={t('Download')}
        >
          <DownloadIcon size={14} />
        </Button>
      </div>
    </div>
  )
}
