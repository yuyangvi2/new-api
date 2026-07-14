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
import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { CheckCircle2, Loader2 } from 'lucide-react'
import { QRCodeSVG } from 'qrcode.react'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import z from 'zod'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { getPaymentStatus } from '@/features/wallet/api'

const wechatPaySearchSchema = z.object({
  code_url: z.string().catch(''),
  trade_no: z.string().catch(''),
})

export const Route = createFileRoute('/console/wechat-pay')({
  validateSearch: wechatPaySearchSchema,
  component: WeChatPayPage,
})

function WeChatPayPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const search = Route.useSearch()
  const [status, setStatus] = useState('')
  const [checking, setChecking] = useState(false)
  const paid = status === 'success'

  const decodedCodeURL = useMemo(() => {
    try {
      return decodeURIComponent(search.code_url)
    } catch {
      return search.code_url
    }
  }, [search.code_url])

  useEffect(() => {
    if (!search.trade_no || paid) return

    let cancelled = false
    const timer = window.setInterval(async () => {
      setChecking(true)
      try {
        const res = await getPaymentStatus(search.trade_no)
        if (!cancelled && res.success && res.data?.status) {
          setStatus(res.data.status)
        }
      } finally {
        if (!cancelled) setChecking(false)
      }
    }, 3000)

    return () => {
      cancelled = true
      window.clearInterval(timer)
    }
  }, [paid, search.trade_no])

  let paymentContent = (
    <p className='text-muted-foreground text-sm'>
      {t('Payment QR code is unavailable. Please try again.')}
    </p>
  )
  if (paid) {
    paymentContent = (
      <div className='space-y-4'>
        <CheckCircle2 className='mx-auto h-12 w-12 text-green-600' />
        <p className='text-sm font-medium'>{t('Payment completed')}</p>
      </div>
    )
  } else if (decodedCodeURL) {
    paymentContent = (
      <div className='space-y-4'>
        <div className='inline-flex rounded-lg border bg-white p-4'>
          <QRCodeSVG value={decodedCodeURL} size={220} />
        </div>
        <p className='text-muted-foreground text-sm'>
          {t('Scan with WeChat to complete the payment.')}
        </p>
        <div className='text-muted-foreground flex items-center justify-center gap-2 text-xs'>
          {checking && <Loader2 className='h-3 w-3 animate-spin' />}
          {t('Waiting for payment confirmation...')}
        </div>
      </div>
    )
  }

  return (
    <main className='bg-background flex min-h-screen items-center justify-center p-4'>
      <Card className='w-full max-w-md'>
        <CardHeader className='text-center'>
          <CardTitle>{t('WeChat Pay')}</CardTitle>
        </CardHeader>
        <CardContent className='space-y-6 text-center'>
          {paymentContent}

          <Button
            type='button'
            className='w-full'
            onClick={() =>
              navigate({
                to: '/wallet',
                search: { show_history: true },
              })
            }
          >
            {paid ? t('Back to wallet') : t('View order history')}
          </Button>
        </CardContent>
      </Card>
    </main>
  )
}
