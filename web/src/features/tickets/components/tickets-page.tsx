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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { SectionPageLayout } from '@/components/layout'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Textarea } from '@/components/ui/textarea'
import { formatDateTimeObject } from '@/lib/time'
import { cn } from '@/lib/utils'

import {
  closeTicket,
  createTicket,
  deleteTicket,
  getTicket,
  listTickets,
  replyTicket,
} from '../api'
import { TICKET_STATUS_LABEL, TICKET_STATUSES } from '../constants'
import type { Ticket, TicketMode, TicketReply, TicketStatus } from '../types'

function formatTs(ts: number) {
  if (!ts) return '-'
  return formatDateTimeObject(new Date(ts * 1000))
}

function StatusBadge({ status }: { status: TicketStatus }) {
  const { t } = useTranslation()
  let variant: 'default' | 'secondary' | 'outline' = 'outline'
  if (status === 'open') {
    variant = 'default'
  } else if (status === 'replied') {
    variant = 'secondary'
  }
  return <Badge variant={variant}>{t(TICKET_STATUS_LABEL[status])}</Badge>
}

function findParentSnippet(
  replies: TicketReply[],
  parentId?: number | null
): string | null {
  if (!parentId) return null
  const parent = replies.find((item) => item.id === parentId)
  if (!parent) return `#${parentId}`
  const text = parent.content.trim()
  return text.length > 80 ? `${text.slice(0, 80)}…` : text
}

