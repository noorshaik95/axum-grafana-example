# Slate LMS — Design System & UX

## 1. Design Philosophy

Slate's UI is built around a single principle: **action-first**. Every page opens by answering "what is the next thing I should do right now?" before presenting any data summaries or navigation.

Secondary principles:
- **Warm, not clinical.** Academic software tends toward cold grays and rigid tables. Slate uses a forest-green and warm-cream palette to feel inviting without being playful.
- **Resume-state everywhere.** Students and instructors should always be able to pick up exactly where they left off without hunting for their place.
- **Zero sidebar tax.** Navigation is in a sticky top bar + a contextual "Now bar" that changes based on the current section. Sidebars hide content behind a click; the Now bar surfaces the most important action inline.

---

## 2. Design System

### Font Stack

| Role | Font | Usage |
|---|---|---|
| Display / headings | Instrument Serif | Page titles, hero text, course names |
| UI / body | Inter | All interface labels, body text, buttons |
| Code / monospace | JetBrains Mono | Code snippets, submission previews, IDs |

### Color Palette

**Forest (primary brand)**

| Token | Hex | Usage |
|---|---|---|
| `forest-50` | `#f0f7f0` | Tinted backgrounds, hover states |
| `forest-100` | `#dceddc` | Card borders, dividers |
| `forest-200` | `#b9dbb9` | Subtle accents |
| `forest-300` | `#89c489` | Icons, progress fills |
| `forest-400` | `#5aad5a` | Secondary actions |
| `forest-500` | `#3d9b3d` | Primary brand |
| `forest-600` | `#2e7d2e` | Primary button hover |
| `forest-700` | `#1f5c1f` | Now bar background, active nav |
| `forest-800` | `#154015` | Dark headings |
| `forest-900` | `#0d280d` | Dark mode base |

**Neutrals (warm, not gray)**

| Token | Hex | Usage |
|---|---|---|
| `cream` | `#fefcf7` | Main page background |
| `paper` | `#f8f4ec` | Card background |
| `warm-50` | `#faf8f3` | Table rows, list items |
| `warm-100` | `#f2ede3` | Input backgrounds |
| `warm-200` | `#e8e0d0` | Borders |
| `warm-700` | `#5c5244` | Body text |
| `warm-900` | `#2a2218` | Headings |

**Semantic**

| Token | Hex | Usage |
|---|---|---|
| `amber-400` | `#f59e0b` | Warnings, due-soon badges |
| `amber-500` | `#d97706` | Amber CTA buttons |
| `red-500` | `#ef4444` | Errors, overdue, critical risk |
| `blue-500` | `#3b82f6` | Info, links |

### Spacing Scale
Standard Tailwind scale. Key values: `4px`, `8px`, `12px`, `16px`, `24px`, `32px`, `48px`, `64px`.

### Border Radii
- Cards: `rounded-2xl` (16px)
- Buttons: `rounded-xl` (12px)
- Badges/chips: `rounded-full`
- Inputs: `rounded-lg` (8px)

---

## 3. Navigation Pattern

### Topbar (sticky, blurred cream background)
- Left: Slate wordmark (Instrument Serif)
- Center: Portal-specific primary nav links (at most 5)
- Right: Search trigger (`⌘K`), notifications bell, user avatar menu

The topbar background is `backdrop-blur-md bg-cream/80` so page content scrolls behind it.

### Now Bar (forest-700 background)
A full-width strip directly below the topbar that surfaces contextual action chips for the current section. Examples:
- On `/today`: "3 assignments due this week", "Your next class is in 2h"
- On `/grade/[id]`: "17 submissions pending — 5 in pattern A"
- On `/roster/[id]`: "2 students at high risk"

Chips are dismissible. The bar disappears on pages where there is no contextual action.

### ⌘K Command Palette
- Fuzzy search across courses, students, assignments, and pages
- Available on every page
- Keyboard shortcut: `Cmd+K` / `Ctrl+K`

