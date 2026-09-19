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
import { UploadIcon, XIcon } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Input } from '@/components/ui/input'

import { uploadReferenceMedia } from '../api'
import { MAX_IMAGE_UPLOAD_BYTES, MAX_REFERENCE_IMAGES } from '../constants'

interface ReferenceImagesInputProps {
  images: string[]
  onChange: (images: string[]) => void
  onUploadingChange?: (isUploading: boolean) => void
  disabled?: boolean
  maxImages?: number
  uploadFiles?: boolean
  allowedMimeTypes?: readonly string[]
}

export function ReferenceImagesInput({
  images,
  onChange,
  onUploadingChange,
  disabled,
  maxImages = MAX_REFERENCE_IMAGES,
  uploadFiles = false,
  allowedMimeTypes,
}: ReferenceImagesInputProps) {
  const { t } = useTranslation()
  const fileRef = useRef<HTMLInputElement>(null)
  const uploadControllerRef = useRef<AbortController | null>(null)
  const uploadingChangeRef = useRef(onUploadingChange)
  uploadingChangeRef.current = onUploadingChange
  const latestPropsRef = useRef({ images, maxImages, onChange, disabled })
  latestPropsRef.current = { images, maxImages, onChange, disabled }
  const [urlText, setUrlText] = useState('')
  const [isUploading, setIsUploading] = useState(false)

  useEffect(() => {
    return () => {
      const controller = uploadControllerRef.current
      uploadControllerRef.current = null
      controller?.abort()
      uploadingChangeRef.current?.(false)
    }
  }, [])

  useEffect(() => {
    if (uploadFiles || !uploadControllerRef.current) return

    const controller = uploadControllerRef.current
    uploadControllerRef.current = null
    controller.abort()
    setIsUploading(false)
    uploadingChangeRef.current?.(false)
  }, [uploadFiles])

  const canAdd = images.length < maxImages && !disabled && !isUploading

  const addImage = (src: string) => {
    const latest = latestPropsRef.current
    if (latest.images.includes(src)) return
    if (latest.disabled || latest.images.length >= latest.maxImages) {
      if (!latest.disabled) {
        toast.error(
          t('Maximum {{count}} reference images', {
            count: latest.maxImages,
          })
        )
      }
      return
    }
    latest.onChange([...latest.images, src])
  }

  const removeImage = (src: string) => {
    onChange(images.filter((image) => image !== src))
  }

  const handleFile = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    e.target.value = ''
    if (!file) return
    if (!file.type.startsWith('image/')) {
      toast.error(t('Please choose an image file'))
      return
    }
    if (
      allowedMimeTypes &&
      !allowedMimeTypes.includes(file.type.toLowerCase())
    ) {
      toast.error(t('Tencent VOD supports JPEG, PNG, and WebP images only'))
      return
    }
    if (file.size > MAX_IMAGE_UPLOAD_BYTES) {
      toast.error(t('Image is too large (max 10MB)'))
      return
    }
    if (uploadFiles) {
      const controller = new AbortController()
      uploadControllerRef.current = controller
      setIsUploading(true)
      uploadingChangeRef.current?.(true)
      try {
        const uploadedURL = await uploadReferenceMedia(
          file,
          'image',
          controller.signal
        )
        if (!controller.signal.aborted) {
          addImage(uploadedURL)
        }
      } catch (error: unknown) {
        if (controller.signal.aborted) return
        const message =
          error instanceof Error ? error.message : t('Upload failed')
        toast.error(message)
      } finally {
        if (uploadControllerRef.current === controller) {
          uploadControllerRef.current = null
          setIsUploading(false)
          uploadingChangeRef.current?.(false)
        }
      }
      return
    }
    const reader = new FileReader()
    reader.addEventListener('load', () => {
      addImage(String(reader.result))
    })
    reader.addEventListener('error', () => {
      toast.error(t('Failed to read the image'))
    })
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
            disabled={disabled || isUploading}
            className='pr-9'
          />
          <button
            type='button'
            disabled={disabled || isUploading}
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
          {[...new Set(images)].map((src) => (
            <div
              key={src}
              className='bg-muted/40 relative overflow-hidden rounded-lg border'
            >
              <img
                src={src}
                alt={t('Reference image {{index}}', {
                  index: images.indexOf(src) + 1,
                })}
                className='aspect-square w-full object-cover'
              />
              {!disabled && (
                <button
                  type='button'
                  onClick={() => removeImage(src)}
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
          max: maxImages,
        })}
      </p>

      {/* Hidden file input */}
      <input
        ref={fileRef}
        type='file'
        accept={allowedMimeTypes?.join(',') ?? 'image/*'}
        className='hidden'
        onChange={handleFile}
      />
    </div>
  )
}
