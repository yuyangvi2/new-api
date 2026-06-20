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
import { useRef, useState } from 'react'
import { UploadCloudIcon, XIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { cn } from '@/lib/utils'
import { Input } from '@/components/ui/input'
import { MAX_IMAGE_UPLOAD_BYTES } from '../constants'
import type { ImageSourceType } from '../types'

interface AvailableImage {
  id: string
  src: string
}

interface ImageSourceInputProps {
  value: string
  sourceType: ImageSourceType
  onChange: (src: string, sourceType: ImageSourceType) => void
  availableImages: AvailableImage[]
  disabled?: boolean
}

const TABS: { type: ImageSourceType; label: string }[] = [
  { type: 'upload', label: 'Upload' },
  { type: 'generated', label: 'From gallery' },
  { type: 'url', label: 'URL' },
]

export function ImageSourceInput({
  value,
  sourceType,
  onChange,
  availableImages,
  disabled,
}: ImageSourceInputProps) {
  const { t } = useTranslation()
  const fileRef = useRef<HTMLInputElement>(null)
  const [urlText, setUrlText] = useState(sourceType === 'url' ? value : '')

  const handleFile = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    e.target.value = ''
    if (!file) return
    if (!file.type.startsWith('image/')) {
      toast.error(t('Please choose an image file'))
      return
    }
    if (file.size > MAX_IMAGE_UPLOAD_BYTES) {
      toast.error(t('Image is too large (max 10MB)'))
      return
    }
    const reader = new FileReader()
    reader.onload = () => onChange(String(reader.result), 'upload')
    reader.onerror = () => toast.error(t('Failed to read the image'))
    reader.readAsDataURL(file)
  }

  const switchTab = (type: ImageSourceType) => {
    if (disabled) return
    // Clear the selection when switching source kinds to avoid confusion.
    if (type !== sourceType) onChange('', type)
  }

  return (
    <div className='space-y-2'>
      {/* Source tabs */}
      <div className='bg-muted flex gap-1 rounded-lg p-1'>
        {TABS.map((tab) => (
          <button
            key={tab.type}
            type='button'
            disabled={disabled}
            onClick={() => switchTab(tab.type)}
            className={cn(
              'flex-1 rounded-md px-2 py-1 text-xs font-medium transition-colors disabled:opacity-50',
              sourceType === tab.type
                ? 'bg-background text-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground'
            )}
          >
            {t(tab.label)}
          </button>
        ))}
      </div>

      {/* Current selection preview */}
      {value ? (
        <div className='bg-muted/40 relative overflow-hidden rounded-lg border'>
          <img
            src={value}
            alt={t('Input image')}
            className='max-h-48 w-full object-contain'
          />
          {!disabled && (
            <button
              type='button'
              onClick={() => onChange('', sourceType)}
              className='bg-background/80 hover:bg-background absolute top-1.5 right-1.5 rounded-full border p-1'
              aria-label={t('Remove image')}
            >
              <XIcon size={14} />
            </button>
          )}
        </div>
      ) : (
        <>
          {sourceType === 'upload' && (
            <button
              type='button'
              disabled={disabled}
              onClick={() => fileRef.current?.click()}
              className='border-muted-foreground/30 hover:border-muted-foreground/60 text-muted-foreground flex w-full flex-col items-center gap-2 rounded-lg border border-dashed p-6 text-sm transition-colors disabled:opacity-50'
            >
              <UploadCloudIcon size={22} />
              {t('Click to upload an image')}
            </button>
          )}

          {sourceType === 'generated' &&
            (availableImages.length === 0 ? (
              <p className='text-muted-foreground rounded-lg border border-dashed p-4 text-center text-xs'>
                {t('No generated images yet. Create some in Image mode first.')}
              </p>
            ) : (
              <div className='grid max-h-44 grid-cols-3 gap-2 overflow-y-auto'>
                {availableImages.map((img) => (
                  <button
                    key={img.id}
                    type='button'
                    disabled={disabled}
                    onClick={() => onChange(img.src, 'generated')}
                    className='hover:ring-primary overflow-hidden rounded-md border hover:ring-2'
                  >
                    <img
                      src={img.src}
                      alt=''
                      className='aspect-square w-full object-cover'
                    />
                  </button>
                ))}
              </div>
            ))}

          {sourceType === 'url' && (
            <Input
              type='url'
              value={urlText}
              disabled={disabled}
              placeholder='https://example.com/image.jpg'
              onChange={(e) => setUrlText(e.target.value)}
              onBlur={() => urlText.trim() && onChange(urlText.trim(), 'url')}
            />
          )}
        </>
      )}

      <input
        ref={fileRef}
        type='file'
        accept='image/*'
        className='hidden'
        onChange={handleFile}
      />
    </div>
  )
}
