import { defineConfig, Plugin } from 'vite'
import react from '@vitejs/plugin-react'
import fs from 'fs'
import path from 'path'

const embedOutDir = path.resolve(__dirname, '../internal/http/static/out')

/**
 * Go 的 `go:embed all:out` 要求该目录在干净克隆中就存在，
 * 因此仓库里保留了 `.gitkeep` 占位。但 `emptyOutDir` 会连它一起清掉，
 * 导致每次构建后 git 都出现一条无意义的删除记录。此插件在构建结束后补回占位文件。
 */
function keepEmbedDirPlaceholder(): Plugin {
  return {
    name: 'vmqfox-keep-embed-placeholder',
    apply: 'build',
    closeBundle() {
      fs.mkdirSync(embedOutDir, { recursive: true })
      fs.writeFileSync(path.join(embedOutDir, '.gitkeep'), '')
    },
  }
}

export default defineConfig({
  plugins: [react(), keepEmbedDirPlaceholder()],
  // Go 服务在 / 与 /index.html 两个路径提供同一份 index.html，
  // 使用根绝对路径可保证两处的资源引用都解析到 /assets/。
  base: '/',
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 5173,
    host: true,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: embedOutDir,
    emptyOutDir: true,
  },
})
