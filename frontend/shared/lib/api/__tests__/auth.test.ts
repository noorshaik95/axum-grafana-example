import { login, register, logout, getProfile } from '../auth'

let fetchMock: jest.Mock

function mockFetchSuccess(body: unknown, status = 200) {
  fetchMock.mockResolvedValue({
    ok: true,
    status,
    statusText: 'OK',
    json: () => Promise.resolve(body),
  } as Response)
  return fetchMock
}

const mockUser = {
  id: 'u1',
  email: 'test@slate.edu',
  firstName: 'Test',
  lastName: 'User',
  isActive: true,
  authMethod: 'normal' as const,
  timezone: 'UTC',
  roles: [],
  createdAt: '2024-01-01T00:00:00Z',
  updatedAt: '2024-01-01T00:00:00Z',
}

const mockTokens = {
  accessToken: 'access-jwt-123',
  refreshToken: 'refresh-jwt-456',
  expiresIn: 900,
  tokenType: 'Bearer' as const,
}

describe('auth module', () => {
  beforeEach(() => {
    fetchMock = jest.fn()
    global.fetch = fetchMock
    localStorage.clear()
  })

  afterEach(() => {
    jest.restoreAllMocks()
  })

  describe('login()', () => {
    it('POSTs to /api/auth/login with credentials', async () => {
      const spy = mockFetchSuccess({ ...mockTokens, user: mockUser })

      await login({ email: 'test@slate.edu', password: 'pass123' })

      expect(spy).toHaveBeenCalledWith(
        expect.stringContaining('/api/auth/login'),
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({
            email: 'test@slate.edu',
            password: 'pass123',
          }),
        })
      )
    })

    it('saves accessToken to localStorage as slate_token', async () => {
      mockFetchSuccess({ ...mockTokens, user: mockUser })

      await login({ email: 'test@slate.edu', password: 'pass123' })

      expect(localStorage.getItem('slate_token')).toBe('access-jwt-123')
    })

    it('saves refreshToken to localStorage as slate_refresh_token', async () => {
      mockFetchSuccess({ ...mockTokens, user: mockUser })

      await login({ email: 'test@slate.edu', password: 'pass123' })

      expect(localStorage.getItem('slate_refresh_token')).toBe('refresh-jwt-456')
    })

    it('returns tokens and user', async () => {
      const responseData = { ...mockTokens, user: mockUser }
      mockFetchSuccess(responseData)

      const result = await login({
        email: 'test@slate.edu',
        password: 'pass123',
      })

      expect(result.accessToken).toBe('access-jwt-123')
      expect(result.user.email).toBe('test@slate.edu')
    })
  })

  describe('register()', () => {
    it('POSTs to /api/auth/register with user data', async () => {
      const spy = mockFetchSuccess({ ...mockTokens, user: mockUser })

      await register({
        email: 'new@slate.edu',
        password: 'pass123',
        firstName: 'New',
        lastName: 'User',
      })

      expect(spy).toHaveBeenCalledWith(
        expect.stringContaining('/api/auth/register'),
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({
            email: 'new@slate.edu',
            password: 'pass123',
            firstName: 'New',
            lastName: 'User',
          }),
        })
      )
    })

    it('saves tokens to localStorage on success', async () => {
      mockFetchSuccess({ ...mockTokens, user: mockUser })

      await register({
        email: 'new@slate.edu',
        password: 'pass123',
        firstName: 'New',
        lastName: 'User',
      })

      expect(localStorage.getItem('slate_token')).toBe('access-jwt-123')
      expect(localStorage.getItem('slate_refresh_token')).toBe('refresh-jwt-456')
    })
  })

  describe('logout()', () => {
    it('POSTs to /api/auth/logout', async () => {
      fetchMock.mockResolvedValue({
        ok: true,
        status: 204,
        statusText: 'No Content',
        json: () => Promise.resolve(undefined),
      } as Response)
      const spy = fetchMock

      localStorage.setItem('slate_token', 'some-token')
      localStorage.setItem('slate_refresh_token', 'some-refresh')

      await logout()

      expect(spy).toHaveBeenCalledWith(
        expect.stringContaining('/api/auth/logout'),
        expect.objectContaining({ method: 'POST' })
      )
    })

    it('removes slate_token and slate_refresh_token from localStorage', async () => {
      fetchMock.mockResolvedValue({
        ok: true,
        status: 204,
        statusText: 'No Content',
        json: () => Promise.resolve(undefined),
      } as Response)

      localStorage.setItem('slate_token', 'some-token')
      localStorage.setItem('slate_refresh_token', 'some-refresh')

      await logout()

      expect(localStorage.getItem('slate_token')).toBeNull()
      expect(localStorage.getItem('slate_refresh_token')).toBeNull()
    })

    it('removes tokens even when the server call fails', async () => {
      fetchMock.mockResolvedValue({
        ok: false,
        status: 500,
        statusText: 'Internal Server Error',
        json: () => Promise.reject(),
      } as Response)

      localStorage.setItem('slate_token', 'some-token')
      localStorage.setItem('slate_refresh_token', 'some-refresh')

      // logout uses try/finally — the error propagates but tokens are still cleared
      await expect(logout()).rejects.toThrow()

      expect(localStorage.getItem('slate_token')).toBeNull()
      expect(localStorage.getItem('slate_refresh_token')).toBeNull()
    })
  })

  describe('getProfile()', () => {
    it('GETs /api/users/profile', async () => {
      const spy = mockFetchSuccess(mockUser)

      await getProfile()

      expect(spy).toHaveBeenCalledWith(
        expect.stringContaining('/api/users/profile'),
        expect.objectContaining({ method: 'GET' })
      )
    })

    it('sends Authorization header when token exists', async () => {
      localStorage.setItem('slate_token', 'my-jwt')
      const spy = mockFetchSuccess(mockUser)

      await getProfile()

      expect(spy).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({
          headers: expect.objectContaining({
            Authorization: 'Bearer my-jwt',
          }),
        })
      )
    })

    it('returns user data', async () => {
      mockFetchSuccess(mockUser)

      const user = await getProfile()

      expect(user.email).toBe('test@slate.edu')
      expect(user.firstName).toBe('Test')
    })
  })
})