export function TicketsPage({ mode }: { mode: TicketMode }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [page, setPage] = useState(1)
  const [status, setStatus] = useState<string>('all')
  const [keyword, setKeyword] = useState('')
  const [selectedId, setSelectedId] = useState<number | null>(null)
  const [createOpen, setCreateOpen] = useState(false)
  const [deleteId, setDeleteId] = useState<number | null>(null)
  const [title, setTitle] = useState('')
  const [content, setContent] = useState('')
  const [replyContent, setReplyContent] = useState('')
  const [replyParentId, setReplyParentId] = useState<number | null>(null)

  const pageSize = 20
  const queryKey = ['tickets', mode, page, status, keyword]

  const { data, isLoading } = useQuery({
    queryKey,
    queryFn: async () => {
      const res = await listTickets(mode, {
        p: page,
        page_size: pageSize,
        status: status === 'all' ? undefined : status,
        keyword: mode === 'admin' ? keyword.trim() || undefined : undefined,
      })
      if (!res.success) {
        throw new Error(res.message || t('Failed to load tickets'))
      }
      return res.data
    },
  })

  const detailQuery = useQuery({
    queryKey: ['ticket-detail', mode, selectedId],
    enabled: selectedId != null,
    queryFn: async () => {
      if (selectedId == null) {
        throw new Error(t('Failed to load ticket'))
      }
      const res = await getTicket(mode, selectedId)
      if (!res.success || !res.data) {
        throw new Error(res.message || t('Failed to load ticket'))
      }
      return res.data
    },
  })

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ['tickets', mode] })
    if (selectedId != null) {
      void queryClient.invalidateQueries({
        queryKey: ['ticket-detail', mode, selectedId],
      })
    }
  }

  const createMutation = useMutation({
    mutationFn: () => createTicket({ title, content }),
    onSuccess: (res) => {
      if (!res.success) {
        toast.error(res.message || t('Failed to create ticket'))
        return
      }
      toast.success(t('Ticket created'))
      setCreateOpen(false)
      setTitle('')
      setContent('')
      invalidate()
    },
    onError: (err: Error) => toast.error(err.message),
  })

  const replyMutation = useMutation({
    mutationFn: () => {
      if (selectedId == null) {
        return Promise.reject(new Error(t('Failed to send reply')))
      }
      return replyTicket(mode, selectedId, {
        content: replyContent,
        parent_id: replyParentId,
      })
    },
    onSuccess: (res) => {
      if (!res.success) {
        toast.error(res.message || t('Failed to send reply'))
        return
      }
      toast.success(t('Reply sent'))
      setReplyContent('')
      setReplyParentId(null)
      invalidate()
    },
    onError: (err: Error) => toast.error(err.message),
  })

  const closeMutation = useMutation({
    mutationFn: (id: number) => closeTicket(mode, id),
    onSuccess: (res) => {
      if (!res.success) {
        toast.error(res.message || t('Failed to close ticket'))
        return
      }
      toast.success(t('Ticket closed'))
      invalidate()
    },
    onError: (err: Error) => toast.error(err.message),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => deleteTicket(mode, id),
    onSuccess: (res, id) => {
      if (!res.success) {
        toast.error(res.message || t('Failed to delete ticket'))
        return
      }
      toast.success(t('Ticket deleted'))
      setDeleteId(null)
      setSelectedId((current) => (current === id ? null : current))
      invalidate()
    },
    onError: (err: Error) => toast.error(err.message),
  })

  const items = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))
  const detail = detailQuery.data
  const parentSnippet = detail
    ? findParentSnippet(detail.replies ?? [], replyParentId)
    : null

  return (
    <>
    <SectionPageLayout fixedContent>
      <SectionPageLayout.Title>
        {mode === 'admin' ? t('Ticket Management') : t('Tickets')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        {mode === 'user' ? (
          <Button onClick={() => setCreateOpen(true)}>
            {t('New Ticket')}
          </Button>
        ) : null}
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='mb-4 flex flex-wrap items-center gap-2'>
          <Select
            value={status}
            onValueChange={(v) => {
              if (v !== null) {
                setStatus(v)
                setPage(1)
              }
            }}
          >
            <SelectTrigger className='w-[160px]'>
              <SelectValue placeholder={t('Status')} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value='all'>{t('All')}</SelectItem>
              {TICKET_STATUSES.map((item) => (
                <SelectItem key={item} value={item}>
                  {t(TICKET_STATUS_LABEL[item])}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          {mode === 'admin' ? (
            <Input
              className='max-w-xs'
              placeholder={t('Search title or content')}
              value={keyword}
              onChange={(e) => {
                setKeyword(e.target.value)
                setPage(1)
              }}
            />
          ) : null}
        </div>

        <div className='flex min-h-0 flex-1 flex-col overflow-hidden rounded-lg border'>
          <div className='min-h-0 flex-1 overflow-auto'>
          <table className='w-full text-sm'>
            <thead className='bg-muted/50 text-left'>
              <tr>
                <th className='px-3 py-2 font-medium'>ID</th>
                {mode === 'admin' ? (
                  <th className='px-3 py-2 font-medium'>{t('User')}</th>
                ) : null}
                <th className='px-3 py-2 font-medium'>{t('Title')}</th>
                <th className='px-3 py-2 font-medium'>{t('Status')}</th>
                <th className='px-3 py-2 font-medium'>{t('Updated')}</th>
                <th className='px-3 py-2 font-medium'>{t('Actions')}</th>
              </tr>
            </thead>
            <tbody>
              {(() => {
                if (isLoading) {
                  return (
                    <tr>
                      <td
                        className='text-muted-foreground px-3 py-6 text-center'
                        colSpan={mode === 'admin' ? 6 : 5}
                      >
                        {t('Loading...')}
                      </td>
                    </tr>
                  )
                }
                if (items.length === 0) {
                  return (
                    <tr>
                      <td
                        className='text-muted-foreground px-3 py-6 text-center'
                        colSpan={mode === 'admin' ? 6 : 5}
                      >
                        {t('No tickets')}
                      </td>
                    </tr>
                  )
                }
                return items.map((ticket: Ticket) => (
                  <tr key={ticket.id} className='border-t'>
                    <td className='px-3 py-2'>{ticket.id}</td>
                    {mode === 'admin' ? (
                      <td className='px-3 py-2'>{ticket.user_id}</td>
                    ) : null}
                    <td className='max-w-[280px] truncate px-3 py-2'>
                      {ticket.title}
                    </td>
                    <td className='px-3 py-2'>
                      <StatusBadge status={ticket.status} />
                    </td>
                    <td className='text-muted-foreground px-3 py-2 whitespace-nowrap'>
                      {formatTs(ticket.updated_at)}
                    </td>
                    <td className='px-3 py-2'>
                      <div className='flex flex-wrap gap-1'>
                        <Button
                          size='sm'
                          variant='outline'
                          onClick={() => setSelectedId(ticket.id)}
                        >
                          {t('View')}
                        </Button>
                        {ticket.status !== 'closed' ? (
                          <Button
                            size='sm'
                            variant='outline'
                            onClick={() => closeMutation.mutate(ticket.id)}
                          >
                            {t('Close')}
                          </Button>
                        ) : null}
                        <Button
                          size='sm'
                          variant='destructive'
                          onClick={() => setDeleteId(ticket.id)}
                        >
                          {t('Delete')}
                        </Button>
                      </div>
                    </td>
                  </tr>
                ))
              })()}
            </tbody>
          </table>
          </div>

        <div className='flex items-center justify-end gap-2 border-t px-3 py-2'>
          <Button
            size='sm'
            variant='outline'
            disabled={page <= 1}
            onClick={() => setPage((p) => Math.max(1, p - 1))}
          >
            {t('Previous')}
          </Button>
          <span className='text-muted-foreground text-sm'>
            {page} / {totalPages}
          </span>
          <Button
            size='sm'
            variant='outline'
            disabled={page >= totalPages}
            onClick={() => setPage((p) => p + 1)}
          >
            {t('Next')}
          </Button>
        </div>
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>

      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('New Ticket')}</DialogTitle>
          </DialogHeader>
          <div className='space-y-3'>
            <div className='space-y-1'>
              <Label>{t('Title')}</Label>
              <Input
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                maxLength={200}
              />
            </div>
            <div className='space-y-1'>
              <Label>{t('Content')}</Label>
              <Textarea
                value={content}
                onChange={(e) => setContent(e.target.value)}
                rows={6}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant='outline' onClick={() => setCreateOpen(false)}>
              {t('Cancel')}
            </Button>
            <Button
              disabled={
                !title.trim() || !content.trim() || createMutation.isPending
              }
              onClick={() => createMutation.mutate()}
            >
              {t('Submit')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Sheet
        open={selectedId != null}
        onOpenChange={(open) => {
          if (!open) {
            setSelectedId(null)
            setReplyContent('')
            setReplyParentId(null)
          }
        }}
      >
        <SheetContent className='flex w-full flex-col sm:max-w-lg'>
          <SheetHeader>
            <SheetTitle>
              {detail?.title ?? t('Ticket details')}
            </SheetTitle>
          </SheetHeader>
          {(() => {
            if (detailQuery.isLoading) {
              return (
                <p className='text-muted-foreground px-4'>{t('Loading...')}</p>
              )
            }
            if (!detail) {
              return null
            }
            return (
            <div className='flex min-h-0 flex-1 flex-col gap-4 overflow-hidden px-4 pb-4'>
              <div className='flex flex-wrap items-center gap-2'>
                <StatusBadge status={detail.status} />
                <span className='text-muted-foreground text-xs'>
                  {t('Created')}: {formatTs(detail.created_at)}
                </span>
                {mode === 'admin' ? (
                  <span className='text-muted-foreground text-xs'>
                    {t('User')}: {detail.user_id}
                  </span>
                ) : null}
              </div>
              <div className='bg-muted/40 rounded-md p-3 text-sm whitespace-pre-wrap'>
                {detail.content}
              </div>
              <div className='min-h-0 flex-1 space-y-3 overflow-y-auto'>
                {(detail.replies ?? []).map((reply) => (
                  <div
                    key={reply.id}
                    className={cn(
                      'rounded-md border p-3 text-sm',
                      reply.is_admin ? 'bg-primary/5' : 'bg-background'
                    )}
                  >
                    <div className='mb-1 flex items-center justify-between gap-2'>
                      <span className='font-medium'>
                        {reply.is_admin ? t('Admin') : t('User')}
                      </span>
                      <span className='text-muted-foreground text-xs'>
                        {formatTs(reply.created_at)}
                      </span>
                    </div>
                    {reply.parent_id ? (
                      <p className='text-muted-foreground mb-2 text-xs'>
                        {t('Replying to')}:{' '}
                        {findParentSnippet(detail.replies, reply.parent_id)}
                      </p>
                    ) : null}
                    <p className='whitespace-pre-wrap'>{reply.content}</p>
                    {detail.status !== 'closed' ? (
                      <Button
                        className='mt-2'
                        size='sm'
                        variant='ghost'
                        onClick={() => setReplyParentId(reply.id)}
                      >
                        {t('Reply')}
                      </Button>
                    ) : null}
                  </div>
                ))}
              </div>
              {detail.status !== 'closed' ? (
                <div className='space-y-2 border-t pt-3'>
                  {parentSnippet ? (
                    <div className='bg-muted flex items-start justify-between gap-2 rounded-md p-2 text-xs'>
                      <span>
                        {t('Replying to')}: {parentSnippet}
                      </span>
                      <Button
                        size='sm'
                        variant='ghost'
                        onClick={() => setReplyParentId(null)}
                      >
                        {t('Cancel')}
                      </Button>
                    </div>
                  ) : null}
                  <Textarea
                    rows={3}
                    placeholder={t('Write a reply')}
                    value={replyContent}
                    onChange={(e) => setReplyContent(e.target.value)}
                  />
                  <div className='flex flex-wrap gap-2'>
                    <Button
                      disabled={
                        !replyContent.trim() || replyMutation.isPending
                      }
                      onClick={() => replyMutation.mutate()}
                    >
                      {t('Send reply')}
                    </Button>
                    <Button
                      variant='outline'
                      onClick={() => closeMutation.mutate(detail.id)}
                    >
                      {t('Close')}
                    </Button>
                    <Button
                      variant='destructive'
                      onClick={() => setDeleteId(detail.id)}
                    >
                      {t('Delete')}
                    </Button>
                  </div>
                </div>
              ) : (
                <div className='flex gap-2 border-t pt-3'>
                  <Button
                    variant='destructive'
                    onClick={() => setDeleteId(detail.id)}
                  >
                    {t('Delete')}
                  </Button>
                </div>
              )}
            </div>
            )
          })()}
        </SheetContent>
      </Sheet>

      <ConfirmDialog
        open={deleteId != null}
        onOpenChange={(open) => {
          if (!open) setDeleteId(null)
        }}
        title={t('Delete ticket')}
        desc={t('This will permanently delete the ticket and all replies.')}
        destructive
        handleConfirm={() => {
          if (deleteId != null) deleteMutation.mutate(deleteId)
        }}
        isLoading={deleteMutation.isPending}
      />
    </>
  )
}
