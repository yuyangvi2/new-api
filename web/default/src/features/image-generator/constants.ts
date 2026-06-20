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
import type { GeneratorConfig, VideoConfig } from './types'

// API endpoints
export const API_ENDPOINTS = {
  // Playground proxy: authenticated by the login session (cookie), so no
  // API key is required — a temp token is created server-side, like chat.
  IMAGE_GENERATIONS: '/pg/images/generations',
  VIDEO_SUBMIT: '/pg/video/generations',
  VIDEO_TASK: (taskId: string) => `/pg/video/generations/${taskId}`,
  USER_MODELS: '/api/user/models',
  USER_GROUPS: '/api/user/self/groups',
} as const

// Aspect-ratio presets mapped to the size string the API expects.
// `label` is shown in the UI, `ratio` drives the preview box shape.
export interface SizePreset {
  label: string
  value: string
  ratioLabel: string
  ratio: number // width / height
}

export const SIZE_PRESETS: SizePreset[] = [
  { label: 'Square', value: '1024x1024', ratioLabel: '1:1', ratio: 1 },
  { label: 'Portrait', value: '1024x1792', ratioLabel: '9:16', ratio: 9 / 16 },
  { label: 'Landscape', value: '1792x1024', ratioLabel: '16:9', ratio: 16 / 9 },
  { label: 'Small', value: '512x512', ratioLabel: '1:1', ratio: 1 },
]

export const QUALITY_OPTIONS = [
  { label: 'Standard', value: 'standard' },
  { label: 'HD', value: 'hd' },
] as const

// Number-of-images choices
export const COUNT_OPTIONS = [1, 2, 3, 4] as const

export const MAX_PROMPT_LENGTH = 4000

export const DEFAULT_GROUP = 'default' as const

export const DEFAULT_CONFIG: GeneratorConfig = {
  model: 'dall-e-3',
  group: DEFAULT_GROUP,
  prompt: '',
  size: '1024x1024',
  quality: 'standard',
  n: 1,
}

// Storage key for persisting non-sensitive config (prompt/model/size/etc.)
export const STORAGE_KEY = 'image_generator_config'

// Example prompts surfaced as quick-start chips.
export const EXAMPLE_PROMPTS = [
  'A serene Japanese garden at sunrise, soft mist, cinematic lighting',
  'A futuristic city skyline at night, neon reflections, ultra detailed',
  'Cute corgi astronaut floating in space, digital art, vibrant colors',
  'Watercolor portrait of a fox in an autumn forest',
] as const

// ---------------------------------------------------------------------------
// Image-to-video
// ---------------------------------------------------------------------------

// Video size presets map an aspect ratio to width/height the API expects.
export interface VideoSizePreset {
  label: string
  ratioLabel: string
  width: number
  height: number
  ratio: number // width / height
}

export const VIDEO_SIZE_PRESETS: VideoSizePreset[] = [
  { label: 'Square', ratioLabel: '1:1', width: 720, height: 720, ratio: 1 },
  {
    label: 'Landscape',
    ratioLabel: '16:9',
    width: 1280,
    height: 720,
    ratio: 16 / 9,
  },
  {
    label: 'Portrait',
    ratioLabel: '9:16',
    width: 720,
    height: 1280,
    ratio: 9 / 16,
  },
]

export const VIDEO_DURATIONS = [5, 10] as const

// Max input image size for upload (bytes). Larger files are rejected.
export const MAX_IMAGE_UPLOAD_BYTES = 10 * 1024 * 1024

// Polling cadence and ceiling for async video tasks.
export const VIDEO_POLL_INTERVAL_MS = 3000
export const VIDEO_POLL_TIMEOUT_MS = 10 * 60 * 1000

export const VIDEO_STORAGE_KEY = 'video_generator_config'

export const DEFAULT_VIDEO_CONFIG: VideoConfig = {
  model: 'kling-v1',
  group: DEFAULT_GROUP,
  prompt: '',
  image: '',
  imageSourceType: 'upload',
  duration: 5,
  size: '1280x720',
}

// Terminal task statuses reported by the backend (case-insensitive).
export const VIDEO_SUCCESS_STATUSES = ['succeeded', 'success', 'completed']
export const VIDEO_FAILED_STATUSES = ['failed', 'error', 'cancelled', 'canceled']
