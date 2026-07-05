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
import { createElement, type SVGProps } from 'react'
import { HugeiconsIcon, type IconSvgElement } from '@hugeicons/react'
import {
  AiChat02Icon,
  AiContentGenerator02Icon,
  AiSecurity02Icon,
  ApiGatewayIcon,
  BrainCogIcon,
  CheckListIcon,
  ComputerSettingsIcon,
  ComputerTerminal02Icon,
  CouponPercentIcon,
  CreditCardAcceptIcon,
  DashboardSpeed02Icon,
  DashboardSquare03Icon,
  FileChartLineIcon,
  Key02Icon,
  LayoutGridIcon,
  LimitationIcon,
  PaintBoardIcon,
  Payment02Icon,
  Route03Icon,
  ServerStack03Icon,
  ToolCaseIcon,
  UserAccountIcon,
  UserGroup03Icon,
  UserShield01Icon,
  Wallet03Icon,
} from '@hugeicons/core-free-icons'

type SidebarIconProps = Omit<SVGProps<SVGSVGElement>, 'strokeWidth'>

function createSidebarIcon(icon: IconSvgElement) {
  return function SidebarIcon(props: SidebarIconProps) {
    return createElement(HugeiconsIcon, {
      icon,
      strokeWidth: 1.8,
      absoluteStrokeWidth: true,
      ...props,
    })
  }
}

export const SidebarIcons = {
  apiKeys: createSidebarIcon(Key02Icon as IconSvgElement),
  auth: createSidebarIcon(UserShield01Icon as IconSvgElement),
  billing: createSidebarIcon(Payment02Icon as IconSvgElement),
  channels: createSidebarIcon(ApiGatewayIcon as IconSvgElement),
  chat: createSidebarIcon(AiChat02Icon as IconSvgElement),
  consoleContent: createSidebarIcon(LayoutGridIcon as IconSvgElement),
  dashboard: createSidebarIcon(DashboardSquare03Icon as IconSvgElement),
  experienceCenter: createSidebarIcon(
    AiContentGenerator02Icon as IconSvgElement
  ),
  models: createSidebarIcon(BrainCogIcon as IconSvgElement),
  modelsRouting: createSidebarIcon(Route03Icon as IconSvgElement),
  operations: createSidebarIcon(ToolCaseIcon as IconSvgElement),
  overview: createSidebarIcon(DashboardSpeed02Icon as IconSvgElement),
  playground: createSidebarIcon(ComputerTerminal02Icon as IconSvgElement),
  profile: createSidebarIcon(UserAccountIcon as IconSvgElement),
  redemptionCodes: createSidebarIcon(CouponPercentIcon as IconSvgElement),
  security: createSidebarIcon(AiSecurity02Icon as IconSvgElement),
  securityLimits: createSidebarIcon(LimitationIcon as IconSvgElement),
  siteBranding: createSidebarIcon(PaintBoardIcon as IconSvgElement),
  subscription: createSidebarIcon(CreditCardAcceptIcon as IconSvgElement),
  systemInfo: createSidebarIcon(ServerStack03Icon as IconSvgElement),
  taskLogs: createSidebarIcon(CheckListIcon as IconSvgElement),
  usageLogs: createSidebarIcon(FileChartLineIcon as IconSvgElement),
  users: createSidebarIcon(UserGroup03Icon as IconSvgElement),
  wallet: createSidebarIcon(Wallet03Icon as IconSvgElement),
  systemSettings: createSidebarIcon(ComputerSettingsIcon as IconSvgElement),
} as const
