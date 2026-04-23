import type { ImpersonationSession, PaginatedResponse, ListParams } from './types'

/**
 * Impersonation API — start/end removed (see #40 Bug 2).
 *
 * The real start path is `adminUsersApi.impersonate(tenantId, userId)` which
 * calls `POST /api/admin/impersonate/:tenantId/:userId` and returns a
 * `redirect_url` the browser navigates to; the landing page on the target
 * portal handles JWT storage. The admin portal does not own end-of-session
 * state either — the impersonated portal manages that.
 *
 * `listSessions` is kept for the "Recent sessions" table but no
 * corresponding gateway route exists yet. Returns an empty paginated
 * response + logs a console warning so the UI renders the empty state
 * without crashing. Flip the path once admin-auth exposes a session-list
 * RPC (tracked outside this task).
 */
export const impersonationApi = {
  async listSessions(_params?: ListParams): Promise<PaginatedResponse<ImpersonationSession>> {
    if (typeof console !== 'undefined') {
      console.warn(
        '[impersonationApi.listSessions] no gateway route for impersonation session list — returning empty'
      )
    }
    return {
      data: [],
      total: 0,
      page: 1,
      pageSize: 0,
      totalPages: 0,
    }
  },
}

export default impersonationApi
