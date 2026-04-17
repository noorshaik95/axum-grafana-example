/**
 * Property-Based Tests for Widget Component
 *
 * These tests verify universal properties that should hold across all inputs
 * using fast-check for property-based testing.
 *
 * **Feature: frontend-styling-standardization, Property 5: Widget Component Structure**
 *
 * @see .kiro/specs/frontend-styling-standardization/design.md
 */

import * as React from 'react'
import { render, screen } from '@testing-library/react'
import * as fc from 'fast-check'
import {
  Widget,
  WidgetHeader,
  WidgetContent,
  WidgetTitle,
  widgetVariants,
  widgetHeaderVariants,
  widgetContentVariants,
} from '../../components/widget'

/**
 * **Feature: frontend-styling-standardization, Property 5: Widget Component Structure**
 *
 * Property 5: Widget Component Structure
 * *For any* Widget component with title and children props, the rendered output
 * SHALL contain a glass-card container, a header section with the title,
 * and a content section with the children.
 *
 * **Validates: Requirements 4.1, 4.2, 4.3**
 */
describe('Property 5: Widget Component Structure', () => {
  /**
   * Test that Widget always renders with title and children.
   * Requirement 4.1: Widget SHALL accept title, icon, and children props
   */
  it('should always render title and children for any valid props', () => {
    fc.assert(
      fc.property(
        // Generate random title strings (non-empty)
        fc.string({ minLength: 1, maxLength: 100 }),
        // Generate random children content
        fc.string({ minLength: 1, maxLength: 200 }),
        (title, childContent) => {
          const { container } = render(
            <Widget title={title}>
              <p data-testid="child-content">{childContent}</p>
            </Widget>
          )

          // Widget must render the title
          expect(container.textContent).toContain(title)

          // Widget must render the children
          expect(container.textContent).toContain(childContent)

          // Clean up
          container.remove()
        }
      ),
      { numRuns: 100 }
    )
  })

  /**
   * Test that Widget always has glass-card container styling.
   * Requirement 4.2: Widget SHALL apply consistent glass card styling
   */
  it('should always have glass-card container with glassmorphism styling', () => {
    fc.assert(
      fc.property(
        fc.string({ minLength: 1, maxLength: 50 }),
        fc.string({ minLength: 1, maxLength: 100 }),
        (title, content) => {
          const { container } = render(
            <Widget title={title}>
              <p>{content}</p>
            </Widget>
          )

          const widget = container.firstChild as HTMLElement
          const className = widget?.className || ''

          // Must have rounded corners (glass card styling)
          expect(className).toContain('rounded-xl')

          // Must have backdrop blur for glassmorphism
          expect(className).toContain('backdrop-blur')

          // Must have border for glass effect
          expect(className).toContain('border')

          // Must have shadow for depth
          expect(className).toContain('shadow-lg')

          // Must use card background color (slate-themed)
          expect(className).toContain('bg-card')

          // Clean up
          container.remove()
        }
      ),
      { numRuns: 100 }
    )
  })

  /**
   * Test that Widget header section renders with title.
   * Requirement 4.3: Widget SHALL render title with consistent typography
   */
  it('should render header section with title for any title string', () => {
    fc.assert(
      fc.property(fc.string({ minLength: 1, maxLength: 100 }), (title) => {
        const { container } = render(
          <Widget title={title}>
            <p>Content</p>
          </Widget>
        )

        // Find the h3 element (title)
        const titleElement = container.querySelector('h3')

        // Title element must exist
        expect(titleElement).not.toBeNull()

        // Title must contain the provided text
        expect(titleElement?.textContent).toBe(title)

        // Title must have proper typography classes
        expect(titleElement?.className).toContain('font-semibold')
        expect(titleElement?.className).toContain('leading-none')
        expect(titleElement?.className).toContain('tracking-tight')

        // Clean up
        container.remove()
      }),
      { numRuns: 100 }
    )
  })

  /**
   * Test that Widget renders optional icon when provided.
   * Requirement 4.1: Widget SHALL accept icon prop
   */
  it('should render icon when provided', () => {
    fc.assert(
      fc.property(
        fc.string({ minLength: 1, maxLength: 50 }),
        fc.string({ minLength: 1, maxLength: 50 }),
        (title, iconText) => {
          const { container } = render(
            <Widget title={title} icon={<span data-testid="icon">{iconText}</span>}>
              <p>Content</p>
            </Widget>
          )

          // Icon must be rendered
          const icon = container.querySelector('[data-testid="icon"]')
          expect(icon).not.toBeNull()
          expect(icon?.textContent).toBe(iconText)

          // Clean up
          container.remove()
        }
      ),
      { numRuns: 100 }
    )
  })

  /**
   * Test that Widget does not render icon container when icon is not provided.
   */
  it('should not render icon when not provided', () => {
    fc.assert(
      fc.property(fc.string({ minLength: 1, maxLength: 50 }), (title) => {
        const { container } = render(
          <Widget title={title}>
            <p>Content</p>
          </Widget>
        )

        // Icon container should not exist when icon is not provided
        const iconContainer = container.querySelector('[data-testid="icon"]')
        expect(iconContainer).toBeNull()

        // Clean up
        container.remove()
      }),
      { numRuns: 100 }
    )
  })
})

