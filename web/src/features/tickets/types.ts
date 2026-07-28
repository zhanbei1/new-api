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
export type TicketStatus = 'open' | 'replied' | 'closed'

export type TicketMode = 'user' | 'admin'

export interface Ticket {
  id: number
  user_id: number
  title: string
  content: string
  status: TicketStatus
  created_at: number
  updated_at: number
}

export interface TicketReply {
  id: number
  ticket_id: number
  user_id: number
  is_admin: boolean
  content: string
  parent_id?: number | null
  created_at: number
}

export interface TicketDetail extends Ticket {
  replies: TicketReply[]
}

export interface ApiResponse<T = unknown> {
  success: boolean
  message?: string
  data?: T
}

export interface TicketPageData {
  items: Ticket[]
  total: number
  page: number
  page_size: number
}

export interface ListTicketsParams {
  p?: number
  page_size?: number
  status?: string
  keyword?: string
  user_id?: number
}
