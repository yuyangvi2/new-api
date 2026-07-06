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

// Model and group options (mirror the shape consumed by ModelGroupSelector)
export interface ModelOption {
  label: string
  value: string
  groups?: string[]
  modelRatio?: number
}

export interface GroupOption {
  label: string
  value: string
  ratio?: number
  desc?: string
}

// Request payload for task-based image generation (image-gi etc.)
export interface ImageTaskRequest {
  model: string
  group?: string
  prompt: string
  // Reference images: URLs or raw base64 strings
  images?: string[]
  size?: string
  metadata?: Record<string, unknown>
}

// Request payload sent to /pg/images/generations
export interface ImageGenerationRequest {
  model: string
  // Billing group, honored by the playground distributor.
  group?: string
  prompt: string
  n?: number
  size?: string
  quality?: string
  response_format?: 'url' | 'b64_json'
}

// Single image returned by the backend
export interface ImageDataItem {
  url?: string
  b64_json?: string
  revised_prompt?: string
}

export interface ImageGenerationResponse {
  created: number
  data: ImageDataItem[]
}

// A locally-rendered result (one card per image)
export interface GeneratedImage {
  id: string
  // Display-ready src (either remote url or data: URI from b64)
  src: string
  prompt: string
  model: string
  size: string
  createdAt: number
}

// A generation batch lifecycle entry shown in the gallery
export type BatchStatus = 'loading' | 'complete' | 'error'

export interface GenerationBatch {
  id: string
  status: BatchStatus
  prompt: string
  model: string
  size: string
  count: number
  images: GeneratedImage[]
  errorMessage?: string
  createdAt: number
}

// Persisted/in-memory generator configuration
export interface GeneratorConfig {
  model: string
  group: string
  prompt: string
  size: string
  quality: string
  n: number
  // Reference images (URLs or data: URIs) for image-gi models
  images: string[]
  // Model-family-specific parameters, keyed by PascalCase upstream names
  metadata: Record<string, unknown>
}

// ---------------------------------------------------------------------------
// Image-to-video
// ---------------------------------------------------------------------------

// Top-level page mode toggle.
export type GeneratorMode = 'image' | 'video'

// Where the input image came from.
export type ImageSourceType = 'upload' | 'url' | 'generated'

// Request payload sent to /pg/video/generations
export interface VideoGenerationRequest {
  model: string
  group?: string
  prompt?: string
  // Input image: a remote URL, data URI, or raw base64.
  image?: string
  images?: string[]
  size?: string
  duration?: number
  width?: number
  height?: number
  metadata?: Record<string, unknown>
}

// Response of submitting a video task
export interface VideoSubmitResponse {
  task_id: string
  status: string
}

// Response of polling a video task
export interface VideoTaskResponse {
  task_id: string
  status: string
  progress?: string
  url?: string
  format?: string
  error?: { code?: number; message?: string }
}

export type VideoBatchStatus = 'submitting' | 'polling' | 'complete' | 'error'

export interface VideoBatch {
  id: string
  status: VideoBatchStatus
  prompt: string
  model: string
  // Displayable source of the input image (data URI or URL).
  imagePreview?: string
  taskId?: string
  progress?: string
  videoUrl?: string
  errorMessage?: string
  createdAt: number
}

export interface VideoConfig {
  model: string
  group: string
  prompt: string
  // Displayable input image (data URI or URL); normalized at submit time.
  image: string
  imageSourceType: ImageSourceType
  referenceImagesText: string
  referenceVideosText: string
  referenceAudiosText: string
  inputVideoDuration: number
  duration: number
  size: string
  // Model-family-specific parameters, keyed by PascalCase upstream names.
  metadata: Record<string, unknown>
}

export interface VideoModelVariantOption {
  label: string
  value: string
}

export interface VideoModelVariantAxis {
  id: string
  translateLabels?: boolean
  options: VideoModelVariantOption[]
}

export interface VideoModelVariantSet {
  defaultModel: string
  displayName: string
  axes: VideoModelVariantAxis[]
  variants: Record<string, string>
}

export interface VideoModelVariantAxisState extends VideoModelVariantAxis {
  value: string
}

export interface VideoModelVariantState {
  set: VideoModelVariantSet
  selection: Record<string, string>
  axes: VideoModelVariantAxisState[]
}

// ---------------------------------------------------------------------------
// Model-family dynamic parameters
// ---------------------------------------------------------------------------

// Image model families
export type ImageModelFamily =
  | 'dall-e'
  | 'gpt-image'
  | 'image-gi'
  | 'image-gi2'
  | 'hunyuan-image'
  | 'generic-image'

export type ModelFamily = 'kling' | 'vidu' | 'seedance' | 'unknown'

export type ParamType = 'select' | 'slider' | 'text' | 'switch'

export interface FamilyParam {
  key: string // PascalCase key sent in metadata (e.g. "Mode")
  label: string // i18n display label
  type: ParamType
  default: unknown
  options?: { label: string; value: string }[] // for select
  min?: number // for slider
  max?: number
  step?: number
}
