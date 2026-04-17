import { ApiClient, ApiError } from '../client'

const BASE_URL = 'http://localhost:8080'

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

function mockFetchFailure(status: number, statusText: string, body?: unknown) {
  fetchMock.mockResolvedValue({
    ok: false,
    status,
    statusText,
    json: () => (body ? Promise.resolve(body) : Promise.reject()),
  } as Response)
  return fetchMock
}

describe('ApiClient', () => {
  let client: ApiClient

  beforeEach(() => {
    client = new ApiClient(BASE_URL)
    fetchMock = jest.fn()
    global.fetch = fetchMock
    localStorage.clear()
  })

  afterEach(() => {
    jest.restoreAllMocks()
  })

  it('sends Content-Type: application/json', async () => {
    const spy = mockFetchSuccess({ ok: true })

    await client.get('/api/test')

    expect(spy).toHaveBeenCalledWith(
      `${BASE_URL}/api/test`,
      expect.objectContaining({
        headers: expect.objectContaining({
          'Content-Type': 'application/json',
        }),
      })
    )
  })

  it('attaches Authorization header when slate_token exists', async () => {
    localStorage.setItem('slate_token', 'my-jwt-token')
    const spy = mockFetchSuccess({ ok: true })

    await client.get('/api/test')

    expect(spy).toHaveBeenCalledWith(
      `${BASE_URL}/api/test`,
      expect.objectContaining({
        headers: expect.objectContaining({
          Authorization: 'Bearer my-jwt-token',
        }),
      })
    )
  })

  it('omits Authorization header when no token in localStorage', async () => {
    const spy = mockFetchSuccess({ ok: true })

    await client.get('/api/test')

    const headers = spy.mock.calls[0][1]?.headers as Record<string, string>
    expect(headers.Authorization).toBeUndefined()
  })

  it('returns parsed JSON on success', async () => {
    const data = { id: '1', name: 'Test' }
    mockFetchSuccess(data)

    const result = await client.get('/api/test')

    expect(result).toEqual(data)
  })

  it('returns undefined for 204 No Content', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      status: 204,
      statusText: 'No Content',
      json: () => Promise.reject(new Error('no body')),
    } as Response)

    const result = await client.delete('/api/test/1')

    expect(result).toBeUndefined()
  })

  it('throws ApiError with status when response.ok is false', async () => {
    const errorBody = { error: 'not found' }
    mockFetchFailure(404, 'Not Found', errorBody)

    await expect(client.get('/api/missing')).rejects.toThrow(ApiError)

    try {
      await client.get('/api/missing')
    } catch (e) {
      const err = e as ApiError
      expect(err.status).toBe(404)
      expect(err.statusText).toBe('Not Found')
      expect(err.body).toEqual(errorBody)
    }
  })

  it('throws ApiError even when error body is not JSON', async () => {
    mockFetchFailure(500, 'Internal Server Error')

    await expect(client.get('/api/broken')).rejects.toThrow(ApiError)

    try {
      await client.get('/api/broken')
    } catch (e) {
      const err = e as ApiError
      expect(err.status).toBe(500)
      expect(err.body).toBeNull()
    }
  })

  it('post() sends method POST with JSON body', async () => {
    const spy = mockFetchSuccess({ created: true })
    const payload = { email: 'test@example.com' }

    await client.post('/api/users', payload)

    expect(spy).toHaveBeenCalledWith(
      `${BASE_URL}/api/users`,
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify(payload),
      })
    )
  })

  it('put() sends method PUT with JSON body', async () => {
    const spy = mockFetchSuccess({ updated: true })
    const payload = { name: 'Updated' }

    await client.put('/api/users/1', payload)

    expect(spy).toHaveBeenCalledWith(
      `${BASE_URL}/api/users/1`,
      expect.objectContaining({
        method: 'PUT',
        body: JSON.stringify(payload),
      })
    )
  })

  it('delete() sends method DELETE', async () => {
    const spy = mockFetchSuccess(undefined, 204)

    await client.delete('/api/users/1')

    expect(spy).toHaveBeenCalledWith(
      `${BASE_URL}/api/users/1`,
      expect.objectContaining({ method: 'DELETE' })
    )
  })
})
