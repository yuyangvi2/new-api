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
import { createContext, useContext, useState, type ReactNode } from 'react'

/**
 * Sidebar portal system.
 *
 * Allows page-level components to render custom content into the
 * sidebar via `createPortal`. Used by the Experience Center to inject
 * its config panel (mode tabs + generation settings) into the sidebar
 * area, while keeping all state in the page component.
 *
 * Architecture:
 *   - `SidebarPortalProvider` sits above both AppSidebar and page content
 *   - AppSidebar calls `setPortalTarget(ref)` to register the target div
 *   - Page component reads the target via `useSidebarPortalTarget()`
 *     and renders into it using `createPortal`
 */

type SidebarPortalContextValue = {
  /** The DOM element that portal content should render into */
  target: HTMLDivElement | null
  /** Called by AppSidebar to register the portal target element */
  setTarget: (el: HTMLDivElement | null) => void
}

const SidebarPortalContext = createContext<SidebarPortalContextValue>({
  target: null,
  setTarget: () => {},
})

export function SidebarPortalProvider({ children }: { children: ReactNode }) {
  const [target, setTarget] = useState<HTMLDivElement | null>(null)
  return (
    <SidebarPortalContext.Provider value={{ target, setTarget }}>
      {children}
    </SidebarPortalContext.Provider>
  )
}

/** Get the sidebar portal target element (used by page components) */
export function useSidebarPortalTarget() {
  return useContext(SidebarPortalContext).target
}

/** Get the setter for the sidebar portal target (used by AppSidebar) */
export function useSetSidebarPortalTarget() {
  return useContext(SidebarPortalContext).setTarget
}
