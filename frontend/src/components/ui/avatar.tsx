import type { CSSProperties, HTMLAttributes } from 'react'
import type { AgentRole } from '@/types'
import { cn } from '@/lib/utils'

interface AvatarProps extends HTMLAttributes<HTMLDivElement> {
  fallback: string
  seed?: string
  size?: 'sm' | 'md' | 'lg'
  kind?: 'user' | 'agent'
  role?: AgentRole
}

interface AvatarTheme {
  backgroundImage: string
}

const sizeClasses = {
  sm: 'size-7 text-[11px]',
  md: 'size-9 text-sm',
  lg: 'size-11 text-base',
}

const avatarThemes: AvatarTheme[] = [
  {
    backgroundImage: 'linear-gradient(135deg, #2563eb 0%, #06b6d4 100%)',
  },
  {
    backgroundImage: 'linear-gradient(135deg, #7c3aed 0%, #ec4899 100%)',
  },
  {
    backgroundImage: 'linear-gradient(135deg, #059669 0%, #22c55e 100%)',
  },
  {
    backgroundImage: 'linear-gradient(135deg, #ea580c 0%, #f59e0b 100%)',
  },
  {
    backgroundImage: 'linear-gradient(135deg, #0f766e 0%, #14b8a6 100%)',
  },
  {
    backgroundImage: 'linear-gradient(135deg, #db2777 0%, #f43f5e 100%)',
  },
]

function hashSeed(value: string) {
  let hash = 0
  for (let i = 0; i < value.length; i += 1) {
    hash = (hash * 31 + value.charCodeAt(i)) >>> 0
  }
  return hash
}

function getInitials(value: string) {
  const trimmed = value.trim()
  if (!trimmed) return '?'

  const compact = trimmed.replace(/\s+/g, ' ')
  const cjkValue = compact.replace(/[\s·•._-]/g, '')
  if (/[\u3400-\u9fff]/.test(cjkValue)) {
    return cjkValue.slice(0, 2)
  }

  const words = compact.split(/[\s._-]+/).filter(Boolean)
  if (words.length >= 2) {
    return `${words[0][0]}${words[1][0]}`.toUpperCase()
  }

  return compact.slice(0, 2).toUpperCase()
}

function getTheme(seed: string) {
  return avatarThemes[hashSeed(seed) % avatarThemes.length]
}

export function Avatar({
  fallback,
  seed,
  size = 'md',
  kind = 'user',
  role,
  className,
  style,
  ...props
}: AvatarProps) {
  void kind
  void role
  const theme = getTheme(seed ?? fallback)
  const avatarStyle: CSSProperties = {
    backgroundImage: theme.backgroundImage,
    ...style,
  }

  return (
    <div
      className={cn(
        'inline-flex items-center justify-center shrink-0 overflow-hidden font-semibold text-white select-none',
        'rounded-full',
        sizeClasses[size],
        className
      )}
      style={avatarStyle}
      {...props}
    >
      <span>{getInitials(fallback)}</span>
    </div>
  )
}
