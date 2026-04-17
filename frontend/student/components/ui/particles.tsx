'use client';

import { useMemo } from 'react';

interface ParticlesProps {
  count?: number;
  className?: string;
}

/**
 * Floating particle effect overlay component for the Aurora Glass theme.
 * Creates animated particles that float upward with varying colors and speeds.
 * Respects prefers-reduced-motion for accessibility.
 */
export function Particles({ count = 20, className = '' }: ParticlesProps) {
  // Generate particle positions and delays deterministically
  const particles = useMemo(() => {
    return Array.from({ length: count }, (_, i) => ({
      id: i,
      left: `${(i * 37) % 100}%`, // Distribute across width
      delay: `${(i * 1.3) % 10}s`, // Stagger animations
      size: 2 + (i % 3) * 2, // Vary sizes: 2px, 4px, 6px
    }));
  }, [count]);

  return (
    <div className={`particles-container ${className}`} aria-hidden="true" role="presentation">
      {particles.map((particle) => (
        <span
          key={particle.id}
          className="particle"
          style={{
            left: particle.left,
            animationDelay: particle.delay,
            width: `${particle.size}px`,
            height: `${particle.size}px`,
          }}
        />
      ))}
    </div>
  );
}
