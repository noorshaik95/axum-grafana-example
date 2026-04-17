/**
 * Property-Based Tests for Input Component
 *
 * These tests verify universal properties that should hold across all inputs
 * using fast-check for property-based testing.
 *
 * **Feature: frontend-styling-standardization, Property 7: Focus Indicator Slate Colors**
 *
 * @see .kiro/specs/frontend-styling-standardization/design.md
 */

import * as React from 'react'
import { render } from '@testing-library/react'
import * as fc from 'fast-check'
import { Input, inputVariants } from '../../../components/ui/input'

/**
 * **Feature: frontend-styling-standardization, Property 7: Focus Indicator Slate Colors**
 *
 * Property 7: Focus Indicator Slate Colors
 * *For any* interactive element receiving focus, the focus ring SHALL use
 * slate-500 color without aurora glow effects.
 *
 * **Validates: Requirements 10.2, 14.2**
 */
describe('Property 7: Focus Indicator Slate Colors', () => {
  /**
   * Test that input variants always include focus-visible ring classes
   * using slate-500 (via --ring CSS variable).
   */
  it('should always apply focus-visible ring classes for slate-500 focus', () => {
    fc.assert(
      fc.property(
        // Generate random size variants
        fc.constantFrom('default', 'sm', 'lg'),
        // Generate random error states
        fc.boolean(),
        (size, error) => {
          const classes = inputVariants({ size: size as any, error })

          // Must have focus-visible ring classes (Requirement 10.2, 14.2)
          expect(classes).toContain('focus-visible:ring-2')
          expect(classes).toContain('focus-visible:ring-ring')
          expect(classes).toContain('focus-visible:ring-offset-2')

          // Must have focus-visible outline none (using ring instead)
          expect(classes).toContain('focus-visible:outline-none')
        }
      ),
      { numRuns: 100 }
    )
  })

  /**
   * Test that input variants never include aurora, glow, or neon classes.
   * Focus indicators should use slate-500 ring only.
   */
  it('should never include aurora, glow, or neon classes in focus styles', () => {
    fc.assert(
      fc.property(fc.constantFrom('default', 'sm', 'lg'), fc.boolean(), (size, error) => {
        const classes = inputVariants({ size: size as any, error })

        // No aurora effects (Requirement 10.2)
        expect(classes).not.toContain('aurora')
        expect(classes).not.toContain('gradient-aurora')

        // No glow effects (Requirement 10.2)
        expect(classes).not.toContain('glow')
        expect(classes).not.toContain('shadow-glow')
        expect(classes).not.toContain('shadow-neon')

        // No neon effects (Requirement 10.2)
        expect(classes).not.toContain('neon')
        expect(classes).not.toContain('neon-glow')

        // No shimmer animations
        expect(classes).not.toContain('shimmer')
        expect(classes).not.toContain('animate-shimmer')
      }),
      { numRuns: 100 }
    )
  })

  /**
   * Test that rendered Input component has correct focus ring classes.
   */
  it('should render Input with correct focus ring classes for any configuration', () => {
    fc.assert(
      fc.property(
        // Generate random placeholder text
        fc.string({ minLength: 1, maxLength: 50 }),
        // Generate random size variants
        fc.constantFrom('default', 'sm', 'lg'),
        (placeholder, size) => {
          const { container } = render(<Input placeholder={placeholder} size={size as any} />)

          const input = container.querySelector('input')

          // Must have focus-visible ring classes
          expect(input?.className).toContain('focus-visible:ring-2')
          expect(input?.className).toContain('focus-visible:ring-ring')

          // Must NOT have aurora/glow classes
          expect(input?.className).not.toContain('aurora')
          expect(input?.className).not.toContain('glow')
          expect(input?.className).not.toContain('neon')

          // Clean up
          container.remove()
        }
      ),
      { numRuns: 100 }
    )
  })

  /**
   * Test that focus ring offset uses background color for proper visibility.
   */
  it('should have focus ring offset using background color', () => {
    fc.assert(
      fc.property(fc.constantFrom('default', 'sm', 'lg'), (size) => {
        const classes = inputVariants({ size: size as any })

        // Must have ring offset background for proper visibility
        expect(classes).toContain('focus-visible:ring-offset-background')
      }),
      { numRuns: 100 }
    )
  })
})

/**
 * Additional property tests for Input slate-colored styling
 * Ensuring proper slate-themed borders and backgrounds (Requirement 10.1)
 */
describe('Input Slate-Colored Styling', () => {
  /**
   * Test that inputs always have slate-colored border and background.
   */
  it('should always include slate-colored border and background classes', () => {
    fc.assert(
      fc.property(fc.constantFrom('default', 'sm', 'lg'), fc.boolean(), (size, error) => {
        const classes = inputVariants({ size: size as any, error })

        // Must have background color (Requirement 10.1)
        expect(classes).toContain('bg-background')

        // Must have border using input color (slate-themed)
        expect(classes).toContain('border')
        expect(classes).toContain('border-input')

        // Must have foreground text color
        expect(classes).toContain('text-foreground')
      }),
      { numRuns: 100 }
    )
  })

  /**
   * Test that inputs have muted placeholder styling.
   */
  it('should have muted placeholder text color', () => {
    fc.assert(
      fc.property(fc.constantFrom('default', 'sm', 'lg'), (size) => {
        const classes = inputVariants({ size: size as any })

        // Must have muted placeholder color
        expect(classes).toContain('placeholder:text-muted-foreground')
      }),
      { numRuns: 100 }
    )
  })

  /**
   * Test that inputs have smooth transitions.
   */
  it('should have transition classes for smooth state changes', () => {
    fc.assert(
      fc.property(fc.constantFrom('default', 'sm', 'lg'), (size) => {
        const classes = inputVariants({ size: size as any })

        // Must have transition for smooth effects
        expect(classes).toContain('transition-colors')
        expect(classes).toContain('duration-200')
      }),
      { numRuns: 100 }
    )
  })
})

