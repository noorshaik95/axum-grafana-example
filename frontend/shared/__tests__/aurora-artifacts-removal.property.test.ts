/**
 * Property-Based Tests for Aurora Artifacts Removal
 *
 * These tests verify that aurora theme artifacts have been completely removed
 * from the Student Portal CSS files.
 *
 * **Feature: frontend-styling-standardization, Property 1: Aurora Artifacts Removal**
 *
 * @see .kiro/specs/frontend-styling-standardization/design.md
 */

import * as fs from 'fs'
import * as path from 'path'
import * as fc from 'fast-check'

/**
 * **Feature: frontend-styling-standardization, Property 1: Aurora Artifacts Removal**
 *
 * Property 1: Aurora Artifacts Removal
 * *For any* CSS file in the frontend directories, the file SHALL NOT contain
 * aurora color variables (--aurora-cyan, --aurora-violet, --aurora-magenta, --aurora-pink),
 * aurora utility classes (.aurora-bg, .neon-glow-*, .neon-border),
 * or aurora keyframes (aurora-flow, glow-pulse, shimmer-glow).
 *
 * **Validates: Requirements 12.1, 12.2, 12.3, 12.4**
 */
describe('Property 1: Aurora Artifacts Removal', () => {
  // Path to the Student Portal globals.css
  // __dirname is frontend/shared/__tests__, so we need to go up to frontend/shared
  // then navigate to ../student/app/globals.css
  const studentGlobalsCssPath = path.resolve(__dirname, '..', '..', 'student/app/globals.css')

  // Read the CSS file content once for all tests
  let cssContent: string

  beforeAll(() => {
    cssContent = fs.readFileSync(studentGlobalsCssPath, 'utf-8')
  })

  /**
   * Test that aurora CSS variables have been removed (Requirement 12.1)
   *
   * Aurora variables that must NOT exist as declarations:
   * - --aurora-cyan
   * - --aurora-violet
   * - --aurora-magenta
   * - --aurora-pink
   * - --glow-cyan
   * - --glow-violet
   * - --glow-magenta
   *
   * Note: We check for actual CSS variable declarations (with colon),
   * not just the string presence (which may appear in comments).
   */
  describe('Requirement 12.1: Aurora CSS Variables Removal', () => {
    const auroraVariables = [
      '--aurora-cyan',
      '--aurora-violet',
      '--aurora-magenta',
      '--aurora-pink',
      '--glow-cyan',
      '--glow-violet',
      '--glow-magenta',
    ]

    it('should not contain any aurora color variable declarations', () => {
      fc.assert(
        fc.property(fc.constantFrom(...auroraVariables), (variable) => {
          // Check for actual variable declarations (variable: value)
          // This excludes comments that mention the variable names
          const declarationPattern = new RegExp(`${variable}\\s*:`, 'g')
          expect(cssContent).not.toMatch(declarationPattern)
        }),
        { numRuns: auroraVariables.length }
      )
    })

    it('should not contain aurora variable declarations', () => {
      // Check for variable declarations like "--aurora-cyan: 186 100% 50%"
      const auroraVarDeclarationPattern = /--aurora-\w+\s*:/g
      const matches = cssContent.match(auroraVarDeclarationPattern)
      expect(matches).toBeNull()
    })

    it('should not contain glow variable declarations', () => {
      // Check for glow variable declarations
      const glowVarDeclarationPattern = /--glow-\w+\s*:/g
      const matches = cssContent.match(glowVarDeclarationPattern)
      expect(matches).toBeNull()
    })
  })

  /**
   * Test that aurora utility classes have been removed (Requirement 12.2)
   *
   * Aurora classes that must NOT exist:
   * - .aurora-bg
   * - .neon-glow-*
   * - .neon-border
   * - .gradient-aurora-text
   * - .text-aurora-*
   */
  describe('Requirement 12.2: Aurora Utility Classes Removal', () => {
    const auroraClasses = [
      '.aurora-bg',
      '.neon-glow-cyan',
      '.neon-glow-violet',
      '.neon-glow-magenta',
      '.neon-border',
      '.gradient-aurora-text',
      '.text-aurora-cyan',
      '.text-aurora-violet',
      '.text-aurora-magenta',
      '.text-aurora-pink',
      '.text-glow',
      '.text-glow-violet',
      '.text-glow-magenta',
    ]

    it('should not contain any aurora utility classes', () => {
      fc.assert(
        fc.property(fc.constantFrom(...auroraClasses), (className) => {
          // The CSS file should NOT contain any aurora utility classes
          // We check for the class definition (e.g., ".aurora-bg {")
          const classPattern = new RegExp(`\\${className}\\s*\\{`, 'g')
          expect(cssContent).not.toMatch(classPattern)
        }),
        { numRuns: auroraClasses.length }
      )
    })

    it('should not contain neon-glow class definitions', () => {
      // Check for any .neon-glow-* class definitions
      const neonGlowPattern = /\.neon-glow-\w+\s*\{/g
      const matches = cssContent.match(neonGlowPattern)
      expect(matches).toBeNull()
    })

    it('should not contain aurora-bg class definition', () => {
      // Check for .aurora-bg class definition
      const auroraBgPattern = /\.aurora-bg\s*\{/g
      const matches = cssContent.match(auroraBgPattern)
      expect(matches).toBeNull()
    })

    it('should not contain gradient-aurora-text class definition', () => {
      // Check for .gradient-aurora-text class definition
      const gradientAuroraPattern = /\.gradient-aurora-text\s*\{/g
      const matches = cssContent.match(gradientAuroraPattern)
      expect(matches).toBeNull()
    })
  })

  /**
   * Test that aurora animations have been removed (Requirement 12.3)
   *
   * Aurora keyframes that must NOT exist:
   * - aurora-flow
   * - glow-pulse
   * - shimmer-glow
   * - neon-border-flow
   */
  describe('Requirement 12.3: Aurora Animations Removal', () => {
    const auroraKeyframes = [
      'aurora-flow',
      'glow-pulse',
      'shimmer-glow',
      'neon-border-flow',
      'particle-float',
    ]

    it('should not contain any aurora keyframe animations', () => {
      fc.assert(
        fc.property(fc.constantFrom(...auroraKeyframes), (keyframeName) => {
          // Check for @keyframes definitions
          const keyframePattern = new RegExp(`@keyframes\\s+${keyframeName}\\s*\\{`, 'g')
          expect(cssContent).not.toMatch(keyframePattern)
        }),
        { numRuns: auroraKeyframes.length }
      )
    })

    it('should not contain aurora-flow keyframe', () => {
      expect(cssContent).not.toMatch(/@keyframes\s+aurora-flow/)
    })

    it('should not contain glow-pulse keyframe', () => {
      expect(cssContent).not.toMatch(/@keyframes\s+glow-pulse/)
    })

    it('should not contain shimmer-glow keyframe', () => {
      expect(cssContent).not.toMatch(/@keyframes\s+shimmer-glow/)
    })
  })

  /**
   * Test that particle effects have been removed (Requirement 12.4)
   *
   * Particle classes that must NOT exist:
   * - .particles-container
   * - .particle
   */
  describe('Requirement 12.4: Particle Effects Removal', () => {
    const particleClasses = ['.particles-container', '.particle']

    it('should not contain any particle effect classes', () => {
      fc.assert(
        fc.property(fc.constantFrom(...particleClasses), (className) => {
          // Check for particle class definitions
          const classPattern = new RegExp(`\\${className}\\s*\\{`, 'g')
          expect(cssContent).not.toMatch(classPattern)
        }),
        { numRuns: particleClasses.length }
      )
    })

    it('should not contain particles-container class', () => {
      expect(cssContent).not.toMatch(/\.particles-container\s*\{/)
    })

    it('should not contain particle class', () => {
      // Check for .particle class but not .particle: (pseudo-selector)
      expect(cssContent).not.toMatch(/\.particle\s*\{/)
    })

    it('should not contain particle-float keyframe', () => {
      expect(cssContent).not.toMatch(/@keyframes\s+particle-float/)
    })
  })

  /**
   * Test that aurora color references in HSL format have been removed
   */
  describe('Aurora HSL Color References Removal', () => {
    it('should not contain hsl references to aurora variables', () => {
      // Check for hsl(var(--aurora-*)) patterns
      const auroraHslPattern = /hsl\s*\(\s*var\s*\(\s*--aurora-\w+/g
      const matches = cssContent.match(auroraHslPattern)
      expect(matches).toBeNull()
    })

    it('should not contain hsl references to glow variables', () => {
      // Check for hsl(var(--glow-*)) patterns
      const glowHslPattern = /hsl\s*\(\s*var\s*\(\s*--glow-\w+/g
      const matches = cssContent.match(glowHslPattern)
      expect(matches).toBeNull()
    })
  })

  /**
   * Comprehensive property test: For any aurora-related CSS declaration pattern,
   * the CSS file should not contain it.
   *
   * Note: We check for actual CSS declarations and definitions,
   * not just string presence (which may appear in documentation comments).
   */
  describe('Comprehensive Aurora Artifacts Check', () => {
    // Patterns that indicate actual CSS declarations/definitions (not comments)
    const auroraDeclarationPatterns = [
      // Variable declarations (variable: value)
      { pattern: /--aurora-\w+\s*:/g, name: 'aurora variable declaration' },
      { pattern: /--glow-\w+\s*:/g, name: 'glow variable declaration' },
      // Class definitions (.class {)
      { pattern: /\.aurora-bg\s*\{/g, name: '.aurora-bg class' },
      { pattern: /\.neon-glow-\w+\s*\{/g, name: '.neon-glow-* class' },
      { pattern: /\.neon-border\s*\{/g, name: '.neon-border class' },
      { pattern: /\.gradient-aurora-text\s*\{/g, name: '.gradient-aurora-text class' },
      { pattern: /\.text-aurora-\w+\s*\{/g, name: '.text-aurora-* class' },
      { pattern: /\.text-glow\s*\{/g, name: '.text-glow class' },
      { pattern: /\.text-glow-\w+\s*\{/g, name: '.text-glow-* class' },
      { pattern: /\.glow-pulse\s*\{/g, name: '.glow-pulse class' },
      { pattern: /\.shimmer-glow\s*\{/g, name: '.shimmer-glow class' },
      { pattern: /\.particles-container\s*\{/g, name: '.particles-container class' },
      { pattern: /\.particle\s*\{/g, name: '.particle class' },
      // Keyframe definitions
      { pattern: /@keyframes\s+aurora-flow/g, name: 'aurora-flow keyframe' },
      { pattern: /@keyframes\s+glow-pulse/g, name: 'glow-pulse keyframe' },
      { pattern: /@keyframes\s+shimmer-glow/g, name: 'shimmer-glow keyframe' },
      { pattern: /@keyframes\s+particle-float/g, name: 'particle-float keyframe' },
      { pattern: /@keyframes\s+neon-border-flow/g, name: 'neon-border-flow keyframe' },
    ]

    it('should not contain any aurora-related CSS declarations', () => {
      fc.assert(
        fc.property(fc.constantFrom(...auroraDeclarationPatterns), ({ pattern, name }) => {
          const matches = cssContent.match(pattern)
          expect(matches).toBeNull()
        }),
        { numRuns: auroraDeclarationPatterns.length }
      )
    })

    it('should not use hsl(var(--aurora-*)) in any property value', () => {
      // Check for hsl(var(--aurora-*)) patterns which indicate actual usage
      const auroraHslUsagePattern = /hsl\s*\(\s*var\s*\(\s*--aurora-\w+/g
      const matches = cssContent.match(auroraHslUsagePattern)
      expect(matches).toBeNull()
    })

    it('should not use hsl(var(--glow-*)) in any property value', () => {
      // Check for hsl(var(--glow-*)) patterns which indicate actual usage
      const glowHslUsagePattern = /hsl\s*\(\s*var\s*\(\s*--glow-\w+/g
      const matches = cssContent.match(glowHslUsagePattern)
      expect(matches).toBeNull()
    })
  })
})