---

## 4. Component Patterns

### Action Hero
The first thing on every page. A large Instrument Serif heading ("Good morning, Alice") followed by the primary CTA button. Sits above the fold, centered or left-aligned depending on context.

### Stat Cards
`paper` background, `rounded-2xl`, subtle `warm-200` border. Contains:
- Icon (forest-500)
- Numeric value (2xl bold)
- Label (warm-700, small)
- Optional trend indicator (↑/↓ with color)

### List Rows
Full-width, alternating `warm-50`/`cream` backgrounds. Left: icon + primary label + secondary label. Right: badge + action button. Used for inbox, grading queue, roster.

### Cards
`paper` bg, `rounded-2xl`, `shadow-sm`. Used for courses, assignments, announcements. Hover: `shadow-md` + subtle `forest-50` tint.

### Badges

| Variant | Background | Text | Usage |
|---|---|---|---|
| `green` | forest-100 | forest-700 | Active, completed, passing |
| `amber` | amber-100 | amber-700 | Due soon, at-risk medium |
| `red` | red-100 | red-700 | Overdue, failed, high risk |
| `gray` | warm-100 | warm-700 | Draft, inactive |
| `blue` | blue-100 | blue-700 | Info, new, open |

### Buttons

| Variant | Style | Usage |
|---|---|---|
| `primary` | forest-600 bg, white text | Primary CTA |
| `secondary` | warm-100 bg, warm-900 text | Secondary actions |
| `ghost` | transparent bg, warm-700 text, hover warm-50 | Tertiary, inline actions |
| `amber` | amber-500 bg, white text | Deadline-urgent actions |
| `destructive` | red-500 bg, white text | Delete, reject |

---

## 5. Per-Portal UX Intent

### Student Portal (`/` on port 3001)

**Core intent:** Reduce friction from "logged in" to "doing work."

- `/today` is the landing page, not `/dashboard`. It shows: assignments due this week (sorted by urgency), the next scheduled class, a quick resume button for in-progress courses.
- **Resume-state everywhere:** every course card shows exactly where the student left off (module + lesson).
- **Peer social layer:** grade comparison shows percentile position in the class distribution without naming peers.
- **Study plan (`/plan`):** A generated 16-week calendar derived from syllabi. Students can tweak individual weeks.
- **Office hours (`/office-hours`):** Slot browser with professor availability. Booking auto-attaches the student's most recent unanswered question.

### Instructor Portal (`/` on port 3002)

**Core intent:** Make grading and at-risk detection effortless.

- `/teach` is the landing page. It shows: grading queue counts per assignment, roster alerts (high-risk students), upcoming classes.
- **Grading queue (`/grade/[assignmentId]`):** Submissions are pre-grouped by detected answer pattern. Instructors grade a pattern once and apply the score to the whole group (batch grade), or grade individually.
- **Roster health (`/roster/[courseId]`):** Students sorted by risk score (computed from login gaps, late submissions, grade trajectory). One-click "nudge" sends a check-in message.
- **Office hours:** Instructors create availability slots; students book them. Booked slots appear in the instructor's schedule with the student's question pre-attached.

### Admin Portal (`/` on port 3003)

**Core intent:** Ops HQ — know the health of the entire platform at a glance.

- Landing page is the service status board: all microservices, health indicators, active incidents count, tenant provisioning pipeline.
- **Schools (`/schools`):** Each tenant is a "school" page with contract/renewal data, usage metrics, health status, open incidents. Not just a table — a relationship page.
- **Incidents (`/incidents`):** Triage queue with status (open / watch / resolved), priority (P1-P4), timeline entries, and one-click tenant notification.
- **Feature flags (`/flags`):** Toggle features globally, per-tenant, or per-role. Changes take effect without deployment.
- **Broadcast (`/broadcast`):** Send a platform-wide or tenant-scoped message. History tab shows past broadcasts.

---

## 6. Route Tables

### Student Portal

