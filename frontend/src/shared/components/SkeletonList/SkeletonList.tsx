import { Skeleton } from '@mui/material'
import type { SkeletonProps } from '@mui/material'

interface Props {
  count?: number
  variant?: SkeletonProps['variant']
  height?: number
  spacing?: number
}

export default function SkeletonList({ count = 4, variant = 'rectangular', height, spacing = 1 }: Props) {
  return (
    <>
      {Array.from({ length: count }).map((_, i) => (
        <Skeleton
          key={i}
          variant={variant}
          height={height}
          sx={{ mb: spacing, ...(variant === 'rectangular' ? { borderRadius: 1 } : {}) }}
        />
      ))}
    </>
  )
}
