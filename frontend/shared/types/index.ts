// ============================================
// Legacy simple types (User, Course, Assignment, Grade, etc.) have been
// removed — use the canonical definitions from lib/api/types.ts instead,
// which are re-exported through the top-level index.
// ============================================

// ============================================
// Theme Types - Slate Glass Design System
// Requirements: 11.4
// ============================================

/**
 * Theme mode for light/dark theme support
 */
export type ThemeMode = 'light' | 'dark' | 'system'

/**
 * CSS variable-based color token
 */
export type CSSVariableColor = `hsl(var(--${string}))`

/**
 * Color token with optional foreground variant
 */
export interface ColorToken {
  DEFAULT: CSSVariableColor
  foreground?: CSSVariableColor
}

/**
 * Accent color token with hover state
 */
export interface AccentColorToken extends ColorToken {
  hover: CSSVariableColor
}

/**
 * Card color token with border
 */
export interface CardColorToken extends ColorToken {
  border: CSSVariableColor
}

/**
 * Glass effect token
 */
export interface GlassToken {
  bg: string
  border: CSSVariableColor
}

/**
 * Complete Slate Glass color palette type
 */
export interface SlateGlassColorPalette {
  border: CSSVariableColor
  input: CSSVariableColor
  ring: CSSVariableColor
  background: CSSVariableColor
  foreground: CSSVariableColor
  primary: ColorToken
  secondary: ColorToken
  destructive: ColorToken
  muted: ColorToken
  accent: AccentColorToken
  popover: ColorToken
  card: CardColorToken
  glass: GlassToken
}

/**
 * Backdrop blur extension values
 */
export interface BackdropBlurValues {
  xs: string
  '2xl': string
  '3xl': string
}

/**
 * Box shadow extension values (slate-based, no glow)
 */
export interface BoxShadowValues {
  'slate-sm': string
  'slate-md': string
  'slate-lg': string
  'slate-xl': string
  glass: string
  'glass-lg': string
}

/**
 * Animation keyframe definition
 */
export interface KeyframeDefinition {
  [key: string]: { [property: string]: string }
}

/**
 * Theme configuration for components
 */
export interface ThemeConfig {
  mode: ThemeMode
  colors: SlateGlassColorPalette
}

/**
 * Button variant types
 */
export type ButtonVariant = 'primary' | 'secondary' | 'ghost' | 'destructive'

/**
 * Button size types
 */
export type ButtonSize = 'default' | 'sm' | 'lg' | 'icon'

/**
 * Progress component theme colors
 */
export interface ProgressThemeColors {
  track: {
    light: string
    dark: string
  }
  fill: {
    light: string
    dark: string
  }
}

// Component prop types (WidgetProps, StatCardProps, DataTableProps) are
// defined and exported from their respective component files in components/.
