import { renderHook, waitFor, act } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { createElement, type ReactNode } from 'react'
import { useProfile, useCourses, useLogin } from '../hooks'
import * as authApi from '../auth'
import * as coursesApi from '../courses'

// Mock the API modules
jest.mock('../auth')
jest.mock('../courses')

const mockedAuth = authApi as jest.Mocked<typeof authApi>
const mockedCourses = coursesApi as jest.Mocked<typeof coursesApi>

const mockUser = {
  id: 'u1',
  email: 'test@slate.edu',
  firstName: 'Test',
  lastName: 'User',
  isActive: true,
  authMethod: 'normal' as const,
  timezone: 'UTC',
  roles: [] as const,
  createdAt: '2024-01-01T00:00:00Z',
  updatedAt: '2024-01-01T00:00:00Z',
}

const mockTokens = {
  accessToken: 'access-123',
  refreshToken: 'refresh-456',
  expiresIn: 900,
  tokenType: 'Bearer' as const,
}

const mockPaginatedCourses = {
  success: true as const,
  data: [
    { id: 'c1', name: 'Intro to CS', code: 'CS101' },
    { id: 'c2', name: 'Data Structures', code: 'CS201' },
  ],
  pagination: { page: 1, pageSize: 20, totalItems: 2, totalPages: 1 },
  timestamp: '2024-01-01T00:00:00Z',
}

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  })
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children)
  }
}

describe('API hooks', () => {
  beforeEach(() => {
    jest.clearAllMocks()
    localStorage.clear()
  })

  describe('useProfile()', () => {
    it('starts in loading state then returns user data', async () => {
      mockedAuth.getProfile.mockResolvedValue(mockUser)

      const { result } = renderHook(() => useProfile(), {
        wrapper: createWrapper(),
      })

      // Initially loading
      expect(result.current.isLoading).toBe(true)

      // Eventually resolves with user data
      await waitFor(() => expect(result.current.isSuccess).toBe(true))
      expect(result.current.data?.email).toBe('test@slate.edu')
      expect(mockedAuth.getProfile).toHaveBeenCalledTimes(1)
    })

    it('returns error state when API fails', async () => {
      mockedAuth.getProfile.mockRejectedValue(new Error('unauthorized'))

      const { result } = renderHook(() => useProfile(), {
        wrapper: createWrapper(),
      })

      await waitFor(() => expect(result.current.isError).toBe(true))
      expect(result.current.error).toBeDefined()
    })
  })

  describe('useCourses()', () => {
    it('returns paginated courses on success', async () => {
      mockedCourses.listCourses.mockResolvedValue(mockPaginatedCourses as any)

      const { result } = renderHook(() => useCourses(), {
        wrapper: createWrapper(),
      })

      await waitFor(() => expect(result.current.isSuccess).toBe(true))
      expect(result.current.data?.data).toHaveLength(2)
      expect(mockedCourses.listCourses).toHaveBeenCalledWith(undefined)
    })

    it('passes filters to listCourses', async () => {
      mockedCourses.listCourses.mockResolvedValue(mockPaginatedCourses as any)
      const filters = { search: 'intro', department: 'CS' }

      const { result } = renderHook(() => useCourses(filters), {
        wrapper: createWrapper(),
      })

      await waitFor(() => expect(result.current.isSuccess).toBe(true))
      expect(mockedCourses.listCourses).toHaveBeenCalledWith(filters)
    })
  })

  describe('useLogin()', () => {
    it('calls auth.login and returns tokens on mutate', async () => {
      mockedAuth.login.mockResolvedValue({ ...mockTokens, user: mockUser })

      const { result } = renderHook(() => useLogin(), {
        wrapper: createWrapper(),
      })

      act(() => {
        result.current.mutate({
          email: 'test@slate.edu',
          password: 'pass123',
        })
      })

      await waitFor(() => expect(result.current.isSuccess).toBe(true))
      expect(mockedAuth.login).toHaveBeenCalledWith({
        email: 'test@slate.edu',
        password: 'pass123',
      })
      expect(result.current.data?.accessToken).toBe('access-123')
    })

    it('returns error state when login fails', async () => {
      mockedAuth.login.mockRejectedValue(new Error('invalid credentials'))

      const { result } = renderHook(() => useLogin(), {
        wrapper: createWrapper(),
      })

      act(() => {
        result.current.mutate({
          email: 'wrong@slate.edu',
          password: 'bad',
        })
      })

      await waitFor(() => expect(result.current.isError).toBe(true))
      expect(result.current.error?.message).toBe('invalid credentials')
    })
  })
})
