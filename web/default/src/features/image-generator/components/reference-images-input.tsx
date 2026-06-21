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
import { PlusIcon, UploadIcon, XIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
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
  const [urlText, setUrlText] = useState('')

  const canAdd = images.length < MAX_REFERENCE_IMAGES && !disabled

  const addImage = (src: string) => {
    if (images.length >= MAX_REFERENCE_IMAGES) {
      toast.error(t('Maximum {{count}} reference images', { count: MAX_REFERENCE_IMAGES }))
      return
    }
    onChange([...images, src])
  }

  const removeImage = (index: number) => {
    onChange(images.filter((_, i) => i !== index))
  }

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
    reader.onload = () => {
      addImage(String(reader.result))
    }
    reader.onerror = () => toast.error(t('Failed to read the image'))
    reader.readAsDataURL(file)
  }

  const commitUrl = () => {
    const url = urlText.trim()
    if (url && (url.startsWith('http://') || url.startsWith('https://'))) {
      addImage(url)
      setUrlText('')
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      e.preventDefault()
      commitUrl()
    }
  }

  return (
    <div className='space-y-2'>
      {/* Input row: URL field + upload button */}
      {canAdd && (
        <div className='relative flex items-center'>
          <Input
            value={urlText}
            onChange={(e) => setUrlText(e.target.value)}
            onBlur={commitUrl}
            onKeyDown={handleKeyDown}
            placeholder={t('Paste image URL')}
            disabled={disabled}
            className='pr-9'
          />
          <button
            type='button'
            disabled={disabled}
            onClick={() => fileRef.current?.click()}
            className='text-muted-foreground hover:text-foreground absolute right-1.5 rounded-md p-1 transition-colors disabled:opacity-50'
            aria-label={t('Upload image')}
          >
            <UploadIcon size={16} />
          </button>
        </div>
      )}

      {/* Image previews */}
      {images.length > 0 && (
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
                  className='bg-background/80 hover:bg-background absolute top-1 right-1 rounded-full border p-0.5 transition-colors'
                  aria-label={t('Remove image')}
                >
                  <XIcon size={12} />
                </button>
              )}
            </div>
          ))}
        </div>
      )}

      <p className='text-muted-foreground text-xs'>
        {t('{{current}}/{{max}} reference images (optional)', {
          current: images.length,
          max: MAX_REFERENCE_IMAGES,
        })}
      </p>

      {/* Hidden file input */}
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
