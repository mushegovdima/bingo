import apiClient from './client'
import { Balance, LeaderboardEntry, Transaction, SeasonMemberWithSeason } from '@/types'

export const balanceApi = {
  getBalance: async (seasonId: number): Promise<Balance> => {
    const res = await apiClient.get<Balance>(
      `/balance/${seasonId}`,
    )
    return res.data
  },

  getTransactions: async (seasonId: number): Promise<Transaction[]> => {
    const res = await apiClient.get<Transaction[]>(
      `/balance/${seasonId}/transactions`,
    )
    return res.data
  },

  listMy: async (): Promise<SeasonMemberWithSeason[]> => {
    const res = await apiClient.get<SeasonMemberWithSeason[]>('/balance/my')
    return res.data
  },

  joinSeason: async (seasonId: number): Promise<Balance> => {
    const res = await apiClient.post<Balance>(`/balance/${seasonId}/join`)
    return res.data
  },

  getLeaderboard: async (seasonId: number): Promise<LeaderboardEntry[]> => {
    const res = await apiClient.get<LeaderboardEntry[]>(`/balance/leaderboard/${seasonId}`)
    return res.data
  },

  getFullLeaderboard: async (seasonId: number): Promise<LeaderboardEntry[]> => {
    const res = await apiClient.get<LeaderboardEntry[]>(`/balance/leaderboard/${seasonId}/full`)
    return res.data
  },
}
