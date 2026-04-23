# Slate Frontend Design System Gap Audit

**Date:** 2026-04-18  
**Reference:** docs/DESIGN.md

---

## 1. Token Gaps (Tailwind Preset & CSS Variables)

### Critical Mismatches

**Shared/globals.css and tailwind.preset.ts:**

- Missing forest color tokens (preset doesn't define forest-50 through forest-900)
- Missing cream, paper, warm-\* semantic tokens
- Missing design system radii (card=2xl, button=xl, input=lg, badge=full)
- Missing font families (--font-display, --font-ui, --font-mono not set as CSS variables)
- Tailwind preset extends only `hsl()` variables and legacy shadcn palette; no forest/cream tokens
- No `rounded-2xl`, `rounded-xl`, `rounded-full` radius customization pointing to design system values

**Per-Portal globals.css:**

- **Admin:** Defines custom forest/cream palette, but references indigo-600 (#6366f1) as primary in components—violates DESIGN.md §2 (forest-600 = #2e7d2e required)
- **Provider:** Similar issue; forest tokens defined but components use indigo/slate
- **Student:** Correctly defines forest/cream palette and custom radius classes; uses them consistently

### Token Values Defined Locally (Correct):

- Forest palette (50–900): All three portals + shared have matching hex values ✓
- Cream/paper/warm tokens: Present in all globals.css files ✓
- Semantic (amber-400, red-500, blue-500): Present ✓

### Token Values NOT in Tailwind Preset:

- `warm-300`, `warm-600`, `warm-700`, `warm-900` (only 50, 100, 200 in CSS, missing from preset)
- Radius tokens not exposed as Tailwind utilities (e.g., `rounded-card` = 2xl not aliased)
- Font families not exposed as Tailwind utilities (must hardcode 'Instrument Serif')

---

## 2. Missing Shared Components

**DESIGN.md specifies these required components; audit status:**

| Component      | Location                             | Status                                                                                |
| -------------- | ------------------------------------ | ------------------------------------------------------------------------------------- |
| ActionHero     | —                                    | **MISSING** — No component for "Good morning, Alice" + primary CTA hero pattern       |
| StatCard       | shared/components/stat-card.tsx      | Present but incomplete (no trend indicator styles)                                    |
| ListRows       | —                                    | **MISSING** — Alternating warm-50/cream rows with left icon+label, right badge+action |
| Cards          | shared/components/ui/card.tsx        | Present (basic Card, no hover shadow/tint behavior)                                   |
| Badges         | shared/components/ui/badge.tsx       | Present but missing green/amber/red/gray/blue variants from DESIGN.md §4              |
| Buttons        | shared/components/ui/button.tsx      | Present but uses var(--color-primary) not forest tokens; no amber variant             |
| TopBar         | —                                    | **MISSING** — No sticky blurred cream topbar with wordmark, nav, search/bell/avatar   |
| NowBar         | —                                    | **MISSING** — No forest-700 contextual action bar with dismissible chips              |
| CommandPalette | —                                    | **MISSING** — No ⌘K command palette (search courses/students/pages)                   |
| StatusPills    | —                                    | **MISSING** — No "Schools 42" count pill component                                    |
| EmptyState     | shared/components/ui/empty-state.tsx | Present but not referenced in DESIGN.md; unclear if matches spec                      |
| PageHeader     | shared/components/ui/page-header.tsx | Present; uses breadcrumbs/title/description pattern                                   |

**Component Export Status:**

- shared/components/ui/index.ts exports: Button, Card, Badge, Avatar, Spinner, EmptyState, PageHeader, StatCard, Input, Progress, SidebarNav
- Missing from export: ActionHero, ListRows, TopBar, NowBar, CommandPalette, StatusPills

---

## 3. Duplicated Components to Consolidate

Each portal has reimplemented UI primitives instead of using shared versions:

### Admin Portal

- **Button** (`admin/components/ui/button.tsx`): Uses `indigo-600`, different focus ring color
- **Badge** (`admin/components/ui/badge.tsx`): Variants (success/warning/info) but missing forest-based palette
- **Card** (`admin/components/ui/card.tsx`): Duplicate of shared with minor styling differences
- **Input, Progress, Dropdown, Label, Select, Tabs, Toast, Dialog, etc.**: All custom implementations

### Provider Portal

- **Button** (`provider/components/ui/button.tsx`): Uses `indigo-600`, includes custom sidebar variant
- **Badge** (`provider/components/ui/badge.tsx`): Similar to admin; yellow/blue variants instead of forest
- **Card, Input, Progress, Dialog, Dropdown, Textarea, Tabs, etc.**: All duplicated

### Student Portal

- **Button** (`student/components/ui/button.tsx`): Uses `blue-500` primary, includes deprecated aurora/glow variants
- **Badge** (`student/components/ui/badge.tsx`): Simpler variant set; forest not consistently used
- **Card, Input, Progress, Dropdown, Toast, etc.**: Duplicated

**Consolidation Impact:**

- 9+ UI primitives × 3 portals = 27 duplicate files to merge into shared
- Each portal's custom components define styles locally (indigo, blue, slate) instead of inheriting forest tokens
- Maintenance burden: color/radius changes require updates across 9 locations

---

## 4. Navigation Migration Scope

### Sidebar Usage (DESIGN.md §3 specifies TopBar + NowBar pattern, NOT sidebar)

**Files with `<aside>` or Sidebar component:**

| Portal       | File                                     | Status                          | Change Required                        |
| ------------ | ---------------------------------------- | ------------------------------- | -------------------------------------- |
| **Admin**    | `components/layout/admin-shell.tsx`      | Uses sidebar                    | Migrate to TopBar + contextual Now Bar |
| **Admin**    | `components/layout/sidebar.tsx`          | Sidebar component               | Delete; move nav to TopBar             |
| **Admin**    | `components/layout/dashboard-layout.tsx` | Wraps sidebar                   | Refactor to flex/grid TopBar layout    |
| **Provider** | `components/layout/instructor-shell.tsx` | Uses sidebar                    | Migrate to TopBar + Now Bar            |
| **Provider** | `components/layout/sidebar.tsx`          | Sidebar component               | Delete; nav to TopBar                  |
| **Provider** | `components/layout/dashboard-layout.tsx` | Wraps sidebar                   | Refactor to TopBar layout              |
| **Student**  | `components/layout/student-shell.tsx`    | Uses sidebar                    | Migrate to TopBar + contextual Now Bar |
| **Student**  | `components/layout/sidebar.tsx`          | Sidebar component (collapsible) | Delete; nav to TopBar                  |
| **Student**  | `app/courses/[id]/page.tsx`              | References sidebar indirectly   | Update route layout                    |
| **Shared**   | `components/ui/sidebar-nav.tsx`          | Helper component                | Deprecate (no longer needed)           |

**Sidebar Properties in globals.css (to remove):**

- `--color-sidebar`, `--color-sidebar-hover`, `--color-sidebar-active` (defined in all three portal globals.css)

---

## 5. Hardcoded Colors/Fonts Violating DESIGN.md Tokens

### Critical Violations (Sample)

**Admin Button:**

```tsx
// admin/components/ui/button.tsx:12
'bg-indigo-600 text-white shadow-sm hover:bg-indigo-700'
// VIOLATION: indigo-600 !== forest-600 (#2e7d2e)
// FIX: Use var(--color-primary) = forest-600
```

**Provider Sidebar:**

```tsx
// provider/components/layout/sidebar.tsx:41
<aside className="flex flex-col w-64 min-h-screen bg-[#0f172a] text-white">
// VIOLATION: #0f172a is slate, not forest-900 (#0d280d)
// FIX: Use bg-forest-900 or var(--color-sidebar) = forest-700
```

**Student Button:**

```tsx
// student/components/ui/button.tsx:60, 65
'bg-blue-500 text-white' / 'hover:bg-blue-600'
// VIOLATION: blue-500 !== forest-600; uses different primary color than spec
// FIX: Use forest-600 for primary, forest-700 for hover
```

**Shared Card (Correct):**

```tsx
// shared/components/ui/card.tsx:8
className={cn('rounded-xl border border-[var(--color-border)] bg-white shadow-sm', className)}
// PARTIALLY CORRECT: Uses var(--color-border) but bg-white is hardcoded
// FIX: Use bg-[var(--color-bg)] = cream, border should be warm-100 per DESIGN.md §4
```

### Font Violations

**Student globals.css (Correct):**

```css
.serif {
  font-family: 'Instrument Serif', Georgia, serif;
}
.mono {
  font-family: 'JetBrains Mono', monospace;
}
/* Applied via class name, not CSS variable */
```

**Admin/Provider globals.css (Missing):**

- No utility classes for serif/mono fonts; components hardcode font names
- Admin header: `font-semibold` without font-family override

**Missing CSS Variable:**

- `--font-display: 'Instrument Serif'` should be in :root (DESIGN.md §7)

---

## 6. Portal App Integration Issues

### Does each portal use the shared Tailwind preset?

- **Admin:** ✓ `tailwind.config.ts` imports `slatePreset`
- **Provider:** ✓ `tailwind.config.ts` imports `slatePreset`
- **Student:** ✓ `tailwind.config.ts` imports `slatePreset`

### Does globals.css import shared tokens?

- **Admin, Provider, Student:** No imports; each defines own :root tokens
- **Shared:** Defines tokens in globals.css, NOT re-exported for portal consumption

**Improvement:** Move token definitions to shared globals.css and import into each portal via `@import '../shared/styles/globals.css'`

---

## 7. Summary Table

| Category                             | Status   | Count                                                                        | Priority |
| ------------------------------------ | -------- | ---------------------------------------------------------------------------- | -------- |
| **Missing Design System Components** | CRITICAL | ActionHero, ListRows, TopBar, NowBar, CommandPalette, StatusPills            | P0       |
| **Token Mismatches**                 | HIGH     | Indigo/blue/slate used instead of forest; missing radii/fonts as utilities   | P0       |
| **Duplicated UI Components**         | HIGH     | 27 files (9 primitives × 3 portals)                                          | P1       |
| **Sidebar → TopBar Migration**       | HIGH     | 9 files across 3 portals + shared                                            | P1       |
| **Hardcoded Colors**                 | MEDIUM   | Admin button (indigo), Provider sidebar (#0f172a), Student button (blue-500) | P1       |
| **Font Stack**                       | MEDIUM   | Missing --font-display variable; manual font-family in components            | P2       |

---

## 8. Recommended Action Plan

### Phase 1 (P0 - Design Compliance)

1. Add forest/cream/warm tokens to shared/tailwind.preset.ts as Tailwind color utilities
2. Create ActionHero, ListRows, TopBar, NowBar components in shared/components
3. Fix shared Button/Badge/Card to use forest tokens + correct radii

### Phase 2 (P1 - Consolidation)

4. Delete duplicated Button/Badge/Card/Input from admin, provider, student; use shared imports
5. Migrate all three portals from sidebar → TopBar + contextual NowBar layout
6. Remove sidebar-specific tokens from globals.css

### Phase 3 (P2 - Polish)

7. Implement ⌘K CommandPalette component
8. Add CSS variables for fonts (--font-display, --font-ui, --font-mono)
9. Verify all color hex values match DESIGN.md §2 exactly
