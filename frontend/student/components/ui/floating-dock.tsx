'use client';

import { cn } from '@/lib/utils';
import { AnimatePresence, motion, useMotionValue, useSpring, useTransform } from 'framer-motion';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { useRef, useState } from 'react';

export const FloatingDock = ({
  items,
  desktopClassName,
  mobileClassName,
}: {
  items: { title: string; icon: React.ReactNode; href: string }[];
  desktopClassName?: string;
  mobileClassName?: string;
}) => {
  return (
    <>
      <FloatingDockDesktop items={items} className={desktopClassName} />
      <FloatingDockMobile items={items} className={mobileClassName} />
    </>
  );
};

const FloatingDockMobile = ({
  items,
  className,
}: {
  items: { title: string; icon: React.ReactNode; href: string }[];
  className?: string;
}) => {
  const [open, setOpen] = useState(false);
  const pathname = usePathname();

  const isActive = (href: string) => {
    if (href === '/') return pathname === '/';
    return pathname.startsWith(href);
  };

  return (
    // Mobile dock container - optimized for 375px viewport (Requirement 13.1)
    <div className={cn('relative block md:hidden', className)}>
      <AnimatePresence>
        {open && (
          <motion.div
            layoutId="nav"
            className="absolute bottom-full mb-3 left-1/2 -translate-x-1/2 flex flex-row flex-wrap justify-center gap-3 p-3 glass-panel rounded-2xl max-w-[calc(100vw-2rem)]"
          >
            {items.map((item, idx) => {
              const active = isActive(item.href);
              return (
                <motion.div
                  key={item.title}
                  initial={{ opacity: 0, scale: 0.8 }}
                  animate={{
                    opacity: 1,
                    scale: 1,
                  }}
                  exit={{
                    opacity: 0,
                    scale: 0.8,
                    transition: {
                      delay: idx * 0.03,
                    },
                  }}
                  transition={{
                    type: 'spring',
                    stiffness: 300,
                    damping: 25,
                    delay: (items.length - 1 - idx) * 0.03,
                  }}
                >
                  <Link
                    href={item.href}
                    onClick={() => setOpen(false)}
                    className={cn(
                      // Minimum touch target 44x44px for accessibility
                      // Slate styling - no neon glow (Requirement 9.2, 9.3)
                      'h-11 w-11 min-h-[44px] min-w-[44px] rounded-full glass-card flex items-center justify-center transition-all duration-300',
                      active &&
                        'border-slate-400 dark:border-slate-500 bg-slate-100/20 dark:bg-slate-700/50'
                    )}
                    aria-label={item.title}
                    aria-current={active ? 'page' : undefined}
                  >
                    <div
                      className={cn(
                        'h-5 w-5 transition-all duration-300',
                        active ? 'text-blue-500' : 'text-slate-600 dark:text-slate-300'
                      )}
                    >
                      {item.icon}
                    </div>
                  </Link>
                </motion.div>
              );
            })}
          </motion.div>
        )}
      </AnimatePresence>
      <button
        onClick={() => setOpen(!open)}
        // Minimum touch target 44x44px for accessibility (Requirement 13.1)
        // Slate styling - no neon glow on hover (Requirement 9.2, 9.3)
        className="h-12 w-12 min-h-[44px] min-w-[44px] rounded-full glass-panel flex items-center justify-center hover:bg-slate-200/50 dark:hover:bg-slate-700/50 hover:border-slate-400 dark:hover:border-slate-500 active:scale-95 transition-all duration-300"
        aria-label={open ? 'Close navigation menu' : 'Open navigation menu'}
        aria-expanded={open}
      >
        <div className="h-5 w-5 text-slate-600 dark:text-slate-300">
          {open ? (
            // Close icon when menu is open
            <svg
              xmlns="http://www.w3.org/2000/svg"
              width="24"
              height="24"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <line x1="18" x2="6" y1="6" y2="18" />
              <line x1="6" x2="18" y1="6" y2="18" />
            </svg>
          ) : (
            // Menu icon when closed
            <svg
              xmlns="http://www.w3.org/2000/svg"
              width="24"
              height="24"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <line x1="4" x2="20" y1="12" y2="12" />
              <line x1="4" x2="20" y1="6" y2="6" />
              <line x1="4" x2="20" y1="18" y2="18" />
            </svg>
          )}
        </div>
      </button>
    </div>
  );
};

