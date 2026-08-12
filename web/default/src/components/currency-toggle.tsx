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
import { DollarSign } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { useBillingCurrencyConfig, useCurrencyOverride } from '@/lib/currency'
import { cn } from '@/lib/utils'

/**
 * Pill toggle that switches model-price display between USD and CNY.
 *
 * The selection persists as `currencyOverride` and is honored only by billing
 * price formatters (`formatBillingCurrencyFromUSD`, `formatPrice`, dynamic
 * pricing). Wallet balances and payment amounts stay on the admin display type.
 *
 * Hidden when the admin display mode is TOKENS or CUSTOM.
 */
export function CurrencyToggle({ className }: { className?: string }) {
  const { t } = useTranslation()
  const { setOverride } = useCurrencyOverride()
  const currency = useBillingCurrencyConfig()

  if (
    currency.quotaDisplayType === 'TOKENS' ||
    currency.quotaDisplayType === 'CUSTOM'
  ) {
    return null
  }

  const isCNY = currency.quotaDisplayType === 'CNY'

  const handleToggle = () => {
    setOverride(isCNY ? 'USD' : 'CNY')
  }

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <button
            type='button'
            onClick={handleToggle}
            aria-label={t('Toggle billing currency')}
            aria-pressed={isCNY}
            className={cn(
              'bg-card hover:bg-muted/70 inline-flex h-9 items-center gap-1.5 rounded-full border px-3 font-mono text-[13px] font-bold transition-colors',
              className
            )}
          />
        }
      >
        <DollarSign className='size-3.5 shrink-0 opacity-70' />
        <span
          className={cn(
            !isCNY && 'text-foreground',
            isCNY && 'text-muted-foreground'
          )}
        >
          $
        </span>
        <span aria-hidden='true' className='text-muted-foreground/50'>
          /
        </span>
        <span
          className={cn(
            isCNY && 'text-foreground',
            !isCNY && 'text-muted-foreground'
          )}
        >
          ¥
        </span>
      </TooltipTrigger>
      <TooltipContent side='bottom'>
        {isCNY
          ? t('Show prices in USD')
          : t('Show prices in CNY at the configured exchange rate')}
      </TooltipContent>
    </Tooltip>
  )
}
