// ─── Enums ───────────────────────────────────────────────────────────────────

export type UserRole = 'manager' | 'resident'
export type RewardStatus = 'available' | 'hidden'
export type RewardClaimStatus = 'pending' | 'completed' | 'cancelled'
export type TaskSubmissionStatus = 'pending' | 'approved' | 'rejected'
export type TransactionReason = 'event' | 'task' | 'manual' | 'reward'

// ─── Domain models ────────────────────────────────────────────────────────────

export interface User {
  id: number
  telegram_id: number
  name: string
  username: string
  photo_url?: string
  roles: UserRole[]
  is_blocked: boolean
  created_at: string
}

export interface Season {
  id: number
  title: string
  start_date: string
  end_date: string
  is_active: boolean
}

export interface Balance {
  id: number
  user_id: number
  season_id: number
  balance: number
  total_earned: number
  updated_at: string
}

export interface SeasonMemberWithSeason extends Balance {
  season: Season
}

export interface Transaction {
  id: number
  member_id: number
  amount: number
  reason: TransactionReason
  ref_id?: number
  ref_title: string
  created_at: string
}

export interface Task {
  id: number
  season_id: number
  title: string
  category: string
  description?: string
  reward_coins: number
  sort_order: number
  is_active: boolean
}

export interface TaskSubmission {
  id: number
  user_id: number
  user_name?: string
  task_id: number
  status: TaskSubmissionStatus
  comment: string
  review_comment: string
  reviewer_id?: number
  reviewer_name?: string
  submitted_at: string
  reviewed_at?: string
}

export interface Reward {
  id: number
  season_id: number
  title: string
  description?: string
  cost_coins: number
  limit?: number
  status: RewardStatus
}

export interface RewardClaim {
  id: number
  user_id: number
  reward_id: number
  season_id: number
  status: RewardClaimStatus
  created_at: string
}

export interface TemplateVar { key: string; label: string }

export interface LeaderboardEntry {
  position: number
  user_id: number
  name: string
  username: string
  photo_url?: string
  balance: number
  is_current: boolean
}

export interface Template {
  id: number
  codename: string
  body: string
  vars: TemplateVar[]
  created_at: string
  updated_at: string
}

export interface TemplateHistory {
  id: number
  template_id: number
  body: string
  changed_by: number
  changed_at: string
}

// ─── Telegram Login Widget callback payload ───────────────────────────────────

export interface TelegramAuthData {
  id: number
  first_name: string
  last_name?: string
  username?: string
  photo_url?: string
  auth_date: number
  hash: string
}

