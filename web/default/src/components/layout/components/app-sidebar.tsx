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
import { useEffect } from 'react'
import { AnimatePresence, motion, useReducedMotion } from 'motion/react'
import { MOTION_TRANSITION, MOTION_VARIANTS } from '@/lib/motion'
import { useLayout } from '@/context/layout-provider'
import { useSetSidebarPortalTarget } from '@/context/sidebar-portal'
import { useSidebarView } from '@/hooks/use-sidebar-view'
import { Sidebar, SidebarContent, SidebarRail } from '@/components/ui/sidebar'
import { NavGroup } from './nav-group'
import { SidebarViewHeader } from './sidebar-view-header'

/** Width of the icon-rail sidebar used by portal views (Experience Center). */
const PORTAL_SIDEBAR_WIDTH = '80px'
/** Default sidebar width — must match the constant in sidebar.tsx. */
const DEFAULT_SIDEBAR_WIDTH = '13rem'

/**
 * Application sidebar.
 *
 * Adopts the Vercel / Cloudflare "drill-in" pattern: the URL drives
 * which sidebar *view* is rendered. Clicking a top-level entry like
 * `System Settings` swaps the sidebar to a contextual workspace —
 * with a `← Back to Dashboard` affordance — instead of stacking the
 * sub-navigation inside the root tree.
 *
 * Portal views (e.g. Experience Center) shrink the sidebar to an 80 px
 * icon rail and let the page component inject its own content.
 */
export function AppSidebar() {
  const { collapsible, variant } = useLayout()
  const { key, view, navGroups } = useSidebarView()
  const shouldReduce = useReducedMotion()
  const setPortalTarget = useSetSidebarPortalTarget()

  // Override --sidebar-width on the provider wrapper so both the gap
  // and the fixed container respect the narrower rail width.
  // Clean up on unmount or when leaving the portal view.
  const isPortal = !!view?.usePortal
  useEffect(() => {
    if (!isPortal) return
    const wrapper = document.querySelector<HTMLElement>(
      '[data-slot="sidebar-wrapper"]'
    )
    if (!wrapper) return
    wrapper.style.setProperty('--sidebar-width', PORTAL_SIDEBAR_WIDTH)
    return () => {
      wrapper.style.setProperty('--sidebar-width', DEFAULT_SIDEBAR_WIDTH)
    }
  }, [isPortal])

  return (
    <Sidebar collapsible={collapsible} variant={variant}>
      {/* Portal views manage their own back button inside the rail */}
      {view && !view.usePortal && <SidebarViewHeader view={view} />}

      {view?.usePortal ? (
        /* Portal mode: narrow icon rail — page injects via createPortal */
        <SidebarContent className='p-0'>
          <div ref={setPortalTarget} className='flex h-full flex-col' />
        </SidebarContent>
      ) : (
        <SidebarContent className='py-2'>
          <AnimatePresence mode='wait' initial={false}>
            <motion.div
              key={key}
              initial={
                shouldReduce ? false : MOTION_VARIANTS.sidebarSlide.initial
              }
              animate={MOTION_VARIANTS.sidebarSlide.animate}
              exit={
                shouldReduce ? undefined : MOTION_VARIANTS.sidebarSlide.exit
              }
              transition={MOTION_TRANSITION.fast}
              className='flex flex-col'
            >
              {navGroups.map((props) => (
                <NavGroup key={props.id || props.title} {...props} />
              ))}
            </motion.div>
          </AnimatePresence>
        </SidebarContent>
      )}

      <SidebarRail />
    </Sidebar>
  )
}
