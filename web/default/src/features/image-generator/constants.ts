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
import type {
  FamilyParam,
  GeneratorConfig,
  ImageModelFamily,
  ModelFamily,
  VideoConfig,
  VideoModelVariantAxisState,
  VideoModelVariantSet,
  VideoModelVariantState,
} from './types'

// API endpoints
export const API_ENDPOINTS = {
  // Playground proxy: authenticated by the login session (cookie), so no
  // API key is required — a temp token is created server-side, like chat.
  IMAGE_GENERATIONS: '/pg/images/generations',
  // Task-based image generation (for async models like image-gi)
  IMAGE_TASK_SUBMIT: '/pg/video/generations',
  IMAGE_TASK: (taskId: string) => `/pg/video/generations/${taskId}`,
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
  images: [],
  metadata: {},
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
  metadata: {},
}

export const VIDEO_MODEL_VARIANT_SETS: VideoModelVariantSet[] = [
  {
    defaultModel: 'seedance2.0_direct',
    displayName: 'seedance2.0',
    axes: [
      {
        id: 'quality',
        translateLabels: false,
        options: [
          { label: 'Direct', value: 'direct' },
          { label: 'Vision', value: 'vision' },
        ],
      },
      {
        id: 'speed',
        translateLabels: false,
        options: [
          { label: 'Standard', value: 'standard' },
          { label: 'Fast', value: 'fast' },
        ],
      },
    ],
    variants: {
      'direct:standard': 'seedance2.0_direct',
      'direct:fast': 'seedance2.0_fast_direct',
      'vision:standard': 'seedance2.0_vision',
      'vision:fast': 'seedance2.0_fast_vision',
    },
  },
]

function getVariantKey(
  set: VideoModelVariantSet,
  selection: Record<string, string>
): string {
  return set.axes.map((axis) => selection[axis.id]).join(':')
}

function getDefaultSelection(
  set: VideoModelVariantSet
): Record<string, string> {
  return Object.fromEntries(
    set.axes.map((axis) => [axis.id, axis.options[0]?.value || ''])
  )
}

export function getVideoVariantSet(
  model: string
): VideoModelVariantSet | undefined {
  return VIDEO_MODEL_VARIANT_SETS.find((set) =>
    Object.values(set.variants).includes(model)
  )
}

export function getVideoVariantDisplayName(model: string): string | undefined {
  const set = getVideoVariantSet(model)
  return set?.displayName
}

export function getSelectionForVideoModel(
  set: VideoModelVariantSet,
  model: string
): Record<string, string> {
  const match = Object.entries(set.variants).find(([, m]) => m === model)
  if (!match) return getDefaultSelection(set)
  const values = match[0].split(':')
  return Object.fromEntries(set.axes.map((axis, i) => [axis.id, values[i]]))
}

export function isHiddenVideoVariantModel(
  model: string,
  availableModels: string[]
): boolean {
  const set = getVideoVariantSet(model)
  if (!set || model === set.defaultModel) return false
  return availableModels.includes(set.defaultModel)
}

export function getVideoModelVariantState(
  model: string,
  availableModels: string[]
): VideoModelVariantState | undefined {
  const set = getVideoVariantSet(model)
  if (!set) return undefined

  const availableSet = new Set(availableModels)
  const selection = getSelectionForVideoModel(set, model)
  const axes: VideoModelVariantAxisState[] = set.axes
    .map((axis) => {
      const options = axis.options.filter((option) => {
        const nextSelection = { ...selection, [axis.id]: option.value }
        const nextModel = set.variants[getVariantKey(set, nextSelection)]
        return !!nextModel && availableSet.has(nextModel)
      })
      return { id: axis.id, value: selection[axis.id], options }
    })
    .filter((axis) => axis.options.length > 1)

  if (axes.length === 0) return undefined

  return { set, selection, axes }
}

export function resolveVideoVariantModel(
  model: string,
  axisId: string,
  value: string,
  availableModels: string[]
): string | undefined {
  const set = getVideoVariantSet(model)
  if (!set) return undefined
  const selection = getSelectionForVideoModel(set, model)
  const nextSelection = { ...selection, [axisId]: value }
  const nextModel = set.variants[getVariantKey(set, nextSelection)]
  if (!nextModel || !availableModels.includes(nextModel)) return undefined
  return nextModel
}

