// Zustand auth store — fetches current user session and exposes login state
import { create } from 'zustand';
import { api, ApiClientError } from '../api/client.ts';
import { adminLogin } from '../api/auth.ts';
import type { AuthUser } from '../types/api.ts';

interface AuthState {
  user: AuthUser | null;
  isLoggedIn: boolean;
  isLoading: boolean;
  login: (usrId: string, password: string) => Promise<void>;
  fetchUser: () => Promise<void>;
  logout: () => Promise<void>;
}

export const useAuth = create<AuthState>((set, get) => ({
  user: null,
  isLoggedIn: false,
  isLoading: true,

  login: async (usrId: string, password: string) => {
    await adminLogin({ usrId, password });
    await get().fetchUser();
  },

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
