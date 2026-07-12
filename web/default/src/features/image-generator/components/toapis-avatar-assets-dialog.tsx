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
import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ChangeEvent,
  type MouseEvent,
} from 'react'
import {
  CheckIcon,
  ImageIcon,
  Loader2Icon,
  RefreshCwIcon,
  Trash2Icon,
  UploadIcon,
  UserRoundIcon,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { cn } from '@/lib/utils'

import {
  createToAPIsAvatarAsset,
  createToAPIsAvatarGroup,
  deleteToAPIsAvatarAsset,
  deleteToAPIsAvatarGroup,
  listToAPIsAvatarAssets,
  listToAPIsAvatarGroups,
  refreshToAPIsAvatarAssets,
  uploadReferenceMedia,
} from '../api'
import { MAX_IMAGE_UPLOAD_BYTES } from '../constants'
import type { ToAPIsAvatarAsset, ToAPIsAvatarGroup } from '../types'

interface ToAPIsAvatarAssetsDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  selectedValues: string[]
  maxItems: number
  onSelect: (value: string) => void
}

type AssetCount = {
  available: number
  total: number
}

export function ToAPIsAvatarAssetsDialog(
  props: ToAPIsAvatarAssetsDialogProps
) {
  const { t } = useTranslation()
  const uploadInputRef = useRef<HTMLInputElement>(null)
  const [groups, setGroups] = useState<ToAPIsAvatarGroup[]>([])
  const [allAssets, setAllAssets] = useState<ToAPIsAvatarAsset[]>([])
  const [selectedGroupId, setSelectedGroupId] = useState('')
  const [groupName, setGroupName] = useState('')
  const [groupDescription, setGroupDescription] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [isCreatingGroup, setIsCreatingGroup] = useState(false)
  const [isUploadingAsset, setIsUploadingAsset] = useState(false)
  const [isRefreshingAssets, setIsRefreshingAssets] = useState(false)
  const [deletingGroupId, setDeletingGroupId] = useState('')
  const [deletingAssetId, setDeletingAssetId] = useState('')

  const selectedSet = useMemo(
    () => new Set(props.selectedValues),
    [props.selectedValues]
  )
  const limitReached = props.selectedValues.length >= props.maxItems

  const selectedGroup = useMemo(
    () => groups.find((group) => group.group_id === selectedGroupId),
    [groups, selectedGroupId]
  )

  const imageAssets = useMemo(
    () => allAssets.filter((asset) => asset.asset_type === 'image'),
    [allAssets]
  )

  const selectedGroupAssets = useMemo(
    () => imageAssets.filter((asset) => asset.group_id === selectedGroupId),
    [imageAssets, selectedGroupId]
  )

  const countsByGroupId = useMemo(() => {
    const counts = new Map<string, AssetCount>()
    for (const asset of imageAssets) {
      const count = counts.get(asset.group_id) ?? { available: 0, total: 0 }
      count.total += 1
      if (asset.status === 'active') {
        count.available += 1
      }
      counts.set(asset.group_id, count)
    }
    return counts
  }, [imageAssets])

  const loadLibrary = useCallback(async () => {
    setIsLoading(true)
    try {
      const [nextGroups, nextAssets] = await Promise.all([
        listToAPIsAvatarGroups(),
        listToAPIsAvatarAssets(),
      ])
      setGroups(nextGroups)
      setAllAssets(nextAssets)
      setSelectedGroupId((current) => {
        if (nextGroups.length === 0) return ''
        if (nextGroups.some((group) => group.group_id === current)) {
          return current
        }
        return nextGroups[0].group_id
      })
    } catch (error: unknown) {
      toast.error(error instanceof Error ? error.message : t('Load failed'))
    } finally {
      setIsLoading(false)
    }
  }, [t])

  useEffect(() => {
    if (!props.open) return
    void loadLibrary()
  }, [loadLibrary, props.open])

  useEffect(() => {
    if (props.open) return
    setGroupName('')
    setGroupDescription('')
  }, [props.open])

  const handleCreateGroup = async () => {
    const name = groupName.trim()
    if (!name) {
      toast.error(t('Group name is required'))
      return
    }
    setIsCreatingGroup(true)
    try {
      const group = await createToAPIsAvatarGroup({
        name,
        description: groupDescription.trim() || undefined,
      })
      setGroups((prev) => [
        group,
        ...prev.filter((item) => item.group_id !== group.group_id),
      ])
      setSelectedGroupId(group.group_id)
      setGroupName('')
      setGroupDescription('')
      toast.success(t('Created'))
    } catch (error: unknown) {
      toast.error(error instanceof Error ? error.message : t('Create failed'))
    } finally {
      setIsCreatingGroup(false)
    }
  }

  const handleUploadAsset = async (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0]
    event.target.value = ''
    if (!file) return
    if (!selectedGroupId) {
      toast.error(t('Please create or select an avatar asset group first.'))
      return
    }
    if (file.size > MAX_IMAGE_UPLOAD_BYTES) {
      toast.error(
        t('File is too large (max {{size}})', {
          size: `${Math.round(MAX_IMAGE_UPLOAD_BYTES / 1024 / 1024)} MB`,
        })
      )
      return
    }

    setIsUploadingAsset(true)
    try {
      const uploadedUrl = await uploadReferenceMedia(file, 'image')
      const asset = await createToAPIsAvatarAsset({
        group_id: selectedGroupId,
        source_url: uploadedUrl,
        name: file.name.replace(/\.[^.]+$/, ''),
      })
      setAllAssets((prev) => [
        asset,
        ...prev.filter((item) => item.asset_id !== asset.asset_id),
      ])
      toast.success(t('Upload complete'))
    } catch (error: unknown) {
      toast.error(error instanceof Error ? error.message : t('Upload failed'))
    } finally {
      setIsUploadingAsset(false)
    }
  }

  const handleRefreshAssets = async () => {
    if (!selectedGroupId) return
    setIsRefreshingAssets(true)
    try {
      const nextAssets = await refreshToAPIsAvatarAssets(selectedGroupId)
      setAllAssets((prev) => [
        ...nextAssets,
        ...prev.filter((asset) => asset.group_id !== selectedGroupId),
      ])
      toast.success(t('Refreshed'))
    } catch (error: unknown) {
      toast.error(error instanceof Error ? error.message : t('Refresh failed'))
    } finally {
      setIsRefreshingAssets(false)
    }
  }

  const handleDeleteGroup = async (
    group: ToAPIsAvatarGroup,
    event: MouseEvent<HTMLButtonElement>
  ) => {
    event.stopPropagation()
    if (!window.confirm(t('Delete this avatar asset group?'))) return

    setDeletingGroupId(group.group_id)
    try {
      await deleteToAPIsAvatarGroup(group.group_id)
      const nextGroups = groups.filter((item) => item.group_id !== group.group_id)
      setGroups(nextGroups)
      setAllAssets((prev) =>
        prev.filter((asset) => asset.group_id !== group.group_id)
      )
      if (selectedGroupId === group.group_id) {
        setSelectedGroupId(nextGroups[0]?.group_id ?? '')
      }
      toast.success(t('Deleted'))
    } catch (error: unknown) {
      toast.error(error instanceof Error ? error.message : t('Delete failed'))
    } finally {
      setDeletingGroupId('')
    }
  }

  const handleDeleteAsset = async (
    asset: ToAPIsAvatarAsset,
    event: MouseEvent<HTMLButtonElement>
  ) => {
    event.stopPropagation()
    if (!window.confirm(t('Delete this avatar asset?'))) return

    setDeletingAssetId(asset.asset_id)
    try {
      await deleteToAPIsAvatarAsset(asset.asset_id)
      setAllAssets((prev) =>
        prev.filter((item) => item.asset_id !== asset.asset_id)
      )
      toast.success(t('Deleted'))
    } catch (error: unknown) {
      toast.error(error instanceof Error ? error.message : t('Delete failed'))
    } finally {
      setDeletingAssetId('')
    }
  }

  const handleSelectAsset = (asset: ToAPIsAvatarAsset) => {
    const referenceValue = `asset://${asset.asset_id}`
    if (asset.status !== 'active' || selectedSet.has(referenceValue)) return
    if (limitReached) {
      toast.error(t('Reference image limit reached'))
      return
    }
    props.onSelect(referenceValue)
  }

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className='max-h-[calc(100vh-2rem)] overflow-hidden border-[#2b3039] bg-[#151922] p-8 text-zinc-100 shadow-2xl sm:max-w-[1264px]'>
        <DialogHeader className='gap-2 pr-8'>
          <DialogTitle className='text-2xl font-semibold text-zinc-100'>
            {t('Virtual avatar asset library')}
          </DialogTitle>
          <DialogDescription className='text-base text-zinc-400'>
            {t(
              'Uploaded assets can be reused in the Seedance-2 playground for the current account.'
            )}
          </DialogDescription>
        </DialogHeader>

        <div className='grid min-h-0 gap-5 overflow-hidden pt-10 lg:grid-cols-[380px_1fr]'>
          <aside className='flex min-h-0 flex-col rounded-3xl border border-[#303541] bg-[#191e27] p-5'>
            <div className='space-y-3'>
              <h3 className='text-lg font-semibold text-zinc-100'>
                {t('Create asset group')}
              </h3>
              <Input
                value={groupName}
                onChange={(event) => setGroupName(event.target.value)}
                placeholder={t('Enter asset group name')}
                disabled={isCreatingGroup}
                className='h-12 rounded-2xl border-[#303541] bg-[#10151d] px-4 text-base text-zinc-100 placeholder:text-zinc-500 focus-visible:ring-orange-500/40'
              />
              <Textarea
                value={groupDescription}
                onChange={(event) => setGroupDescription(event.target.value)}
                placeholder={t('Enter asset group description, optional')}
                disabled={isCreatingGroup}
                className='min-h-28 rounded-2xl border-[#303541] bg-[#10151d] px-4 py-3 text-base text-zinc-100 placeholder:text-zinc-500 focus-visible:ring-orange-500/40'
              />
              <Button
                type='button'
                disabled={isCreatingGroup || !groupName.trim()}
                onClick={handleCreateGroup}
                className='h-12 rounded-2xl bg-orange-400 px-6 text-base font-semibold text-white hover:bg-orange-500'
              >
                {isCreatingGroup ? (
                  <Loader2Icon className='animate-spin' size={17} />
                ) : null}
                {t('Create asset group')}
              </Button>
            </div>

            <div className='mt-8 flex min-h-0 flex-1 flex-col'>
              <div className='mb-4 flex items-center justify-between gap-2'>
                <h3 className='text-lg font-semibold text-zinc-100'>
                  {t('Asset group history')}
                </h3>
                <Button
                  type='button'
                  variant='ghost'
                  size='icon'
                  disabled={isLoading}
                  onClick={loadLibrary}
                  aria-label={t('Refresh')}
                  className='size-8 text-zinc-400 hover:bg-zinc-800 hover:text-zinc-100'
                >
                  {isLoading ? (
                    <Loader2Icon className='animate-spin' size={18} />
                  ) : (
                    <RefreshCwIcon size={18} />
                  )}
                </Button>
              </div>

              <div className='min-h-0 flex-1 space-y-3 overflow-y-auto pr-1'>
                {groups.length === 0 && (
                  <div className='flex min-h-28 items-center justify-center rounded-2xl border border-dashed border-[#303541] text-sm text-zinc-500'>
                    {isLoading ? t('Loading') : t('No avatar asset groups')}
                  </div>
                )}
                {groups.map((group) => {
                  const count = countsByGroupId.get(group.group_id) ?? {
                    available: 0,
                    total: 0,
                  }
                  const selected = group.group_id === selectedGroupId
                  return (
                    <div
                      key={group.group_id}
                      role='button'
                      tabIndex={0}
                      className={cn(
                        'grid min-h-20 cursor-pointer grid-cols-[1fr_auto] items-center gap-3 rounded-3xl border px-5 py-4 outline-none transition-colors',
                        selected
                          ? 'border-orange-500/70 bg-orange-500/10'
                          : 'border-[#303541] bg-[#10151d] hover:border-zinc-500'
                      )}
                      onClick={() => setSelectedGroupId(group.group_id)}
                      onKeyDown={(event) => {
                        if (event.key === 'Enter' || event.key === ' ') {
                          event.preventDefault()
                          setSelectedGroupId(group.group_id)
                        }
                      }}
                    >
                      <div className='min-w-0 space-y-1'>
                        <p className='truncate text-base font-semibold text-zinc-100'>
                          {group.name || group.group_id}
                        </p>
                        <p className='text-sm text-zinc-400'>
                          {t('{{available}}/{{total}} available', count)}
                        </p>
                      </div>
                      <div className='flex items-center gap-2'>
                        <UserRoundIcon size={18} className='text-zinc-400' />
                        <Button
                          type='button'
                          variant='ghost'
                          size='icon'
                          disabled={deletingGroupId === group.group_id}
                          onClick={(event) => handleDeleteGroup(group, event)}
                          aria-label={t('Delete')}
                          className='size-8 text-zinc-400 hover:bg-zinc-800 hover:text-red-300'
                        >
                          {deletingGroupId === group.group_id ? (
                            <Loader2Icon className='animate-spin' size={16} />
                          ) : (
                            <Trash2Icon size={16} />
                          )}
                        </Button>
                      </div>
                    </div>
                  )
                })}
              </div>
            </div>
          </aside>

          <section className='flex min-h-[560px] min-w-0 flex-col rounded-3xl border border-[#303541] bg-[#10151d] p-5'>
            <div className='flex flex-wrap items-start justify-between gap-4'>
              <div className='min-w-0'>
                <h3 className='truncate text-lg font-semibold text-zinc-100'>
                  {selectedGroup?.name || t('Select group')}
                </h3>
                {selectedGroup && (
                  <p className='mt-1 truncate text-sm text-zinc-400'>
                    {selectedGroup.description || selectedGroup.group_id}
                  </p>
                )}
              </div>
              <div className='flex items-center gap-2'>
                <Button
                  type='button'
                  variant='outline'
                  disabled={!selectedGroupId || isUploadingAsset}
                  onClick={() => uploadInputRef.current?.click()}
                  className='h-12 rounded-2xl border-[#303541] bg-transparent px-5 text-base font-semibold text-zinc-100 hover:bg-zinc-800 hover:text-zinc-100'
                >
                  {isUploadingAsset ? (
                    <Loader2Icon className='animate-spin' size={18} />
                  ) : (
                    <UploadIcon size={18} />
                  )}
                  {t('Upload image')}
                </Button>
                <Button
                  type='button'
                  variant='outline'
                  size='icon'
                  disabled={!selectedGroupId || isRefreshingAssets}
                  onClick={handleRefreshAssets}
                  aria-label={t('Refresh')}
                  className='size-12 rounded-2xl border-[#303541] bg-transparent text-zinc-400 hover:bg-zinc-800 hover:text-zinc-100'
                >
                  {isRefreshingAssets ? (
                    <Loader2Icon className='animate-spin' size={18} />
                  ) : (
                    <RefreshCwIcon size={18} />
                  )}
                </Button>
                <input
                  ref={uploadInputRef}
                  type='file'
                  accept='image/*'
                  className='hidden'
                  onChange={handleUploadAsset}
                />
              </div>
            </div>

            <div className='mt-6 min-h-0 flex-1 overflow-y-auto pr-1'>
              {!selectedGroupId && (
                <div className='flex h-full min-h-64 items-center justify-center rounded-2xl border border-dashed border-[#303541] text-sm text-zinc-500'>
                  {t('Please create or select an avatar asset group first.')}
                </div>
              )}
              {selectedGroupId && selectedGroupAssets.length === 0 && (
                <div className='flex h-full min-h-64 items-center justify-center rounded-2xl border border-dashed border-[#303541] text-sm text-zinc-500'>
                  {isLoading ? t('Loading') : t('No avatar assets')}
                </div>
              )}
              {selectedGroupAssets.length > 0 && (
                <div className='grid gap-4 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4'>
                  {selectedGroupAssets.map((asset) => {
                    const referenceValue = `asset://${asset.asset_id}`
                    const isActive = asset.status === 'active'
                    const isSelected = selectedSet.has(referenceValue)
                    const canUse = isActive && !isSelected && !limitReached
                    return (
                      <div
                        key={asset.asset_id}
                        role='button'
                        tabIndex={isActive ? 0 : -1}
                        aria-disabled={!canUse}
                        className={cn(
                          'group overflow-hidden rounded-3xl border bg-[#151922] text-left outline-none transition-colors',
                          isSelected
                            ? 'border-orange-500/80'
                            : 'border-[#303541] hover:border-zinc-500',
                          canUse ? 'cursor-pointer' : 'cursor-default'
                        )}
                        onClick={() => handleSelectAsset(asset)}
                        onKeyDown={(event) => {
                          if (event.key !== 'Enter' && event.key !== ' ') return
                          event.preventDefault()
                          handleSelectAsset(asset)
                        }}
                      >
                        <div className='relative aspect-square overflow-hidden bg-[#0d1118]'>
                          {asset.source_url ? (
                            <img
                              src={asset.source_url}
                              alt={asset.name || asset.asset_id}
                              className='size-full object-cover'
                            />
                          ) : (
                            <div className='flex size-full items-center justify-center'>
                              <ImageIcon className='text-zinc-600' size={34} />
                            </div>
                          )}
                          <Badge className='absolute top-3 left-3 rounded-md bg-zinc-950/70 text-zinc-100 hover:bg-zinc-950/70'>
                            {t('Virtual avatar asset')}
                          </Badge>
                          {isSelected && (
                            <div className='absolute top-3 right-3 flex size-8 items-center justify-center rounded-full bg-orange-500 text-white'>
                              <CheckIcon size={18} />
                            </div>
                          )}
                        </div>
                        <div className='space-y-1 px-4 py-3'>
                          <div className='flex items-center gap-2'>
                            <p className='min-w-0 flex-1 truncate text-base font-semibold text-zinc-100'>
                              {asset.name || asset.asset_id}
                            </p>
                            <Button
                              type='button'
                              variant='ghost'
                              size='icon'
                              disabled={deletingAssetId === asset.asset_id}
                              onClick={(event) => handleDeleteAsset(asset, event)}
                              aria-label={t('Delete')}
                              className='size-8 shrink-0 text-zinc-400 hover:bg-zinc-800 hover:text-red-300'
                            >
                              {deletingAssetId === asset.asset_id ? (
                                <Loader2Icon
                                  className='animate-spin'
                                  size={16}
                                />
                              ) : (
                                <Trash2Icon size={16} />
                              )}
                            </Button>
                          </div>
                          <p className='text-sm text-zinc-400'>
                            {formatAvatarAssetStatus(asset.status, t)}
                          </p>
                          <p className='truncate text-sm text-zinc-400'>
                            {t('Reusable under the current account')}
                          </p>
                        </div>
                      </div>
                    )
                  })}
                </div>
              )}
            </div>
          </section>
        </div>
      </DialogContent>
    </Dialog>
  )
}

function formatAvatarAssetStatus(
  status: string,
  t: (key: string) => string
): string {
  if (status === 'active') return t('Available')
  if (status === 'processing') return t('Processing')
  if (status === 'failed') return t('Failed')
  return status
}