const IMAGE_OPTIONAL_VIDEO_MODELS = new Set([
  'doubao-seedance-2-0-260128',
  'doubao-seedance-2-0-fast-260128',
  'dreamina-seedance-2-0-260128',
  'dreamina-seedance-2-0-fast-260128',
  'seedance2.0_direct',
  'seedance2.0_fast_direct',
])

const IMAGE_REQUIRED_VIDEO_MODELS = new Set([
  'seedance2.0_vision',
  'seedance2.0_fast_vision',
])

export function videoModelRequiresImage(model: string): boolean {
  if (IMAGE_REQUIRED_VIDEO_MODELS.has(model)) return true
  if (IMAGE_OPTIONAL_VIDEO_MODELS.has(model)) return false
  return false
}

export function videoModelSupportsImageInput(model: string): boolean {
  if (IMAGE_REQUIRED_VIDEO_MODELS.has(model)) return true
  if (model === 'seedance2.0_direct') return false
  if (model === 'seedance2.0_fast_direct') return false
  return true
}

// Terminal task statuses reported by the backend (case-insensitive).
export const VIDEO_SUCCESS_STATUSES = ['succeeded', 'success', 'completed']
export const VIDEO_FAILED_STATUSES = [
  'failed',
  'failure',
  'error',
  'cancelled',
  'canceled',
]

// ---------------------------------------------------------------------------
// Model-family detection & per-family parameter schemas
// ---------------------------------------------------------------------------

const KLING_RE = /kling/i
const VIDU_RE = /vidu/i
const SEEDANCE_RE = /seedance|doubao-seedance/i

export function detectModelFamily(model: string): ModelFamily {
  if (KLING_RE.test(model)) return 'kling'
  if (VIDU_RE.test(model)) return 'vidu'
  if (SEEDANCE_RE.test(model)) return 'seedance'
  return 'unknown'
}

// ---------------------------------------------------------------------------
// Image model-family detection & per-family parameter schemas
// ---------------------------------------------------------------------------

const DALLE_RE = /dall-e/i
const GPT_IMAGE_RE = /gpt-image/i
const IMAGE_GI2_RE = /image-gi-?2/i
const IMAGE_GI_RE = /image-gi/i
const HUNYUAN_IMAGE_RE = /hunyuan-image/i

export function detectImageModelFamily(model: string): ImageModelFamily {
  if (GPT_IMAGE_RE.test(model)) return 'gpt-image'
  if (DALLE_RE.test(model)) return 'dall-e'
  if (HUNYUAN_IMAGE_RE.test(model)) return 'hunyuan-image'
  if (IMAGE_GI2_RE.test(model)) return 'image-gi2'
  if (IMAGE_GI_RE.test(model)) return 'image-gi'
  return 'generic-image'
}

/** Whether the model uses async task-based generation (submit/poll) */
export function isTaskBasedImageModel(family: ImageModelFamily): boolean {
  return family === 'image-gi' || family === 'image-gi2' || family === 'hunyuan-image'
}

/** Whether the model supports reference image input */
export function supportsReferenceImages(family: ImageModelFamily): boolean {
  return family === 'image-gi' || family === 'image-gi2' || family === 'hunyuan-image'
}

/** AIART aspect ratio presets (different from dall-e WxH format) */
export const AIART_ASPECT_RATIOS = [
  { label: '1:1', value: '1:1' },
  { label: '3:4', value: '3:4' },
  { label: '4:3', value: '4:3' },
  { label: '16:9', value: '16:9' },
  { label: '9:16', value: '9:16' },
  { label: '2:3', value: '2:3' },
  { label: '3:2', value: '3:2' },
] as const

/** 混元生图 3.0 resolution presets (pixel-based, W:H format) */
export const HUNYUAN_IMAGE_RESOLUTIONS = [
  { label: '1024x1024 (1:1)', value: '1024:1024' },
  { label: '768x1024 (3:4)', value: '768:1024' },
  { label: '1024x768 (4:3)', value: '1024:768' },
  { label: '1280x720 (16:9)', value: '1280:720' },
  { label: '720x1280 (9:16)', value: '720:1280' },
  { label: '768x1152 (2:3)', value: '768:1152' },
  { label: '1152x768 (3:2)', value: '1152:768' },
] as const

