import React, { useState } from 'react'
import { authApi } from '@/api'
import { useAuthStore } from '@/stores'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { ShieldCheck, User, Lock, ArrowRight } from 'lucide-react'
import { toast } from 'sonner'

export const LoginView: React.FC = () => {
  const { setAuth } = useAuthStore()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!username || !password) {
      toast.error('请输入用户名和密码')
      return
    }

    setLoading(true)
    try {
      const res = await authApi.login({ username, password })
      setAuth(res.accessToken, res.username, res.expiresAt)
      toast.success('登录成功，欢迎回来！')
      window.location.hash = '#/dashboard'
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '登录失败，请检查账号密码')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center p-4 relative">
      <Card className="max-w-sm w-full p-8 shadow-2xl border-border/80 bg-card/85 backdrop-blur-md">
        <CardHeader className="p-0 pb-6 text-center">
          <div className="size-12 rounded-2xl bg-primary/15 text-primary flex items-center justify-center mx-auto mb-3 shadow-inner border border-primary/20">
            <ShieldCheck className="size-6" />
          </div>
          <CardTitle className="text-xl font-bold tracking-tight">VMQFox 控制台</CardTitle>
          <p className="text-xs text-muted-foreground mt-1">请输入管理员凭据以进入管理系统</p>
        </CardHeader>

        <CardContent className="p-0">
          <form onSubmit={handleLogin} className="space-y-4">
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">管理员账号</label>
              <div className="relative">
                <User className="absolute left-3.5 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
                <Input
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  placeholder="admin"
                  className="pl-10"
                  required
                />
              </div>
            </div>

            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">管理员密码</label>
              <div className="relative">
                <Lock className="absolute left-3.5 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
                <Input
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="••••••••"
                  className="pl-10"
                  required
                />
              </div>
            </div>

            <Button
              type="submit"
              disabled={loading}
              className="w-full gap-2 rounded-2xl mt-4"
            >
              <span>{loading ? '正在验证...' : '登 录'}</span>
              <ArrowRight className="size-4" />
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
