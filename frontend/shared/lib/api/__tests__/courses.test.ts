import { listCourses, getCourse, enrollInCourse } from '../courses'

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

const mockPaginatedCourses = {
  success: true,
  data: [
    { id: 'c1', name: 'Intro to CS', code: 'CS101' },
    { id: 'c2', name: 'Data Structures', code: 'CS201' },
  ],
  pagination: {
    page: 1,
    pageSize: 20,
    totalItems: 2,
    totalPages: 1,
  },
  timestamp: '2024-01-01T00:00:00Z',
}

const mockCourse = {
  id: 'c1',
  name: 'Intro to CS',
  code: 'CS101',
}

describe('courses module', () => {
  beforeEach(() => {
    fetchMock = jest.fn()
    global.fetch = fetchMock
    localStorage.clear()
  })

  afterEach(() => {
    jest.restoreAllMocks()
  })

  describe('listCourses()', () => {
    it('GETs /api/courses with no filters', async () => {
      const spy = mockFetchSuccess(mockPaginatedCourses)

      await listCourses()

      expect(spy).toHaveBeenCalledWith(
        expect.stringContaining('/api/courses'),
        expect.objectContaining({ method: 'GET' })
      )
      // Should NOT have query string when no filters
      const url = spy.mock.calls[0][0] as string
      expect(url).not.toContain('?')
    })

    it('appends search param when filter.search is set', async () => {
      const spy = mockFetchSuccess(mockPaginatedCourses)

      await listCourses({ search: 'intro' })

      const url = spy.mock.calls[0][0] as string
      expect(url).toContain('search=intro')
    })

    it('appends term param when filter.term is set', async () => {
      const spy = mockFetchSuccess(mockPaginatedCourses)

      await listCourses({ term: 'Fall 2024' })

      const url = spy.mock.calls[0][0] as string
      expect(url).toContain('term=Fall+2024')
    })

    it('appends instructorId param', async () => {
      const spy = mockFetchSuccess(mockPaginatedCourses)

      await listCourses({ instructorId: 'inst-1' })

      const url = spy.mock.calls[0][0] as string
      expect(url).toContain('instructorId=inst-1')
    })

    it('appends isPublished as string', async () => {
      const spy = mockFetchSuccess(mockPaginatedCourses)

      await listCourses({ isPublished: true })

      const url = spy.mock.calls[0][0] as string
      expect(url).toContain('isPublished=true')
    })

    it('appends department param', async () => {
      const spy = mockFetchSuccess(mockPaginatedCourses)

      await listCourses({ department: 'CS' })

      const url = spy.mock.calls[0][0] as string
      expect(url).toContain('department=CS')
    })

    it('appends page and pageSize params', async () => {
      const spy = mockFetchSuccess(mockPaginatedCourses)

      await listCourses({ page: 2, pageSize: 10 })

      const url = spy.mock.calls[0][0] as string
      expect(url).toContain('page=2')
      expect(url).toContain('pageSize=10')
    })

    it('combines multiple filters into query string', async () => {
      const spy = mockFetchSuccess(mockPaginatedCourses)

      await listCourses({ search: 'data', department: 'CS', page: 1 })

      const url = spy.mock.calls[0][0] as string
      expect(url).toContain('search=data')
      expect(url).toContain('department=CS')
      expect(url).toContain('page=1')
    })

    it('returns paginated response', async () => {
      mockFetchSuccess(mockPaginatedCourses)

      const result = await listCourses()

      expect(result.data).toHaveLength(2)
      expect(result.pagination.totalItems).toBe(2)
    })
  })

  describe('getCourse()', () => {
    it('GETs /api/courses/:id', async () => {
      const spy = mockFetchSuccess(mockCourse)

      await getCourse('c1')

      expect(spy).toHaveBeenCalledWith(
        expect.stringContaining('/api/courses/c1'),
        expect.objectContaining({ method: 'GET' })
      )
    })

    it('returns course data', async () => {
      mockFetchSuccess(mockCourse)

      const result = await getCourse('c1')

      expect(result).toEqual(mockCourse)
    })
  })

  describe('enrollInCourse()', () => {
    it('POSTs to /api/courses/:id/enroll', async () => {
      const mockEnrollment = { id: 'e1', courseId: 'c1', studentId: 's1' }
      const spy = mockFetchSuccess(mockEnrollment)

      await enrollInCourse('c1')

      expect(spy).toHaveBeenCalledWith(
        expect.stringContaining('/api/courses/c1/enroll'),
        expect.objectContaining({ method: 'POST' })
      )
    })

    it('returns enrollment data', async () => {
      const mockEnrollment = {
        id: 'e1',
        courseId: 'c1',
        studentId: 's1',
        status: 'active',
      }
      mockFetchSuccess(mockEnrollment)

      const result = await enrollInCourse('c1')

      expect(result).toEqual(mockEnrollment)
    })
  })
})