/** gpt-image size presets */
export const GPT_IMAGE_SIZE_PRESETS = [
  { label: 'Auto', value: 'auto' },
  { label: '1024x1024', value: '1024x1024', ratioLabel: '1:1' },
  { label: '1536x1024', value: '1536x1024', ratioLabel: '3:2' },
  { label: '1024x1536', value: '1024x1536', ratioLabel: '2:3' },
] as const

export const GPT_IMAGE_QUALITY_OPTIONS = [
  { label: 'Auto', value: 'auto' },
  { label: 'Low', value: 'low' },
  { label: 'Medium', value: 'medium' },
  { label: 'High', value: 'high' },
] as const

export const GPT_IMAGE_BACKGROUND_OPTIONS = [
  { label: 'Auto', value: 'auto' },
  { label: 'Transparent', value: 'transparent' },
  { label: 'Opaque', value: 'opaque' },
] as const

export const IMAGE_FAMILY_PARAMS: Record<ImageModelFamily, FamilyParam[]> = {
  'dall-e': [],      // dall-e quality/n handled inline
  'gpt-image': [
    {
      key: 'background',
      label: 'Background',
      type: 'select',
      default: 'auto',
      options: [
        { label: 'Auto', value: 'auto' },
        { label: 'Transparent', value: 'transparent' },
        { label: 'Opaque', value: 'opaque' },
      ],
    },
  ],
  'image-gi': [
    {
      key: 'ImageSearch',
      label: 'Image search',
      type: 'switch',
      default: false,
    },
  ],
  'image-gi2': [
    {
      key: 'ThinkingLevel',
      label: 'Thinking level',
      type: 'select',
      default: 'default',
      options: [
        { label: 'Default', value: 'default' },
        { label: 'Think', value: 'think' },
      ],
    },
    {
      key: 'ImageSearch',
      label: 'Image search',
      type: 'switch',
      default: false,
    },
  ],
  'hunyuan-image': [],
  'generic-image': [],
}

// Task-based image generation polling config
export const IMAGE_TASK_POLL_INTERVAL_MS = 3000
export const IMAGE_TASK_POLL_TIMEOUT_MS = 5 * 60 * 1000
export const IMAGE_TASK_SUCCESS_STATUSES = ['succeeded', 'success', 'completed']
export const IMAGE_TASK_FAILED_STATUSES = [
  'failed',
  'failure',
  'error',
  'cancelled',
  'canceled',
]

// Max reference images for AIART
export const MAX_REFERENCE_IMAGES = 5

export const FAMILY_PARAMS: Record<ModelFamily, FamilyParam[]> = {
  kling: [
    {
      key: 'Mode',
      label: 'Generation mode',
      type: 'select',
      default: 'std',
      options: [
        { label: 'Standard', value: 'std' },
        { label: 'Pro', value: 'pro' },
      ],
    },
    {
      key: 'CfgScale',
      label: 'Prompt adherence',
      type: 'slider',
      default: 0.5,
      min: 0,
      max: 1,
      step: 0.1,
    },
    {
      key: 'NegativePrompt',
      label: 'Negative prompt',
      type: 'text',
      default: '',
    },
  ],
  vidu: [
    {
      key: 'Resolution',
      label: 'Resolution',
      type: 'select',
      default: '720p',
      options: [
        { label: '540p', value: '540p' },
        { label: '720p', value: '720p' },
        { label: '1080p', value: '1080p' },
      ],
    },
    {
      key: 'MovementAmplitude',
      label: 'Movement amplitude',
      type: 'select',
      default: 'auto',
      options: [
        { label: 'Auto', value: 'auto' },
        { label: 'Small', value: 'small' },
        { label: 'Medium', value: 'medium' },
        { label: 'Large', value: 'large' },
      ],
    },
  ],
  seedance: [
    {
      key: 'resolution',
      label: 'Resolution',
      type: 'select',
      default: '720p',
      options: [
        { label: '480p', value: '480p' },
        { label: '720p', value: '720p' },
        { label: '1080p', value: '1080p' },
        { label: '4K', value: '4k' },
      ],
    },
  ],
  unknown: [],
}