const FloatingDockDesktop = ({
  items,
  className,
}: {
  items: { title: string; icon: React.ReactNode; href: string }[];
  className?: string;
}) => {
  const mouseX = useMotionValue(Infinity);
  const pathname = usePathname();

  const isActive = (href: string) => {
    if (href === '/') return pathname === '/';
    return pathname.startsWith(href);
  };

  return (
    <motion.div
      onMouseMove={(e) => mouseX.set(e.pageX)}
      onMouseLeave={() => mouseX.set(Infinity)}
      className={cn(
        // Frosted glass background with enhanced blur (Requirement 5.1)
        'mx-auto hidden md:flex h-16 gap-3 items-end rounded-2xl glass-panel px-4 pb-3',
        className
      )}
    >
      {items.map((item) => (
        <IconContainer mouseX={mouseX} key={item.title} isActive={isActive(item.href)} {...item} />
      ))}
    </motion.div>
  );
};

// Spring physics configuration for smooth, natural motion (Requirement 5.4)
const springConfig = {
  mass: 0.1,
  stiffness: 260,
  damping: 20,
};

function IconContainer({
  mouseX,
  title,
  icon,
  href,
  isActive,
}: {
  mouseX: ReturnType<typeof useMotionValue<number>>;
  title: string;
  icon: React.ReactNode;
  href: string;
  isActive: boolean;
}) {
  const ref = useRef<HTMLDivElement>(null);

  const distance = useTransform(mouseX, (val: number) => {
    const bounds = ref.current?.getBoundingClientRect() ?? { x: 0, width: 0 };
    return val - bounds.x - bounds.width / 2;
  });

  const widthTransform = useTransform(distance, [-150, 0, 150], [40, 80, 40]);
  const heightTransform = useTransform(distance, [-150, 0, 150], [40, 80, 40]);

  // Spring physics for smooth animations (Requirement 5.4)
  const width = useSpring(widthTransform, springConfig);
  const height = useSpring(heightTransform, springConfig);

  const [hovered, setHovered] = useState(false);

  return (
    <Link href={href}>
      <motion.div
        ref={ref}
        style={{ width, height }}
        onMouseEnter={() => setHovered(true)}
        onMouseLeave={() => setHovered(false)}
        whileTap={{ scale: 0.9 }}
        className={cn(
          'aspect-square rounded-full glass-card flex items-center justify-center relative transition-all duration-300',
          // Slate indicator for active item - no neon glow (Requirement 9.2, 9.3)
          isActive && 'border-slate-400 dark:border-slate-500 bg-slate-100/20 dark:bg-slate-700/50',
          // Subtle slate hover - no glow effects (Requirement 9.3)
          hovered &&
            !isActive &&
            'border-slate-300 dark:border-slate-600 bg-slate-100/10 dark:bg-slate-800/30'
        )}
      >
        {/* Active indicator dot - slate themed (Requirement 9.2) */}
        {isActive && (
          <motion.div
            layoutId="active-indicator"
            className="absolute -bottom-1 w-1.5 h-1.5 rounded-full bg-blue-500"
            transition={springConfig}
          />
        )}
        <AnimatePresence>
          {hovered && (
            <motion.div
              initial={{ opacity: 0, y: 10, x: '-50%' }}
              animate={{ opacity: 1, y: -10, x: '-50%' }}
              exit={{ opacity: 0, y: 2, x: '-50%' }}
              transition={springConfig}
              className="px-3 py-1 whitespace-pre rounded-lg glass-panel text-slate-700 dark:text-slate-200 tracking-wide absolute left-1/2 -top-12 w-fit text-sm font-medium"
            >
              {title}
            </motion.div>
          )}
        </AnimatePresence>
        <motion.div
          style={{ width: widthTransform, height: heightTransform }}
          className={cn(
            'flex items-center justify-center transition-all duration-300 [&_svg]:w-4 [&_svg]:h-4',
            // Slate colors - no glow effects (Requirement 9.2, 9.3)
            isActive ? 'text-blue-500' : 'text-slate-500 dark:text-slate-400',
            hovered && !isActive && 'text-slate-700 dark:text-slate-300'
          )}
        >
          {icon}
        </motion.div>
      </motion.div>
    </Link>
  );
}
