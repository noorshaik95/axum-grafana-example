import { cn } from '@/lib/utils';

export const BentoGrid = ({
  className,
  children,
}: {
  className?: string;
  children?: React.ReactNode;
}) => {
  return (
    <div
      className={cn(
        // Single column on mobile, 3 columns on desktop (Requirement 13.2)
        'grid auto-rows-auto sm:auto-rows-[16rem] md:auto-rows-[18rem] grid-cols-1 md:grid-cols-3 gap-3 sm:gap-4 max-w-7xl mx-auto',
        className
      )}
    >
      {children}
    </div>
  );
};

export const BentoGridItem = ({
  className,
  title,
  description,
  header,
  icon,
  float = false,
}: {
  className?: string;
  title?: string | React.ReactNode;
  description?: string | React.ReactNode;
  header?: React.ReactNode;
  icon?: React.ReactNode;
  float?: boolean;
}) => {
  return (
    <div
      className={cn(
        // Base styles with glassmorphism (Requirement 8.2)
        // Responsive padding (Requirement 13.3)
        'row-span-1 rounded-xl group/bento transition-all duration-300 p-3 sm:p-4 justify-between flex flex-col space-y-3 sm:space-y-4',
        // Glass card styling
        'glass-card',
        // 3D transform on hover (Requirement 8.2)
        'card-3d',
        // Optional float animation
        float && 'float',
        className
      )}
    >
      {header}
      <div className="group-hover/bento:translate-x-2 transition duration-200">
        {icon}
        {/* Responsive text sizes - minimum 14px (Requirement 13.3) */}
        <div className="font-semibold text-lg sm:text-xl text-slate-100 mb-1.5 sm:mb-2 mt-1.5 sm:mt-2 tracking-wide">
          {title}
        </div>
        {/* Minimum 14px font size on mobile (Requirement 13.3) */}
        <div className="font-sans font-normal text-slate-300 text-sm sm:text-xs leading-relaxed">
          {description}
        </div>
      </div>
    </div>
  );
};
