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
import { api } from '@/lib/api'

import type {
  ApiResponse,
  ListTicketsParams,
  Ticket,
  TicketDetail,
  TicketMode,
  TicketPageData,
  TicketReply,
} from './types'

function buildQuery(params: ListTicketsParams): string {
  const query = new URLSearchParams()
  query.set('p', String(params.p ?? 1))
  query.set('page_size', String(params.page_size ?? 20))
  if (params.status) query.set('status', params.status)
  if (params.keyword) query.set('keyword', params.keyword)
  if (params.user_id) query.set('user_id', String(params.user_id))
  return query.toString()
}

export async function listTickets(
  mode: TicketMode,
  params: ListTicketsParams = {}
): Promise<ApiResponse<TicketPageData>> {
  const path =
    mode === 'admin'
      ? `/api/ticket/?${buildQuery(params)}`
      : `/api/ticket/self?${buildQuery(params)}`
  const res = await api.get(path)
  return res.data
}

export async function getTicket(
  mode: TicketMode,
  id: number
): Promise<ApiResponse<TicketDetail>> {
  const path =
    mode === 'admin' ? `/api/ticket/${id}` : `/api/ticket/self/${id}`
  const res = await api.get(path)
  return res.data
}

export async function createTicket(data: {
  title: string
  content: string
}): Promise<ApiResponse<Ticket>> {
  const res = await api.post('/api/ticket/', data)
  return res.data
}

export async function replyTicket(
  mode: TicketMode,
  id: number,
  data: { content: string; parent_id?: number | null }
): Promise<ApiResponse<TicketReply>> {
  const path =
    mode === 'admin'
      ? `/api/ticket/${id}/replies`
      : `/api/ticket/self/${id}/replies`
  const res = await api.post(path, data)
  return res.data
}

export async function closeTicket(
  mode: TicketMode,
  id: number
): Promise<ApiResponse> {
  const path =
    mode === 'admin'
      ? `/api/ticket/${id}/close`
      : `/api/ticket/self/${id}/close`
  const res = await api.put(path)
  return res.data
}

export async function deleteTicket(
  mode: TicketMode,
  id: number
): Promise<ApiResponse> {
  const path =
    mode === 'admin' ? `/api/ticket/${id}` : `/api/ticket/self/${id}`
  const res = await api.delete(path)
  return res.data
}
