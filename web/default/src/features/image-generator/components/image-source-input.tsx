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
import { useEffect, useRef, useState } from 'react'
import { UploadIcon, XIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Input } from '@/components/ui/input'
import { MAX_IMAGE_UPLOAD_BYTES } from '../constants'
import type { ImageSourceType } from '../types'

interface ImageSourceInputProps {
  value: string
  sourceType: ImageSourceType
  onChange: (src: string, sourceType: ImageSourceType) => void
  /** @deprecated No longer used in the new layout. Kept for API compat. */
  availableImages?: { id: string; src: string }[]
  disabled?: boolean
  placeholder?: string
}

/**
 * Compact image input: a text field (for pasting a URL) with an upload
 * icon-button on the right. Below the field, a preview thumbnail appears
 * once an image is set, with a close button to clear it.
 */
export function ImageSourceInput({
  value,
  sourceType,
  onChange,
  disabled,
  placeholder,
}: ImageSourceInputProps) {
  const { t } = useTranslation()
  const fileRef = useRef<HTMLInputElement>(null)

  // The text shown in the input: either the URL or a short label for uploads.
  const [inputText, setInputText] = useState(() => displayText(value, sourceType))

  // Keep the displayed text in sync when the value is changed externally.
  useEffect(() => {
    setInputText(displayText(value, sourceType))
  }, [value, sourceType])

  // ---- handlers ----

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
      const dataUri = String(reader.result)
      onChange(dataUri, 'upload')
    }
    reader.onerror = () => toast.error(t('Failed to read the image'))
    reader.readAsDataURL(file)
  }

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setInputText(e.target.value)
  }

  const commitUrl = () => {
    const url = inputText.trim()
    if (url && (url.startsWith('http://') || url.startsWith('https://'))) {
      onChange(url, 'url')
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      e.preventDefault()
      commitUrl()
    }
  }

  const clear = () => {
    onChange('', 'upload')
    setInputText('')
  }

  return (
    <div className='space-y-2'>
      {/* Input row: text field + upload button */}
      <div className='relative flex items-center'>
        <Input
          value={inputText}
          onChange={handleInputChange}
          onBlur={commitUrl}
          onKeyDown={handleKeyDown}
          placeholder={placeholder ?? t('Paste image URL')}
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

      {/* Preview */}
      {value && (
        <div className='bg-muted/40 relative overflow-hidden rounded-lg border'>
          <img
            src={value}
            alt={t('Input image')}
            className='max-h-48 w-full object-contain'
          />
          {!disabled && (
            <button
              type='button'
              onClick={clear}
              className='bg-background/80 hover:bg-background absolute top-1.5 right-1.5 rounded-full border p-1 transition-colors'
              aria-label={t('Remove image')}
            >
              <XIcon size={14} />
            </button>
          )}
        </div>
      )}

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

/** Derive the text to display in the input field. */
function displayText(value: string, sourceType: ImageSourceType): string {
  if (!value) return ''
  if (sourceType === 'url' || value.startsWith('http')) return value
  // For uploaded / data-URI images, show a truncated indicator.
  if (value.startsWith('data:')) {
    const semi = value.indexOf(';')
    const mime = semi > 5 ? value.slice(5, semi) : 'image'
    return `[${mime}]`
  }
  return value
}
