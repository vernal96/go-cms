import { adminRequest } from '../../api/admin-api'
import type { FilesystemItem } from '../../types/admin'
export interface ImageTransform {
  crop?: { x: number; y: number; width: number; height: number }
  rotate: number; scale_x: number; scale_y: number; width: number; height: number
  fit: 'contain' | 'cover' | 'stretch'; position: string; quality: number
}
export interface ImageState {
  media_id: number; current_file: FilesystemItem; original_file: FilesystemItem
  expected_updated_at: string; transform: ImageTransform | null; can_restore: boolean; editable: boolean
  limits: { output_dimension: number; output_pixels: number; min_quality: number; max_quality: number }
}
export const defaultTransform = (): ImageTransform => ({ rotate: 0, scale_x: 1, scale_y: 1, width: 0, height: 0, fit: 'contain', position: 'center', quality: 85 })
export function imageState(base: string, token: string) { return adminRequest<ImageState>(base, token) }
export function saveImage(base: string, token: string, state: ImageState, transform: ImageTransform) {
  return adminRequest<ImageState>(base, token, { method: 'POST', body: JSON.stringify({ expected_updated_at: state.expected_updated_at, transform }) })
}
export function restoreImage(base: string, token: string, state: ImageState) {
  return adminRequest<ImageState>(`${base}/restore`, token, { method: 'POST', body: JSON.stringify({ expected_updated_at: state.expected_updated_at }) })
}
