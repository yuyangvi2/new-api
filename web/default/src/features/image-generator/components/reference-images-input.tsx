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
import { useRef } from 'react'
import { PlusIcon, XIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { MAX_IMAGE_UPLOAD_BYTES, MAX_REFERENCE_IMAGES } from '../constants'

interface ReferenceImagesInputProps {
  images: string[]
  onChange: (images: string[]) => void
  disabled?: boolean
}

export function ReferenceImagesInput({
  images,
  onChange,
  disabled,
}: ReferenceImagesInputProps) {
  const { t } = useTranslation()
  const fileRef = useRef<HTMLInputElement>(null)

  const handleFiles = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(e.target.files ?? [])
    e.target.value = ''
    if (files.length === 0) return

    const remaining = MAX_REFERENCE_IMAGES - images.length
    if (remaining <= 0) {
      toast.error(t('Maximum {{count}} reference images', { count: MAX_REFERENCE_IMAGES }))
      return
    }

    const toProcess = files.slice(0, remaining)
    let loaded = 0
    const newImages: string[] = []

    for (const file of toProcess) {
      if (!file.type.startsWith('image/')) {
        toast.error(t('Please choose an image file'))
        continue
      }
      if (file.size > MAX_IMAGE_UPLOAD_BYTES) {
        toast.error(t('Image is too large (max 10MB)'))
        continue
      }
      const reader = new FileReader()
      reader.onload = () => {
        newImages.push(String(reader.result))
        loaded++
        if (loaded === toProcess.length) {
          onChange([...images, ...newImages])
        }
      }
      reader.onerror = () => {
        loaded++
        toast.error(t('Failed to read the image'))
        if (loaded === toProcess.length && newImages.length > 0) {
          onChange([...images, ...newImages])
        }
      }
      reader.readAsDataURL(file)
    }
  }

  const removeImage = (index: number) => {
    onChange(images.filter((_, i) => i !== index))
  }

  const canAdd = images.length < MAX_REFERENCE_IMAGES && !disabled

  return (
    <div className='space-y-2'>
      <div className='grid grid-cols-3 gap-2'>
        {images.map((src, i) => (
          <div
            key={i}
            className='bg-muted/40 relative overflow-hidden rounded-lg border'
          >
            <img
              src={src}
              alt={t('Reference image {{index}}', { index: i + 1 })}
              className='aspect-square w-full object-cover'
            />
            {!disabled && (
              <button
                type='button'
                onClick={() => removeImage(i)}
                className='bg-background/80 hover:bg-background absolute top-1 right-1 rounded-full border p-0.5'
                aria-label={t('Remove image')}
              >
                <XIcon size={12} />
              </button>
            )}
          </div>
        ))}

        {canAdd && (
          <button
            type='button'
            onClick={() => fileRef.current?.click()}
            className='border-muted-foreground/30 hover:border-muted-foreground/60 text-muted-foreground flex aspect-square flex-col items-center justify-center gap-1 rounded-lg border border-dashed text-xs transition-colors'
          >
            <PlusIcon size={18} />
            <span>{t('Add')}</span>
          </button>
        )}
      </div>

      <p className='text-muted-foreground text-xs'>
        {t('{{current}}/{{max}} reference images (optional)', {
          current: images.length,
          max: MAX_REFERENCE_IMAGES,
        })}
      </p>

      <input
        ref={fileRef}
        type='file'
        accept='image/*'
        multiple
        className='hidden'
        onChange={handleFiles}
      />
    </div>
  )
}
