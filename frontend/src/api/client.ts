import type { APIError } from '../types/api';

class ApiClientError extends Error {
  code: string;
  status: number;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = 'ApiClientError';
    this.status = status;
    this.code = code;
  }
}

async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(url, {
    credentials: 'include',
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  });

  if (!res.ok) {
    let apiError: APIError = { code: 'UNKNOWN', message: res.statusText };
    try {
      apiError = (await res.json()) as APIError;
    } catch {
      // response body was not JSON
    }
    throw new ApiClientError(res.status, apiError.code, apiError.message);
  }

  if (res.status === 204) {
    return undefined as T;
  }

  return res.json() as Promise<T>;
}

export const api = {
  get<T>(url: string): Promise<T> {
    return request<T>(url, { method: 'GET' });
  },
  post<T>(url: string, body?: unknown): Promise<T> {
    return request<T>(url, {
      method: 'POST',
      body: body ? JSON.stringify(body) : undefined,
    });
  },
  put<T>(url: string, body?: unknown): Promise<T> {
    return request<T>(url, {
      method: 'PUT',
      body: body ? JSON.stringify(body) : undefined,
    });
  },
  del<T>(url: string, body?: unknown): Promise<T> {
    return request<T>(url, {
      method: 'DELETE',
      body: body ? JSON.stringify(body) : undefined,
    });
  },
};

export { ApiClientError };
