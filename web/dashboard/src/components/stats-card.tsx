import type { LucideIcon } from 'lucide-react'
import { cn } from '@/lib/utils'

interface StatsCardProps {
  title: string
  value: string | number
  description?: string
  icon: LucideIcon
  trend?: {
    value: number
    positive: boolean
  }
  color?: 'cyan' | 'violet' | 'emerald' | 'amber' | 'rose'
  className?: string
}

const colorVariants = {
  cyan: 'from-cyan-500/20 to-blue-500/20 border-cyan-500/30 text-cyan-400',
  violet: 'from-violet-500/20 to-purple-500/20 border-violet-500/30 text-violet-400',
  emerald: 'from-emerald-500/20 to-green-500/20 border-emerald-500/30 text-emerald-400',
  amber: 'from-amber-500/20 to-orange-500/20 border-amber-500/30 text-amber-400',
  rose: 'from-rose-500/20 to-pink-500/20 border-rose-500/30 text-rose-400',
}

export function StatsCard({
  title,
  value,
  description,
  icon: Icon,
  trend,
  color = 'cyan',
  className,
}: StatsCardProps) {
  return (
    <div
      className={cn(
        'relative overflow-hidden rounded-2xl bg-gradient-to-br p-6 backdrop-blur-xl transition-all duration-300 hover:scale-[1.02]',
        colorVariants[color],
        'border',
        className
      )}
    >
      <div className="flex items-start justify-between">
        <div className="space-y-2">
          <p className="text-sm font-medium text-white/70">{title}</p>
          <h3 className="text-3xl font-bold text-white">{value}</h3>
          {description && (
            <p className="text-xs text-white/50">{description}</p>
          )}
          {trend && (
            <div className={cn(
              'flex items-center gap-1 text-xs font-medium',
              trend.positive ? 'text-emerald-400' : 'text-rose-400'
            )}>
              <span>{trend.positive ? '↑' : '↓'} {Math.abs(trend.value)}%</span>
              <span className="text-white/50">vs last hour</span>
            </div>
          )}
        </div>
        <div className={cn(
          'rounded-xl p-3 backdrop-blur-sm',
          'bg-white/10'
        )}>
          <Icon className={cn('h-6 w-6', colorVariants[color].split(' ').pop())} />
        </div>
      </div>
      
      {/* Decorative gradient orb */}
      <div className={cn(
        'absolute -right-6 -top-6 h-24 w-24 rounded-full opacity-30 blur-2xl',
        colorVariants[color].split(' ')[1].replace('/20', '')
      )} />
    </div>
  )
}
