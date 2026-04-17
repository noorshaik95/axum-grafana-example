import { cn } from '@/lib/utils'

export const BentoGrid = ({
  className,
  children,
}: {
  className?: string
  children?: React.ReactNode
}) => {
  return (
    <div
      className={cn(
        'grid md:auto-rows-[18rem] grid-cols-1 md:grid-cols-3 gap-4 max-w-7xl mx-auto',
        className
      )}
    >
      {children}
    </div>
  )
}

export const BentoGridItem = ({
  className,
  title,
  description,
  header,
  icon,
  float = false,
}: {
  className?: string
  title?: string | React.ReactNode
  description?: string | React.ReactNode
  header?: React.ReactNode
  icon?: React.ReactNode
  float?: boolean
}) => {
  return (
    <div
      className={cn(
        // Base styles with glassmorphism
        'row-span-1 rounded-xl group/bento transition-all duration-300 p-4 justify-between flex flex-col space-y-4',
        // Glass card styling
        'glass-card',
        // 3D transform on hover
        'card-3d',
        // Optional float animation
        float && 'float',
        className
      )}
    >
      {header}
      <div className="group-hover/bento:translate-x-2 transition duration-200">
        {icon}
        <div className="font-semibold text-xl text-slate-100 mb-2 mt-2 tracking-wide">{title}</div>
        <div className="font-sans font-normal text-slate-300 text-xs">{description}</div>
      </div>
    </div>
  )
}
