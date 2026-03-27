import { cn } from '@/lib/utils'
import logoSvg from '@/assets/logo.svg'

interface TrustMeshLogoProps {
  size?: number
  className?: string
}

export function TrustMeshLogo({ size = 24, className }: TrustMeshLogoProps) {
  return (
    <img
      src={logoSvg}
      width={size}
      height={size}
      alt="TrustMesh"
      className={cn('shrink-0', className)}
    />
  )
}
