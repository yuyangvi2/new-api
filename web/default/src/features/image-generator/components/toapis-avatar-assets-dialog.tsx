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
import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  CheckIcon,
  Loader2Icon,
  PlusIcon,
  RefreshCwIcon,
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
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  createToAPIsAvatarAsset,
  createToAPIsAvatarGroup,
  listToAPIsAvatarAssets,
  listToAPIsAvatarGroups,
  refreshToAPIsAvatarAssets,
} from '../api'
import type { ToAPIsAvatarAsset, ToAPIsAvatarGroup } from '../types'

interface ToAPIsAvatarAssetsDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  selectedValues: string[]
  maxItems: number
  onSelect: (value: string) => void
}

export function ToAPIsAvatarAssetsDialog(
  props: ToAPIsAvatarAssetsDialogProps
) {
  const { t } = useTranslation()
  const [groups, setGroups] = useState<ToAPIsAvatarGroup[]>([])
  const [assets, setAssets] = useState<ToAPIsAvatarAsset[]>([])
  const [selectedGroupId, setSelectedGroupId] = useState('')
  const [groupName, setGroupName] = useState('')
  const [groupDescription, setGroupDescription] = useState('')
  const [assetName, setAssetName] = useState('')
  const [sourceUrl, setSourceUrl] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [isCreatingGroup, setIsCreatingGroup] = useState(false)
  const [isCreatingAsset, setIsCreatingAsset] = useState(false)
  const [isRefreshing, setIsRefreshing] = useState(false)

  const selectedSet = useMemo(
    () => new Set(props.selectedValues),
    [props.selectedValues]
  )
  const limitReached = props.selectedValues.length >= props.maxItems

  const loadGroups = useCallback(async () => {
    setIsLoading(true)
    try {
      const nextGroups = await listToAPIsAvatarGroups()
      setGroups(nextGroups)
      setSelectedGroupId((current) => {
        if (
          nextGroups.length > 0 &&
          !nextGroups.some((group) => group.group_id === current)
        ) {
          return nextGroups[0].group_id
        }
        return current
      })
    } catch (error: unknown) {
      toast.error(error instanceof Error ? error.message : t('Load failed'))
    } finally {
      setIsLoading(false)
    }
  }, [t])

  const loadAssets = useCallback(async (groupId: string) => {
    if (!groupId) {
      setAssets([])
      return
    }
    setIsLoading(true)
    try {
      setAssets(await listToAPIsAvatarAssets(groupId))
    } catch (error: unknown) {
      toast.error(error instanceof Error ? error.message : t('Load failed'))
    } finally {
      setIsLoading(false)
    }
  }, [t])

  useEffect(() => {
    if (!props.open) return
    void loadGroups()
  }, [loadGroups, props.open])

  useEffect(() => {
    if (!props.open) return
    void loadAssets(selectedGroupId)
  }, [loadAssets, props.open, selectedGroupId])

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
      setGroups((prev) => [group, ...prev])
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

  const handleCreateAsset = async () => {
    const url = sourceUrl.trim()
    if (!selectedGroupId) {
      toast.error(t('Please create or select an avatar asset group first.'))
      return
    }
    if (!isHTTPURL(url)) {
      toast.error(t('Please enter a valid HTTP URL'))
      return
    }
    setIsCreatingAsset(true)
    try {
      const asset = await createToAPIsAvatarAsset({
        group_id: selectedGroupId,
        source_url: url,
        name: assetName.trim() || undefined,
      })
      setAssets((prev) => [asset, ...prev])
      setAssetName('')
      setSourceUrl('')
      toast.success(t('Created'))
    } catch (error: unknown) {
      toast.error(error instanceof Error ? error.message : t('Create failed'))
    } finally {
      setIsCreatingAsset(false)
    }
  }

  const handleRefresh = async () => {
    setIsRefreshing(true)
    try {
      setAssets(await refreshToAPIsAvatarAssets(selectedGroupId))
      toast.success(t('Refreshed'))
    } catch (error: unknown) {
      toast.error(error instanceof Error ? error.message : t('Refresh failed'))
    } finally {
      setIsRefreshing(false)
    }
  }

  const selectableAssets = assets.filter((asset) => asset.asset_type === 'image')

  let assetsContent = (
    <div className='text-muted-foreground flex min-h-28 items-center justify-center rounded-md border text-sm'>
      {t('No avatar assets')}
    </div>
  )
  if (isLoading && selectableAssets.length === 0) {
    assetsContent = (
      <div className='text-muted-foreground flex min-h-28 items-center justify-center gap-2 rounded-md border text-sm'>
        <Loader2Icon className='animate-spin' size={16} />
        {t('Loading')}
      </div>
    )
  } else if (selectableAssets.length > 0) {
    assetsContent = (
      <div className='grid max-h-80 gap-2 overflow-y-auto'>
        {selectableAssets.map((asset) => {
          const referenceValue = `asset://${asset.asset_id}`
          const isActive = asset.status === 'active'
          const isSelected = selectedSet.has(referenceValue)
          const canUse = isActive && !isSelected && !limitReached
          return (
            <div
              key={asset.asset_id}
              className='grid gap-3 rounded-md border p-2 sm:grid-cols-[64px_1fr_auto]'
            >
              <div className='bg-muted flex size-16 items-center justify-center overflow-hidden rounded-md'>
                {asset.source_url ? (
                  <img
                    src={asset.source_url}
                    alt={asset.name || asset.asset_id}
                    className='size-full object-cover'
                  />
                ) : (
                  <UserRoundIcon
                    className='text-muted-foreground'
                    size={22}
                  />
                )}
              </div>
              <div className='min-w-0 space-y-1'>
                <div className='flex items-center gap-2'>
                  <span className='truncate text-sm font-medium'>
                    {asset.name || asset.asset_id}
                  </span>
                  <Badge variant='outline'>{asset.status}</Badge>
                </div>
                <p className='text-muted-foreground truncate text-xs'>
                  {asset.asset_id}
                </p>
                <p
                  className='text-muted-foreground truncate text-xs'
                  title={asset.source_url}
                >
                  {asset.source_url}
                </p>
              </div>
              <Button
                type='button'
                variant={isSelected ? 'secondary' : 'default'}
                size='sm'
                disabled={!canUse}
                onClick={() => props.onSelect(referenceValue)}
              >
                {isSelected && <CheckIcon size={14} />}
                {isSelected ? t('Added') : t('Use')}
              </Button>
            </div>
          )
        })}
      </div>
    )
  }

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className='max-h-[calc(100vh-2rem)] overflow-hidden sm:max-w-2xl'>
        <DialogHeader>
          <DialogTitle>{t('Select avatar asset')}</DialogTitle>
          <DialogDescription>
            {t(
              'Use active ToAPIs private avatar image assets as reference images.'
            )}
          </DialogDescription>
        </DialogHeader>

        <div className='space-y-4 overflow-y-auto pr-1'>
          <div className='space-y-2 rounded-md border p-3'>
            <div className='flex items-center justify-between gap-2'>
              <Label>{t('Avatar asset group')}</Label>
              <Button
                type='button'
                variant='outline'
                size='sm'
                disabled={!selectedGroupId || isRefreshing || isLoading}
                onClick={handleRefresh}
              >
                {isRefreshing ? (
                  <Loader2Icon className='animate-spin' size={14} />
                ) : (
                  <RefreshCwIcon size={14} />
                )}
                {t('Refresh')}
              </Button>
            </div>
            <Select
              value={selectedGroupId}
              onValueChange={(value) => setSelectedGroupId(value ?? '')}
              disabled={groups.length === 0 || isLoading}
            >
              <SelectTrigger className='w-full'>
                <SelectValue placeholder={t('Select group')} />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {groups.map((group) => (
                    <SelectItem key={group.group_id} value={group.group_id}>
                      {group.name || group.group_id}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
            <div className='grid gap-2 sm:grid-cols-[1fr_1fr_auto]'>
              <Input
                value={groupName}
                onChange={(event) => setGroupName(event.target.value)}
                placeholder={t('Group name')}
                disabled={isCreatingGroup}
              />
              <Input
                value={groupDescription}
                onChange={(event) => setGroupDescription(event.target.value)}
                placeholder={t('Group description (optional)')}
                disabled={isCreatingGroup}
              />
              <Button
                type='button'
                disabled={isCreatingGroup || !groupName.trim()}
                onClick={handleCreateGroup}
              >
                {isCreatingGroup ? (
                  <Loader2Icon className='animate-spin' size={14} />
                ) : (
                  <PlusIcon size={14} />
                )}
                {t('New group')}
              </Button>
            </div>
          </div>

          <div className='space-y-2 rounded-md border p-3'>
            <Label>{t('New avatar asset')}</Label>
            <div className='grid gap-2 sm:grid-cols-[0.8fr_1fr_auto]'>
              <Input
                value={assetName}
                onChange={(event) => setAssetName(event.target.value)}
                placeholder={t('Asset name (optional)')}
                disabled={isCreatingAsset}
              />
              <Input
                value={sourceUrl}
                onChange={(event) => setSourceUrl(event.target.value)}
                placeholder={t('Public image URL')}
                disabled={isCreatingAsset}
              />
              <Button
                type='button'
                disabled={
                  isCreatingAsset || !selectedGroupId || !sourceUrl.trim()
                }
                onClick={handleCreateAsset}
              >
                {isCreatingAsset ? (
                  <Loader2Icon className='animate-spin' size={14} />
                ) : (
                  <PlusIcon size={14} />
                )}
                {t('Add asset')}
              </Button>
            </div>
          </div>

          <div className='space-y-2'>
            <div className='flex items-center justify-between gap-2'>
              <Label>{t('Avatar assets')}</Label>
              {limitReached && (
                <span className='text-muted-foreground text-xs'>
                  {t('Reference image limit reached')}
                </span>
              )}
            </div>
            {assetsContent}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

function isHTTPURL(value: string): boolean {
  try {
    const url = new URL(value)
    return url.protocol === 'http:' || url.protocol === 'https:'
  } catch {
    return false
  }
}
