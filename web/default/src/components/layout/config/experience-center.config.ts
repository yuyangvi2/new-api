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
import type { SidebarView } from '../types'

/**
 * Experience Center sidebar view.
 *
 * Unlike System Settings (which uses nav groups), the Experience Center
 * uses `usePortal: true` so that the page component can inject its own
 * interactive config panel (mode tabs + generation settings) into the
 * sidebar via React Portal.
 */
export const EXPERIENCE_CENTER_VIEW: SidebarView = {
  id: 'experience-center',
  pathPattern: /^\/image-generator(\/|$)/,
  parent: {
    to: '/dashboard/overview',
    label: 'Back to Dashboard',
  },
  usePortal: true,
  getNavGroups: () => [],
}
