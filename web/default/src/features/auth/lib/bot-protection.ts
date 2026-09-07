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
  BotProtectionConfig,
  BotProtectionScene,
  SystemStatus,
} from '../types'

const disabledConfig: BotProtectionConfig = {
  enabled: false,
  provider: 'disabled',
  public_endpoint: '',
  site_key: '',
}

export function resolveBotProtectionConfig(
  status: SystemStatus | null | undefined,
  scene: BotProtectionScene
): BotProtectionConfig {
  const statusData = status?.data ?? status
  const configured = statusData?.bot_protection?.[scene]
  if (configured) {
    return configured
  }

  if (statusData?.turnstile_check && statusData.turnstile_site_key) {
    return {
      enabled: true,
      provider: 'turnstile',
      public_endpoint: '',
      site_key: statusData.turnstile_site_key,
    }
  }

  return disabledConfig
}

export function buildCapEndpoint(config: BotProtectionConfig): string {
  const base = config.public_endpoint.replace(/\/+$/, '')
  return `${base}/${encodeURIComponent(config.site_key)}/`
}
