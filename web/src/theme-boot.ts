/**
 * 主题初始化。
 *
 * 原先这段逻辑是 index.html 里的内联 <script>，会强制 CSP 放开 'unsafe-inline'。
 * 移到独立模块后，入口可以使用严格的 script-src 'self'。
 */
function bootstrapTheme(): void {
  try {
    const stored = localStorage.getItem('vmq_theme')
    const prefersDark =
      typeof window.matchMedia === 'function' &&
      window.matchMedia('(prefers-color-scheme: dark)').matches
    const theme = stored === 'dark' || stored === 'light' ? stored : prefersDark ? 'dark' : 'light'

    if (theme === 'dark') {
      document.documentElement.classList.add('dark')
    } else {
      document.documentElement.classList.remove('dark')
    }
  } catch {
    // 存储不可用时保持默认浅色主题
  }
}

bootstrapTheme()
