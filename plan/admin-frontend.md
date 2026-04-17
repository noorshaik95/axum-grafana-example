# Admin Frontend Plan

## Owner Agent: `admin-expert`

## Stack: Next.js 14 (App Router), TypeScript, TanStack Query, Tailwind CSS, shadcn/ui, Playwright

---

## Objective

Build a complete, production-ready Super Admin control plane for the LMS platform. Covers tenant management, billing, impersonation, system health, and role management.

---

## Route Structure

```
app/
├── (auth)/
│   └── login/page.tsx                     ← Already exists, fixed
├── dashboard/
│   └── page.tsx                           ← Platform stats + health overview
├── onboarding/
│   ├── page.tsx                           ← List all onboarding jobs
│   ├── new/page.tsx                       ← Start new onboarding wizard
│   └── [id]/
│       ├── page.tsx                       ← View onboarding status
│       └── steps/[step]/page.tsx          ← Wizard step editor
├── universities/
│   ├── page.tsx                           ← Paginated list + search + filter
│   └── [id]/
│       ├── page.tsx                       ← University profile
│       ├── plan/page.tsx                  ← Resource plan management
│       ├── billing/page.tsx               ← Billing history + credits
│       └── users/page.tsx                 ← University admin users
├── iam/
│   ├── users/page.tsx                     ← Admin user list + invite
│   ├── roles/page.tsx                     ← Role-permission matrix
│   └── audit/page.tsx                     ← Audit log
├── billing/
│   └── page.tsx                           ← Platform-wide billing overview
├── system/
│   ├── health/page.tsx                    ← Service health dashboard
│   ├── kafka/page.tsx                     ← Kafka event stream monitor
│   └── alerts/page.tsx                    ← Alert configuration
└── impersonation/
    └── page.tsx                           ← Impersonation management
```

---

## Key Components

### 1. University Onboarding Wizard (`components/onboarding/wizard/`)

Multi-step wizard with state persisted to Restate via API:

- Step 1: Basic institution details (name, address, logo upload, timezone, locale)
- Step 2: Institution Admin user (email invite, role assignment)
- Step 3: Resource plan (storage quota, max users, max courses, feature flags)
- Step 4: Review & submit

```typescript
// components/onboarding/wizard/OnboardingWizard.tsx
const steps = [
  { id: 1, title: 'Institution Details', component: InstitutionDetailsStep },
  { id: 2, title: 'Admin User', component: AdminUserStep },
  { id: 3, title: 'Resource Plan', component: ResourcePlanStep },
  { id: 4, title: 'Review', component: ReviewStep },
]
```

Progress tracked via `useOnboardingWizard` hook that calls Restate API.

### 2. University List (`components/universities/UniversityList.tsx`)

- DataTable with server-side pagination, search, status filter
- Bulk actions: enable/disable selected
- Status badges: Active | Suspended | Pending | In Review
- Quick actions per row: view, enable/disable, impersonate admin

### 3. Platform Health Dashboard (`components/dashboard/`)

- ServiceHealthGrid — shows each microservice status (green/yellow/red)
- KPI cards: active tenants, DAU, storage used, error rate
- Kafka consumer lag widget
- Recent error events feed

### 4. Role-Permission Matrix (`components/iam/RolePermissionMatrix.tsx`)

Interactive grid:

- Rows: roles (Super Admin, Billing Admin, Support Admin, Viewer)
- Columns: permissions (manage_tenants, manage_billing, view_metrics, impersonate, etc.)
- Checkboxes to toggle permissions, saved on submit

### 5. Impersonation Banner (`components/impersonation/ImpersonationBanner.tsx`)

Persistent fixed banner when in impersonation session:

```tsx
<div className="fixed top-0 w-full bg-yellow-500 text-black py-2 px-4 z-50">
  ⚠️ Impersonating {userName} ({tenantName}) —
  <Button onClick={endImpersonation}>Return to Admin</Button>
</div>
```

### 6. Billing Management

- Per-university billing history table
- Credit/adjustment modal with justification field
- Upcoming invoices list
- Stripe customer portal link (opens in new tab)

---

## API Integration

### API Client (`lib/api/`)

```typescript
// lib/api/onboarding.ts
export const onboardingApi = {
  start: (id: string, payload: StartPayload) => post(`/onboarding/${id}/start`, payload),
  saveStep: (id: string, step: number, data: unknown) =>
    put(`/onboarding/${id}/step/${step}`, data),
  submit: (id: string) => post(`/onboarding/${id}/submit`),
  approve: (id: string, adminId: string) => post(`/onboarding/${id}/approve`, { adminId }),
  getStatus: (id: string) => get(`/onboarding/${id}/status`),
}

// lib/api/tenants.ts
export const tenantsApi = {
  list: (params: ListTenantsParams) => get('/tenants', { params }),
  get: (id: string) => get(`/tenants/${id}`),
  toggleAccess: (id: string, enabled: boolean, reason: string) =>
    patch(`/tenants/${id}/access`, { enabled, reason }),
  updatePlan: (id: string, plan: Plan) => put(`/tenants/${id}/plan`, plan),
  getUsage: (id: string) => get(`/tenants/${id}/usage`),
}
```

---

## Playwright Tests (`tests/e2e/admin/`)

```typescript
// tests/e2e/admin/onboarding.spec.ts
test('Full university onboarding flow', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/onboarding/new');
  // Step 1: Fill institution details
  await page.fill('[name="institutionName"]', 'Test University');
  await page.fill('[name="address"]', '123 Main St');
  await page.click('button:has-text("Next")');
  // Step 2: Admin user
  await page.fill('[name="adminEmail"]', 'admin@testuni.edu');
  await page.click('button:has-text("Next")');
  // Step 3: Plan
  await page.selectOption('[name="plan"]', 'standard');
  await page.click('button:has-text("Next")');
  // Step 4: Review + submit
  await page.click('button:has-text("Submit for Review")');
  await expect(page.getByText('Pending Review')).toBeVisible();
});

// tests/e2e/admin/universities.spec.ts
test('Enable/disable university', async ({ page }) => { ... });

// tests/e2e/admin/impersonation.spec.ts
test('Impersonation start and end', async ({ page }) => { ... });

// tests/e2e/admin/iam.spec.ts
test('Role creation and permission assignment', async ({ page }) => { ... });
```

---

## Files to Create/Modify

- [MODIFY] `frontend/admin/app/` — all route pages
- [NEW] `frontend/admin/components/onboarding/wizard/`
- [NEW] `frontend/admin/components/universities/`
- [NEW] `frontend/admin/components/dashboard/health/`
- [NEW] `frontend/admin/components/iam/`
- [NEW] `frontend/admin/components/billing/`
- [NEW] `frontend/admin/components/impersonation/`
- [NEW] `frontend/admin/lib/api/` — typed API clients
- [NEW] `tests/e2e/admin/` — Playwright tests
