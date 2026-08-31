import { create } from 'zustand'

/**
 * 管理员会话状态。
 *
 * access token 只写入 sessionStorage：关闭标签页即失效，
 * 不再像 localStorage 那样长期驻留在浏览器磁盘上被任意脚本读取。
 * 主题偏好不是凭据，仍保留在 localStorage 以便跨会话记忆。
 */
const TOKEN_KEY = 'vmq_token'
const USER_KEY = 'vmq_user'
const EXPIRES_KEY = 'vmq_exp'

/** 读取会话存储；在禁用存储的隐私模式下返回 null 而不是抛错 */
function readSession(key: string): string | null {
  try {
    return sessionStorage.getItem(key)
  } catch {
    return null
  }
}

function writeSession(key: string, value: string): void {
  try {
    sessionStorage.setItem(key, value)
  } catch {
    // 存储不可用时仅依赖内存中的状态，不影响本次会话可用性
  }
}

function removeSession(key: string): void {
  try {
    sessionStorage.removeItem(key)
  } catch {
    // 同上，忽略存储异常
  }
}

/** 清理历史版本遗留在 localStorage 中的长期 Token，避免旧凭据继续留存 */
function purgeLegacyLocalToken(): void {
  try {
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(USER_KEY)
    localStorage.removeItem(EXPIRES_KEY)
  } catch {
    // 忽略存储异常
  }
}

purgeLegacyLocalToken()

interface AuthState {
  token: string | null
  username: string | null
  expiresAt: number | null
  setAuth: (token: string, username: string, expiresAt: number) => void
  clearAuth: () => void
  isAuthenticated: () => boolean
}

export const useAuthStore = create<AuthState>((set, get) => ({
  token: readSession(TOKEN_KEY),
  username: readSession(USER_KEY),
  expiresAt: Number(readSession(EXPIRES_KEY)) || null,

  setAuth: (token, username, expiresAt) => {
    writeSession(TOKEN_KEY, token)
    writeSession(USER_KEY, username)
    writeSession(EXPIRES_KEY, String(expiresAt))
    set({ token, username, expiresAt })
  },

  clearAuth: () => {
    removeSession(TOKEN_KEY)
    removeSession(USER_KEY)
    removeSession(EXPIRES_KEY)
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
  },
}))

type Theme = 'light' | 'dark'

const THEME_KEY = 'vmq_theme'

function applyThemeClass(theme: Theme): void {
  if (theme === 'dark') {
    document.documentElement.classList.add('dark')
  } else {
    document.documentElement.classList.remove('dark')
  }
}

function readTheme(): Theme {
  try {
    const stored = localStorage.getItem(THEME_KEY)
    if (stored === 'dark' || stored === 'light') return stored
  } catch {
    // 忽略存储异常
  }
  return document.documentElement.classList.contains('dark') ? 'dark' : 'light'
}

function writeTheme(theme: Theme): void {
  try {
    localStorage.setItem(THEME_KEY, theme)
  } catch {
    // 忽略存储异常
  }
}

interface ThemeState {
  theme: Theme
  setTheme: (theme: Theme) => void
  toggleTheme: () => void
}

export const useThemeStore = create<ThemeState>((set) => ({
  theme: readTheme(),
  setTheme: (theme) => {
    writeTheme(theme)
    applyThemeClass(theme)
    set({ theme })
  },
  toggleTheme: () => {
    set((state) => {
      const nextTheme: Theme = state.theme === 'dark' ? 'light' : 'dark'
      writeTheme(nextTheme)
      applyThemeClass(nextTheme)
      return { theme: nextTheme }
    })
  },
}))
