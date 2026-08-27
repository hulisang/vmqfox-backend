import { create } from 'zustand'

interface AuthState {
  token: string | null
  username: string | null
  expiresAt: number | null
  setAuth: (token: string, username: string, expiresAt: number) => void
  clearAuth: () => void
  isAuthenticated: () => boolean
}

export const useAuthStore = create<AuthState>((set, get) => ({
  token: localStorage.getItem('vmq_token'),
  username: localStorage.getItem('vmq_user'),
  expiresAt: Number(localStorage.getItem('vmq_exp')) || null,

  setAuth: (token, username, expiresAt) => {
    localStorage.setItem('vmq_token', token)
    localStorage.setItem('vmq_user', username)
    localStorage.setItem('vmq_exp', String(expiresAt))
    set({ token, username, expiresAt })
  },

  clearAuth: () => {
    localStorage.removeItem('vmq_token')
    localStorage.removeItem('vmq_user')
    localStorage.removeItem('vmq_exp')
    set({ token: null, username: null, expiresAt: null })
  },

  isAuthenticated: () => {
    const { token, expiresAt } = get()
    if (!token) return false
    if (expiresAt && Date.now() / 1000 > expiresAt) {
      get().clearAuth()
      return false
    }
    return true
  }
}))

interface ThemeState {
  theme: 'light' | 'dark'
  setTheme: (theme: 'light' | 'dark') => void
  toggleTheme: () => void
}

export const useThemeStore = create<ThemeState>((set) => ({
  theme: (localStorage.getItem('vmq_theme') as 'light' | 'dark') || 'light',
  setTheme: (theme) => {
    localStorage.setItem('vmq_theme', theme)
    if (theme === 'dark') {
      document.documentElement.classList.add('dark')
    } else {
      document.documentElement.classList.remove('dark')
    }
    set({ theme })
  },
  toggleTheme: () => {
    set((state) => {
      const nextTheme = state.theme === 'dark' ? 'light' : 'dark'
      localStorage.setItem('vmq_theme', nextTheme)
      if (nextTheme === 'dark') {
        document.documentElement.classList.add('dark')
      } else {
        document.documentElement.classList.remove('dark')
      }
      return { theme: nextTheme }
    })
  }
}))
