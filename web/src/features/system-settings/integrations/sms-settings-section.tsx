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
import { zodResolver } from '@hookform/resolvers/zod'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import * as z from 'zod'

import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { api } from '@/lib/api'

import {
  SettingsForm,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useResetForm } from '../hooks/use-reset-form'
import { useUpdateOption } from '../hooks/use-update-option'

const smsSchema = z.object({
  SMSProviderType: z.string(),
  SMSClientId: z.string(),
  SMSClientSecret: z.string(),
  SMSSignName: z.string(),
  SMSTemplateCode: z.string(),
})

type SmsFormValues = z.infer<typeof smsSchema>

type SmsSettingsSectionProps = {
  defaultValues: SmsFormValues
}

export function SmsSettingsSection(props: SmsSettingsSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const [testPhone, setTestPhone] = useState('')
  const [isTesting, setIsTesting] = useState(false)

  const form = useForm<SmsFormValues>({
    resolver: zodResolver(smsSchema),
    defaultValues: props.defaultValues,
  })

  useResetForm(form, props.defaultValues)

  const onSubmit = async (values: SmsFormValues) => {
    const updates: { key: string; value: string | boolean }[] = []
    const initial = props.defaultValues
    if (values.SMSProviderType !== initial.SMSProviderType) {
      updates.push({ key: 'SMSProviderType', value: values.SMSProviderType })
    }
    if (values.SMSClientId !== initial.SMSClientId) {
      updates.push({ key: 'SMSClientId', value: values.SMSClientId.trim() })
    }
    if (values.SMSClientSecret.trim() !== '') {
      updates.push({
        key: 'SMSClientSecret',
        value: values.SMSClientSecret.trim(),
      })
    }
    if (values.SMSSignName !== initial.SMSSignName) {
      updates.push({ key: 'SMSSignName', value: values.SMSSignName.trim() })
    }
    if (values.SMSTemplateCode !== initial.SMSTemplateCode) {
      updates.push({
        key: 'SMSTemplateCode',
        value: values.SMSTemplateCode.trim(),
      })
    }
    for (const update of updates) {
      await updateOption.mutateAsync(update)
    }
  }

  const handleTestSMS = async () => {
    if (!testPhone.trim()) {
      toast.error(t('Please enter a phone number'))
      return
    }
    setIsTesting(true)
    try {
      const res = await api.post('/api/option/sms/test', {
        phone: testPhone.trim(),
      })
      if (res.data?.success) {
        toast.success(t('Test SMS sent successfully'))
      } else {
        toast.error(res.data?.message || t('Failed to send test SMS'))
      }
    } catch {
      // handled by interceptor
    } finally {
      setIsTesting(false)
    }
  }

  return (
    <SettingsSection title={t('SMS Service')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)} autoComplete='off'>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
            saveLabel='Save SMS settings'
          />
          <FormField
            control={form.control}
            name='SMSProviderType'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('SMS Provider')}</FormLabel>
                <Select value={field.value} onValueChange={field.onChange}>
                  <FormControl>
                    <SelectTrigger>
                      <SelectValue placeholder={t('Select SMS provider')} />
                    </SelectTrigger>
                  </FormControl>
                  <SelectContent>
                    <SelectItem value='alibaba'>
                      {t('Alibaba Cloud SMS')}
                    </SelectItem>
                  </SelectContent>
                </Select>
                <FormDescription>
                  {t('Currently only Alibaba Cloud SMS is supported')}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name='SMSClientId'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Client ID')}</FormLabel>
                <FormControl>
                  <Input
                    autoComplete='off'
                    placeholder='AccessKeyId'
                    {...field}
                  />
                </FormControl>
                <FormDescription>
                  {t('SMS provider access key ID')}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name='SMSClientSecret'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Client Secret')}</FormLabel>
                <FormControl>
                  <Input
                    type='password'
                    autoComplete='new-password'
                    placeholder={t('Leave blank to keep unchanged')}
                    {...field}
                  />
                </FormControl>
                <FormDescription>
                  {t('SMS provider access key secret')}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name='SMSSignName'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Sign Name')}</FormLabel>
                <FormControl>
                  <Input autoComplete='off' {...field} />
                </FormControl>
                <FormDescription>
                  {t('SMS signature name approved by the provider')}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name='SMSTemplateCode'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('SMS Template')}</FormLabel>
                <FormControl>
                  <Input
                    autoComplete='off'
                    placeholder='SMS_xxxxx'
                    {...field}
                  />
                </FormControl>
                <FormDescription>
                  {t(
                    'Verification code template; template variable should be code'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          <div className='flex flex-col gap-3 border-t pt-4 sm:flex-row sm:items-end'>
            <div className='flex-1 space-y-2'>
              <FormLabel>{t('Test phone number')}</FormLabel>
              <Input
                value={testPhone}
                onChange={(e) => setTestPhone(e.target.value)}
                placeholder='13800138000'
                autoComplete='off'
              />
            </div>
            <Button
              type='button'
              variant='outline'
              disabled={isTesting}
              onClick={() => void handleTestSMS()}
            >
              {isTesting ? t('Sending...') : t('Send test SMS')}
            </Button>
          </div>
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
