import { create } from 'zustand';
import { api, ApiClientError } from '../api/client';
import type { AuthUser } from '../types/api';

interface AuthState {
  user: AuthUser | null;
  isLoggedIn: boolean;
  isLoading: boolean;
  fetchUser: () => Promise<void>;
  logout: () => Promise<void>;
}

export const useAuth = create<AuthState>((set) => ({
  user: null,
  isLoggedIn: false,
  isLoading: true,

  fetchUser: async () => {
    try {
      const user = await api.get<AuthUser>('/api/auth/me');
      set({ user, isLoggedIn: true, isLoading: false });
    } catch (err) {
      if (err instanceof ApiClientError && err.status === 401) {
        set({ user: null, isLoggedIn: false, isLoading: false });
        return;
      }
      set({ user: null, isLoggedIn: false, isLoading: false });
    }
  },

  logout: async () => {
    try {
      await api.post('/api/auth/logout');
    } finally {
      set({ user: null, isLoggedIn: false });
    }
  },
}));
