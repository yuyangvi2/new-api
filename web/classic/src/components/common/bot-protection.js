/*
Copyright (C) 2025 QuantumNous

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

const disabledConfig = {
  enabled: false,
  provider: 'disabled',
  public_endpoint: '',
  site_key: '',
};

export function resolveBotProtectionConfig(status, scene) {
  const configured = status?.bot_protection?.[scene];
  if (configured) {
    return configured;
  }

  if (status?.turnstile_check && status.turnstile_site_key) {
    return {
      enabled: true,
      provider: 'turnstile',
      public_endpoint: '',
      site_key: status.turnstile_site_key,
    };
  }

  return disabledConfig;
}

export function buildCapEndpoint(config) {
  const publicEndpoint = config.public_endpoint.replace(/\/+$/, '');
  return `${publicEndpoint}/${encodeURIComponent(config.site_key)}/`;
}