/**
 * Test Widget variants never include glow, neon, or aurora classes.
 * This ensures widgets follow the Slate Glass design without flashy effects.
 */
describe('Widget Glassmorphism Styling (No Glow Effects)', () => {
  /**
   * Test that widget variants never include glow, neon, or aurora classes.
   */
  it('should never include glow, neon, or aurora classes in any variant', () => {
    fc.assert(
      fc.property(fc.constantFrom('default', 'sm', 'lg', 'full'), (size) => {
        const classes = widgetVariants({ size: size as any })

        // No glow effects
        expect(classes).not.toContain('glow')
        expect(classes).not.toContain('shadow-glow')
        expect(classes).not.toContain('shadow-neon')

        // No neon effects
        expect(classes).not.toContain('neon')
        expect(classes).not.toContain('neon-glow')
        expect(classes).not.toContain('neon-border')

        // No aurora effects
        expect(classes).not.toContain('aurora')
        expect(classes).not.toContain('gradient-aurora')

        // No shimmer animations
        expect(classes).not.toContain('shimmer')
        expect(classes).not.toContain('animate-shimmer')
      }),
      { numRuns: 100 }
    )
  })

  /**
   * Test that rendered Widget components don't have glow-related classes.
   */
  it('should render Widget without glow classes for any content', () => {
    fc.assert(
      fc.property(
        fc.string({ minLength: 1, maxLength: 50 }),
        fc.string({ minLength: 1, maxLength: 100 }),
        (title, content) => {
          const { container } = render(
            <Widget title={title}>
              <p>{content}</p>
            </Widget>
          )

          const widget = container.firstChild as HTMLElement
          const className = widget?.className || ''

          // No glow effects in rendered output
          expect(className).not.toContain('glow')
          expect(className).not.toContain('neon')
          expect(className).not.toContain('aurora')
          expect(className).not.toContain('shimmer')

          // Clean up
          container.remove()
        }
      ),
      { numRuns: 100 }
    )
  })
})

/**
 * Test Widget header and content sections have responsive padding.
 * Requirements: 4.2, 13.2, 13.3
 */
describe('Widget Responsive Padding', () => {
  /**
   * Test that header variants have responsive padding classes.
   */
  it('should have responsive padding in header variants', () => {
    fc.assert(
      fc.property(fc.constantFrom('default', 'sm', 'lg'), (padding) => {
        const classes = widgetHeaderVariants({ padding: padding as any })

        // Must have base padding
        expect(classes).toMatch(/p-\d/)

        // Must have responsive padding (sm: breakpoint)
        expect(classes).toMatch(/sm:p-\d/)
      }),
      { numRuns: 100 }
    )
  })

  /**
   * Test that content variants have responsive padding classes.
   */
  it('should have responsive padding in content variants (except none)', () => {
    fc.assert(
      fc.property(fc.constantFrom('default', 'sm', 'lg'), (padding) => {
        const classes = widgetContentVariants({ padding: padding as any })

        // Must have base padding
        expect(classes).toMatch(/p-\d/)

        // Must have responsive padding (sm: breakpoint)
        expect(classes).toMatch(/sm:p-\d/)
      }),
      { numRuns: 100 }
    )
  })

  /**
   * Test that content with padding="none" has no padding.
   */
  it('should have no padding when padding is set to none', () => {
    const classes = widgetContentVariants({ padding: 'none' })

    // Should have p-0 for no padding
    expect(classes).toContain('p-0')
  })
})

