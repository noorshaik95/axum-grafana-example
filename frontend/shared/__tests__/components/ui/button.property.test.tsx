/**
 * Property-Based Tests for Button Component
 *
 * These tests verify universal properties that should hold across all inputs
 * using fast-check for property-based testing.
 *
 * **Feature: frontend-styling-standardization, Property 3: Primary Button Blue Accent**
 *
 * @see .kiro/specs/frontend-styling-standardization/design.md
 */

import * as React from 'react'
import { render } from '@testing-library/react'
import * as fc from 'fast-check'
import { Button, buttonVariants } from '../../../components/ui/button'

/**
 * **Feature: frontend-styling-standardization, Property 3: Primary Button Blue Accent**
 *
 * Property 3: Primary Button Blue Accent
 * *For any* primary button component, the background color SHALL be blue-500 (#3b82f6)
 * and the text color SHALL be white, making it the only colored interactive element.
 *
 * **Validates: Requirements 1.4, 8.1**
 */
describe('Property 3: Primary Button Blue Accent', () => {
  /**
   * Test that primary variant always includes the correct CSS classes
   * for blue-500 background and white text.
   *
   * This property test generates random button content and verifies
   * that the primary variant always applies the correct styling classes.
   */
  it('should always apply bg-primary and text-primary-foreground classes for primary variant', () => {
    fc.assert(
      fc.property(
        // Generate random button text content
        fc.string({ minLength: 1, maxLength: 100 }),
        // Generate random size variants
        fc.constantFrom('default', 'sm', 'lg', 'icon'),
        (buttonText, size) => {
          const { container } = render(
            <Button variant="primary" size={size as any}>
              {buttonText}
            </Button>
          )

          const button = container.querySelector('button')

          // Primary button must have bg-primary class (maps to blue-500)
          expect(button?.className).toContain('bg-primary')

          // Primary button must have text-primary-foreground class (maps to white)
          expect(button?.className).toContain('text-primary-foreground')

          // Clean up
          container.remove()
        }
      ),
      { numRuns: 100 }
    )
  })

  /**
   * Test that the buttonVariants function generates correct classes
   * for the primary variant across all size combinations.
   */
  it('should generate correct variant classes for primary buttons', () => {
    fc.assert(
      fc.property(fc.constantFrom('default', 'sm', 'lg', 'icon'), (size) => {
        const classes = buttonVariants({ variant: 'primary', size: size as any })

        // Must include primary background color class
        expect(classes).toContain('bg-primary')

        // Must include primary foreground (white text) class
        expect(classes).toContain('text-primary-foreground')

        // Must NOT include any aurora or glow classes (Requirement 8.5)
        expect(classes).not.toContain('aurora')
        expect(classes).not.toContain('glow')
        expect(classes).not.toContain('neon')
        expect(classes).not.toContain('shimmer')
      }),
      { numRuns: 100 }
    )
  })

  /**
   * Test that primary buttons have hover state that shifts to blue-600
   * without any glow effects.
   */
  it('should have blue-600 hover state without glow effects', () => {
    fc.assert(
      fc.property(fc.string({ minLength: 1, maxLength: 50 }), (buttonText) => {
        const classes = buttonVariants({ variant: 'primary' })

        // Must include hover state for darker blue (blue-600)
        // The hover class uses hsl(217,91%,50%) which is blue-600
        expect(classes).toContain('hover:bg-[hsl(217,91%,50%)]')

        // Must NOT include any glow or shadow-neon effects on hover
        expect(classes).not.toContain('shadow-neon')
        expect(classes).not.toContain('hover:shadow-neon')
        expect(classes).not.toContain('glow')
      }),
      { numRuns: 100 }
    )
  })

  /**
   * Test that primary is the default variant when no variant is specified.
   */
  it('should default to primary variant when no variant specified', () => {
    fc.assert(
      fc.property(fc.string({ minLength: 1, maxLength: 50 }), (buttonText) => {
        const { container } = render(<Button>{buttonText}</Button>)

        const button = container.querySelector('button')

        // Default variant should be primary with blue background
        expect(button?.className).toContain('bg-primary')
        expect(button?.className).toContain('text-primary-foreground')

        container.remove()
      }),
      { numRuns: 100 }
    )
  })

  /**
   * Test that non-primary variants do NOT have the blue-500 background.
   * This ensures blue is reserved for primary interactive elements only.
   */
  it('should NOT apply bg-primary to non-primary variants', () => {
    fc.assert(
      fc.property(
        fc.constantFrom('secondary', 'ghost', 'destructive', 'outline', 'link'),
        fc.string({ minLength: 1, maxLength: 50 }),
        (variant, buttonText) => {
          const classes = buttonVariants({ variant: variant as any })

          // Non-primary variants should NOT have bg-primary
          // (blue accent is reserved for primary only per Requirement 1.4)
          expect(classes).not.toContain('bg-primary')
        }
      ),
      { numRuns: 100 }
    )
  })
})

/**
 * Additional property tests for button variants
 * Ensuring no aurora/glow effects across all variants (Requirement 8.5)
 */
describe('Property: No Aurora/Glow Effects on Any Button Variant', () => {
  it('should never include aurora, glow, or neon classes in any variant', () => {
    fc.assert(
      fc.property(
        fc.constantFrom('primary', 'secondary', 'ghost', 'destructive', 'outline', 'link'),
        fc.constantFrom('default', 'sm', 'lg', 'icon'),
        (variant, size) => {
          const classes = buttonVariants({ variant: variant as any, size: size as any })

          // No aurora effects (Requirement 8.5)
          expect(classes).not.toContain('aurora')
          expect(classes).not.toContain('gradient-aurora')

          // No glow effects (Requirement 8.5)
          expect(classes).not.toContain('glow')
          expect(classes).not.toContain('shadow-neon')
          expect(classes).not.toContain('neon')

          // No shimmer animations (Requirement 8.5)
          expect(classes).not.toContain('shimmer')
          expect(classes).not.toContain('animate-shimmer')
        }
      ),
      { numRuns: 100 }
    )
  })
})
