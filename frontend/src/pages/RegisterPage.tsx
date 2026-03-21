import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { useAuthStore } from '@/stores/authStore'
import * as authApi from '@/api/auth'
import { ApiRequestError } from '@/api/client'

const MIN_PASSWORD_LENGTH = 8

function getRegisterErrorMessage(error: ApiRequestError) {
  if (error.code !== 'VALIDATION_ERROR') {
    return error.message
  }
  const passwordError = typeof error.details.password === 'string' ? error.details.password : null
  if (passwordError) {
    return `密码${passwordError}`
  }
  return error.message
}

export function RegisterPage() {
  const [email, setEmail] = useState('')
  const [name, setName] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const setAuth = useAuthStore((s) => s.setAuth)
  const navigate = useNavigate()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    if (password.length < MIN_PASSWORD_LENGTH) {
      setError(`密码至少需要 ${MIN_PASSWORD_LENGTH} 位`)
      return
    }
    setLoading(true)
    try {
      const res = await authApi.register({ email, name, password })
      setAuth(res.data.token, res.data.user)
      navigate('/projects')
    } catch (err) {
      if (err instanceof ApiRequestError) setError(getRegisterErrorMessage(err))
      else setError('注册失败，请重试')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-gradient-to-br from-background via-background to-primary/5 px-4">
      <div className="w-full max-w-md">
        <div className="mb-8 text-center">
          <div className="mx-auto mb-4 flex size-12 items-center justify-center rounded-2xl bg-primary text-primary-foreground text-lg font-bold">
            T
          </div>
          <h1 className="text-2xl font-bold">创建账号</h1>
          <p className="mt-1 text-sm text-muted-foreground">注册 TrustMesh 开始 AI 驱动的协作</p>
        </div>
        <Card>
          <CardHeader className="pb-4">
            <CardTitle className="text-lg">注册</CardTitle>
            <CardDescription>填写信息创建新账号</CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit} className="flex flex-col gap-4">
              <div className="flex flex-col gap-2">
                <label className="text-sm font-medium">邮箱</label>
                <Input
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="you@example.com"
                  required
                  autoFocus
                />
              </div>
              <div className="flex flex-col gap-2">
                <label className="text-sm font-medium">用户名</label>
                <Input
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="输入用户名"
                  required
                />
              </div>
              <div className="flex flex-col gap-2">
                <label className="text-sm font-medium">密码</label>
                <Input
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder={`设置密码（至少 ${MIN_PASSWORD_LENGTH} 位）`}
                  required
                  minLength={MIN_PASSWORD_LENGTH}
                />
                <p className="text-xs text-muted-foreground">密码至少需要 {MIN_PASSWORD_LENGTH} 位</p>
              </div>
              {error && <p className="text-sm text-destructive">{error}</p>}
              <Button type="submit" className="w-full" disabled={loading}>
                {loading ? '注册中...' : '注册'}
              </Button>
            </form>
            <p className="mt-4 text-center text-sm text-muted-foreground">
              已有账号？{' '}
              <Link to="/login" className="text-primary hover:underline font-medium">
                登录
              </Link>
            </p>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