/**
 * Test Widget sub-components render correctly
 */
describe('Widget Sub-Components', () => {
  /**
   * Test WidgetHeader renders with proper styling.
   */
  it('should render WidgetHeader with border and padding', () => {
    fc.assert(
      fc.property(
        fc.string({ minLength: 1, maxLength: 50 }),
        fc.constantFrom('default', 'sm', 'lg'),
        (content, padding) => {
          const { container } = render(
            <WidgetHeader padding={padding as any}>
              <span>{content}</span>
            </WidgetHeader>
          )

          const header = container.firstChild as HTMLElement
          const className = header?.className || ''

          // Should have flex layout
          expect(className).toContain('flex')
          expect(className).toContain('items-center')

          // Should have border
          expect(className).toContain('border-b')

          // Clean up
          container.remove()
        }
      ),
      { numRuns: 100 }
    )
  })

  /**
   * Test WidgetContent renders with proper padding.
   */
  it('should render WidgetContent with responsive padding', () => {
    fc.assert(
      fc.property(
        fc.string({ minLength: 1, maxLength: 100 }),
        fc.constantFrom('default', 'sm', 'lg'),
        (content, padding) => {
          const { container } = render(
            <WidgetContent padding={padding as any}>
              <p>{content}</p>
            </WidgetContent>
          )

          const contentEl = container.firstChild as HTMLElement
          const className = contentEl?.className || ''

          // Should have padding classes
          expect(className).toMatch(/p-\d/)

          // Clean up
          container.remove()
        }
      ),
      { numRuns: 100 }
    )
  })

  /**
   * Test WidgetTitle renders with proper typography.
   */
  it('should render WidgetTitle with consistent typography', () => {
    fc.assert(
      fc.property(fc.string({ minLength: 1, maxLength: 50 }), (title) => {
        const { container } = render(<WidgetTitle>{title}</WidgetTitle>)

        const titleEl = container.firstChild as HTMLElement
        const className = titleEl?.className || ''

        // Should have proper typography
        expect(className).toContain('font-semibold')
        expect(className).toContain('leading-none')
        expect(className).toContain('tracking-tight')

        // Should render as h3
        expect(titleEl?.tagName).toBe('H3')

        // Clean up
        container.remove()
      }),
      { numRuns: 100 }
    )
  })
})

/**
 * Test Widget maintains identical styling across all configurations.
 * Requirement 4.4: Widget SHALL maintain identical styling and behavior across portals
 */
describe('Widget Consistent Styling', () => {
  /**
   * Test that Widget always has the same base classes regardless of props.
   */
  it('should have consistent base styling for any valid props combination', () => {
    fc.assert(
      fc.property(
        fc.string({ minLength: 1, maxLength: 50 }),
        fc.string({ minLength: 1, maxLength: 100 }),
        fc.constantFrom('default', 'sm', 'lg', 'none'),
        fc.boolean(),
        (title, content, padding, hasIcon) => {
          const { container } = render(
            <Widget
              title={title}
              padding={padding as any}
              icon={hasIcon ? <span>Icon</span> : undefined}
            >
              <p>{content}</p>
            </Widget>
          )

          const widget = container.firstChild as HTMLElement
          const className = widget?.className || ''

          // Base styling must always be present
          expect(className).toContain('rounded-xl')
          expect(className).toContain('backdrop-blur')
          expect(className).toContain('border')
          expect(className).toContain('shadow-lg')
          expect(className).toContain('bg-card')
          expect(className).toContain('text-card-foreground')
          expect(className).toContain('overflow-hidden')

          // Clean up
          container.remove()
        }
      ),
      { numRuns: 100 }
    )
  })
})
