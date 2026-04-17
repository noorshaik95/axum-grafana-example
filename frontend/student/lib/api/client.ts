const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://api.slate.local';

function getToken(): string | null {
  if (typeof window === 'undefined') return null;
  return localStorage.getItem('student_token') || localStorage.getItem('slate_token');
}

function getTenantId(): string | null {
  if (typeof window === 'undefined') return null;
  return localStorage.getItem('tenant_id');
}

export class ApiError extends Error {
  constructor(
    public status: number,
    public statusText: string,
    public body: Record<string, unknown> | null
  ) {
    super(`${status} ${statusText}`);
    this.name = 'ApiError';
  }
}

export async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = getToken();
  const tenantId = getTenantId();
  const headers: Record<string, string> = {
    ...(options.headers as Record<string, string>),
  };

  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }
  if (tenantId) {
    headers['X-Tenant-ID'] = tenantId;
  }

  // Don't set Content-Type for FormData (browser sets multipart boundary)
  if (!(options.body instanceof FormData)) {
    headers['Content-Type'] = 'application/json';
  }

  const response = await fetch(`${API_URL}${path}`, {
    ...options,
    headers,
  });

  if (!response.ok) {
    let body: Record<string, unknown> | null = null;
    try {
      body = await response.json();
    } catch {
      // not JSON
    }
    throw new ApiError(response.status, response.statusText, body);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return response.json();
}

export function get<T>(path: string): Promise<T> {
  return request<T>(path, { method: 'GET' });
}

export function post<T>(path: string, data?: unknown): Promise<T> {
  if (data instanceof FormData) {
    return request<T>(path, { method: 'POST', body: data });
  }
  return request<T>(path, {
    method: 'POST',
    body: data ? JSON.stringify(data) : undefined,
  });
}

export function put<T>(path: string, data?: unknown): Promise<T> {
  return request<T>(path, {
    method: 'PUT',
    body: data ? JSON.stringify(data) : undefined,
  });
}

export function patch<T>(path: string, data?: unknown): Promise<T> {
  return request<T>(path, {
    method: 'PATCH',
    body: data ? JSON.stringify(data) : undefined,
  });
}

export function del<T>(path: string): Promise<T> {
  return request<T>(path, { method: 'DELETE' });
}
