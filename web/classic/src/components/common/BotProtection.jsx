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

import React, { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import Turnstile from 'react-turnstile';

import { buildCapEndpoint } from './bot-protection';

const BotProtection = ({ config, onVerify, onExpire, className }) => {
  if (!config.enabled) return null;

  if (config.provider === 'turnstile') {
    return (
      <Turnstile
        sitekey={config.site_key}
        onVerify={onVerify}
        onExpire={onExpire}
        className={className}
      />
    );
  }

  if (config.provider === 'cap') {
    return (
      <CapProtection
        config={config}
        onVerify={onVerify}
        onExpire={onExpire}
        className={className}
      />
    );
  }

  return null;
};

const CapProtection = ({ config, onVerify, onExpire, className }) => {
  const { t } = useTranslation();
  const containerRef = useRef(null);
  const onVerifyRef = useRef(onVerify);
  const onExpireRef = useRef(onExpire);
  const [loadFailed, setLoadFailed] = useState(false);
  const publicEndpoint = config.public_endpoint.replace(/\/+$/, '');
  const capEndpoint = buildCapEndpoint(config);

  useEffect(() => {
    onVerifyRef.current = onVerify;
    onExpireRef.current = onExpire;
  }, [onExpire, onVerify]);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return undefined;

    window.CAP_CUSTOM_WASM_URL = `${publicEndpoint}/assets/cap_wasm_bg.wasm`;
    let element = null;
    let disposed = false;

    const handleSolve = (event) => {
      const token = event.detail?.token;
      if (token) onVerifyRef.current(token);
    };
    const handleReset = () => onExpireRef.current?.();

    setLoadFailed(false);
    void import('cap-widget')
      .then(() => {
        if (disposed) return;
        element = document.createElement('cap-widget');
        element.setAttribute('data-cap-api-endpoint', capEndpoint);
        element.setAttribute('required', '');
        element.addEventListener('solve', handleSolve);
        element.addEventListener('reset', handleReset);
        element.addEventListener('error', handleReset);
        container.replaceChildren(element);
      })
      .catch(() => {
        if (!disposed) setLoadFailed(true);
      });

    return () => {
      disposed = true;
      if (element) {
        element.removeEventListener('solve', handleSolve);
        element.removeEventListener('reset', handleReset);
        element.removeEventListener('error', handleReset);
        element.remove();
      }
    };
  }, [capEndpoint, publicEndpoint]);

  if (loadFailed) {
    return (
      <div className={className} role='alert'>
        {t('页面渲染出错，请刷新页面重试')}
      </div>
    );
  }

  return <div ref={containerRef} className={className} />;
};

export default BotProtection;
