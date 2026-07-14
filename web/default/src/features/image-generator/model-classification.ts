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
const VIDEO_MODEL_RE =
  /kling|jimeng|sora|vidu|cogvideo|video|hailuo|wan|seedance|runway|luma|pika|veo|i2v|t2v|s2v/i
const NON_VIDEO_MODEL_RE =
  /embedding|seedream|seed-|suno|lyrics|music|image|dall-e|gpt-image|flux|stable|midjourney|ideogram/i

export function isVideoGenerationModelName(model: string): boolean {
  const name = model.trim()

  if (NON_VIDEO_MODEL_RE.test(name) && !/seedance/i.test(name)) return false
  return VIDEO_MODEL_RE.test(name)
}
