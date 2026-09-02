// useMobileNav — Zustand store for the mobile more-menu sheet state
import { create } from 'zustand';

interface MobileNavState {
  isOpen: boolean;
  setOpen: (isOpen: boolean) => void;
  close: () => void;
  toggle: () => void;
}

export const useMobileNav = create<MobileNavState>((set) => ({
  isOpen: false,
  setOpen: (isOpen) => set({ isOpen }),
  close: () => set({ isOpen: false }),
  toggle: () => set((s) => ({ isOpen: !s.isOpen })),
}));
