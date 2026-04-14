/**
 * Central place for all visual constants.
 * Change these values to restyle the entire application.
 */
export const THEME = {
  // Brand palette
  colors: {
    primary: '#6366f1',    // Indigo — buttons, links, active states
    secondary: '#8b5cf6',  // Violet — secondary actions
    coin: '#d97706',       // Amber/Gold — coin amounts and icons
    success: '#16a34a',    // Green — positive transactions, approvals
    danger: '#dc2626',     // Red — negative transactions, errors
    background: '#f1f5f9', // Light slate — page background
    surface: '#ffffff',    // White — card backgrounds
    border: '#e2e8f0',     // Subtle border color
    textPrimary: '#0f172a',
    textMuted: '#64748b',
  },

  // Gamification constants
  progress: {
    // Target coins for the main prize ("День с МК")
    mainPrizeTarget: 4800,
    mainPrizeLabel: 'День с МК',
  },
} as const
