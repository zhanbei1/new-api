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
import { QRCodeSVG } from 'qrcode.react'
import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { api } from '@/lib/api'

export type PaymentQrDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  tradeNo: string
  qrCode: string
  title?: string
  statusEndpoint: string
  onPaid?: () => void
}

export function PaymentQrDialog({
  open,
  onOpenChange,
  tradeNo,
  qrCode,
  title,
  statusEndpoint,
  onPaid,
}: PaymentQrDialogProps) {
  const { t } = useTranslation()
  const [status, setStatus] = useState('pending')
  const paidRef = useRef(false)
  const onPaidRef = useRef(onPaid)
  const onOpenChangeRef = useRef(onOpenChange)

  useEffect(() => {
    onPaidRef.current = onPaid
  }, [onPaid])

  useEffect(() => {
    onOpenChangeRef.current = onOpenChange
  }, [onOpenChange])

  useEffect(() => {
    if (!open || !tradeNo) return
    paidRef.current = false
    setStatus('pending')
    const timer = window.setInterval(() => {
      void (async () => {
        try {
          const res = await api.get(statusEndpoint, {
            params: { trade_no: tradeNo },
          })
          const nextStatus = res.data?.data?.status as string | undefined
          if (!nextStatus) return
          setStatus(nextStatus)
          if (nextStatus === 'success' && !paidRef.current) {
            paidRef.current = true
            toast.success(t('Payment successful'))
            onPaidRef.current?.()
            onOpenChangeRef.current(false)
          }
        } catch {
          // ignore polling errors
        }
      })()
    }, 2500)
    return () => window.clearInterval(timer)
  }, [open, tradeNo, statusEndpoint, t])

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={title || t('Scan to pay with Alipay')}
      description={t('Please complete the payment in the Alipay app')}
      contentClassName='max-w-sm'
      contentHeight='auto'
      footer={
        <Button
          type='button'
          variant='outline'
          onClick={() => onOpenChange(false)}
        >
          {t('Close')}
        </Button>
      }
    >
      <div className='flex flex-col items-center gap-4 py-2'>
        {qrCode ? <QRCodeSVG value={qrCode} size={200} /> : null}
        <p className='text-muted-foreground text-sm'>
          {t('Order')}: {tradeNo}
        </p>
        <p className='text-sm'>
          {t('Status')}:{' '}
          {status === 'success' ? t('Paid') : t('Waiting for payment')}
        </p>
      </div>
    </Dialog>
  )
}