| Route | Description | Status |
|---|---|---|
| `/today` | Action-first daily dashboard | New |
| `/courses` | Course catalog + enrolled courses | Existing |
| `/courses/[id]` | Course detail + module/lesson navigator | Improved |
| `/grades` | Grade card with peer comparison percentile | Improved |
| `/plan` | AI-generated 16-week study plan | New |
| `/office-hours` | Professor OH slot browser + booking | New |
| `/discuss/[id]` | Course discussion thread detail | New |
| `/inbox` | In-platform messaging inbox | Existing |
| `/profile` | Profile + settings | Existing |
| `/settings` | Notification + accessibility settings | Existing |

### Instructor Portal

| Route | Description | Status |
|---|---|---|
| `/teach` | Instructor ops dashboard | New |
| `/courses` | Manage courses | Existing |
| `/courses/[id]` | Course editor | Existing |
| `/grade/[assignmentId]` | Queue-style grading with pattern grouping | New |
| `/roster/[courseId]` | Student health / risk roster | New |
| `/office-hours` | OH slot creation + schedule | New |
| `/people` | Student directory for course | New |
| `/inbox` | In-platform messaging | Existing |

### Admin Portal

| Route | Description | Status |
|---|---|---|
| `/` | Ops HQ — service status + incident summary | New |
| `/schools` | Tenant relationship pages | New |
| `/schools/[id]` | Individual school detail + incidents | New |
| `/incidents` | Incident triage queue | New |
| `/incidents/[id]` | Incident detail + timeline | New |
| `/flags` | Feature flag management | New |
| `/broadcast` | Platform-wide messaging + history | New |
| `/ops` | Infrastructure health + metrics deep-dive | New |

---

## 7. Design Tokens Reference (CSS Variables)

```css
:root {
  /* Forest */
  --color-forest-50:  #f0f7f0;
  --color-forest-100: #dceddc;
  --color-forest-200: #b9dbb9;
  --color-forest-300: #89c489;
  --color-forest-400: #5aad5a;
  --color-forest-500: #3d9b3d;
  --color-forest-600: #2e7d2e;
  --color-forest-700: #1f5c1f;
  --color-forest-800: #154015;
  --color-forest-900: #0d280d;

  /* Neutrals (warm) */
  --color-cream:     #fefcf7;
  --color-paper:     #f8f4ec;
  --color-warm-50:   #faf8f3;
  --color-warm-100:  #f2ede3;
  --color-warm-200:  #e8e0d0;
  --color-warm-300:  #d4c8b4;
  --color-warm-700:  #5c5244;
  --color-warm-900:  #2a2218;

  /* Semantic */
  --color-amber-400: #f59e0b;
  --color-amber-500: #d97706;
  --color-red-500:   #ef4444;
  --color-blue-500:  #3b82f6;

  /* Typography */
  --font-display: 'Instrument Serif', Georgia, serif;
  --font-ui:      'Inter', system-ui, sans-serif;
  --font-mono:    'JetBrains Mono', 'Fira Code', monospace;

  /* Radii */
  --radius-card:   1rem;    /* 16px */
  --radius-button: 0.75rem; /* 12px */
  --radius-input:  0.5rem;  /* 8px */
  --radius-badge:  9999px;
}
```

---

## 8. Design Files Location

HTML design canvases and the shared CSS design system are in the `design/` directory at the repo root:

```
design/
├── slate-ds.css              # Full design system CSS (tokens, components)
├── student-canvas.html       # Student portal interactive canvas
├── provider-canvas.html      # Instructor portal interactive canvas
└── admin-canvas.html         # Admin portal interactive canvas
```

To copy the design files from the source location (if not already present):
```bash
mkdir -p /Users/noorullahshaik/Code/slate/design
cp /tmp/slate-design/slate/project/*.html /Users/noorullahshaik/Code/slate/design/
cp /tmp/slate-design/slate/project/slate-ds.css /Users/noorullahshaik/Code/slate/design/
```