/**
 * Property tests for Input error state (Requirement 10.3)
 */
describe('Input Error State', () => {
  /**
   * Test that error state applies red border while maintaining slate styling.
   */
  it('should apply red border for error state', () => {
    fc.assert(
      fc.property(fc.constantFrom('default', 'sm', 'lg'), (size) => {
        const classes = inputVariants({ size: size as any, error: true })

        // Error state must have red border (Requirement 10.3)
        expect(classes).toContain('border-red-500')

        // Dark mode should have appropriate red border
        expect(classes).toContain('dark:border-red-400')
      }),
      { numRuns: 100 }
    )
  })

  /**
   * Test that error state changes focus ring to red.
   */
  it('should apply red focus ring for error state', () => {
    fc.assert(
      fc.property(fc.constantFrom('default', 'sm', 'lg'), (size) => {
        const classes = inputVariants({ size: size as any, error: true })

        // Error state must have red focus ring
        expect(classes).toContain('focus-visible:ring-red-500')
        expect(classes).toContain('dark:focus-visible:ring-red-400')
      }),
      { numRuns: 100 }
    )
  })

  /**
   * Test that non-error state does NOT have red border.
   */
  it('should NOT apply red border for non-error state', () => {
    fc.assert(
      fc.property(fc.constantFrom('default', 'sm', 'lg'), (size) => {
        const classes = inputVariants({ size: size as any, error: false })

        // Non-error state should NOT have red border
        expect(classes).not.toContain('border-red-500')
        expect(classes).not.toContain('border-red-400')
      }),
      { numRuns: 100 }
    )
  })

  /**
   * Test that rendered Input with error shows error message.
   */
  it('should render error message when provided', () => {
    fc.assert(
      fc.property(fc.string({ minLength: 1, maxLength: 100 }), (errorMessage) => {
        const { container } = render(<Input error errorMessage={errorMessage} />)

        // Error message should be displayed
        expect(container.textContent).toContain(errorMessage)

        // Error message should have role="alert" for accessibility
        const errorEl = container.querySelector('[role="alert"]')
        expect(errorEl).not.toBeNull()
        expect(errorEl?.textContent).toBe(errorMessage)

        // Clean up
        container.remove()
      }),
      { numRuns: 100 }
    )
  })

  /**
   * Test that rendered Input with error has aria-invalid attribute.
   */
  it('should have aria-invalid attribute when in error state', () => {
    fc.assert(
      fc.property(fc.string({ minLength: 1, maxLength: 50 }), (errorMessage) => {
        const { container } = render(<Input error errorMessage={errorMessage} />)

        const input = container.querySelector('input')

        // Should have aria-invalid for accessibility
        expect(input?.getAttribute('aria-invalid')).toBe('true')

        // Clean up
        container.remove()
      }),
      { numRuns: 100 }
    )
  })
})

/**
 * Property tests for Input label and helper text
 */
describe('Input Label and Helper Text', () => {
  /**
   * Test that label is rendered when provided.
   */
  it('should render label when provided', () => {
    fc.assert(
      fc.property(fc.string({ minLength: 1, maxLength: 50 }), (label) => {
        const { container } = render(<Input label={label} />)

        // Label should be displayed
        const labelEl = container.querySelector('label')
        expect(labelEl).not.toBeNull()
        expect(labelEl?.textContent).toBe(label)

        // Clean up
        container.remove()
      }),
      { numRuns: 100 }
    )
  })

  /**
   * Test that helper text is rendered when provided (and no error).
   */
  it('should render helper text when provided and no error', () => {
    fc.assert(
      fc.property(fc.string({ minLength: 1, maxLength: 100 }), (helperText) => {
        const { container } = render(<Input helperText={helperText} />)

        // Helper text should be displayed
        expect(container.textContent).toContain(helperText)

        // Clean up
        container.remove()
      }),
      { numRuns: 100 }
    )
  })

  /**
   * Test that error message takes precedence over helper text.
   */
  it('should show error message instead of helper text when both provided', () => {
    fc.assert(
      fc.property(
        fc.string({ minLength: 1, maxLength: 50 }),
        fc.string({ minLength: 1, maxLength: 50 }),
        (errorMessage, helperText) => {
          // Ensure they're different to properly test precedence
          const uniqueHelper = `helper-${helperText}`
          const uniqueError = `error-${errorMessage}`

          const { container } = render(
            <Input error errorMessage={uniqueError} helperText={uniqueHelper} />
          )

          // Error message should be displayed
          expect(container.textContent).toContain(uniqueError)

          // Helper text should NOT be displayed when there's an error
          expect(container.textContent).not.toContain(uniqueHelper)

          // Clean up
          container.remove()
        }
      ),
      { numRuns: 100 }
    )
  })
})

/**
 * Property tests for Input disabled state
 */
describe('Input Disabled State', () => {
  /**
   * Test that disabled inputs have correct styling classes.
   */
  it('should have disabled styling classes', () => {
    fc.assert(
      fc.property(fc.constantFrom('default', 'sm', 'lg'), (size) => {
        const classes = inputVariants({ size: size as any })

        // Must have disabled cursor and opacity
        expect(classes).toContain('disabled:cursor-not-allowed')
        expect(classes).toContain('disabled:opacity-50')
        expect(classes).toContain('disabled:bg-muted')
      }),
      { numRuns: 100 }
    )
  })
})
