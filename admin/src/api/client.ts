// Fetch-based API client with multipart upload support for admin operations
import type { APIError } from '../types/api.ts';

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
  del<T>(url: string): Promise<T> {
    return request<T>(url, { method: 'DELETE' });
  },
  async upload<T>(url: string, formData: FormData): Promise<T> {
    const res = await fetch(url, {
      method: 'POST',
      credentials: 'include',
      body: formData,
      // No Content-Type header — browser sets multipart boundary automatically
    });

    if (!res.ok) {
      let apiError: APIError = { code: 'UNKNOWN', message: res.statusText };
      try {
        apiError = (await res.json()) as APIError;
      } catch {
        // not JSON
      }
      throw new ApiClientError(res.status, apiError.code, apiError.message);
    }

    return res.json() as Promise<T>;
  },
};

export { ApiClientError };
