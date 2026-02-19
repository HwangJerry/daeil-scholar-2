// useConfirmDialog — manages open/close state and pending value for confirmation dialogs
import { useState, useCallback } from 'react';

export function useConfirmDialog<T = void>() {
  const [isOpen, setIsOpen] = useState(false);
  const [pendingValue, setPendingValue] = useState<T | null>(null);

  const open = useCallback((value: T) => {
    setPendingValue(value);
    setIsOpen(true);
  }, []);

  const close = useCallback(() => {
    setIsOpen(false);
    setPendingValue(null);
  }, []);

  const confirm = useCallback(() => {
    const value = pendingValue;
    setIsOpen(false);
    setPendingValue(null);
    return value;
  }, [pendingValue]);

  return { isOpen, pendingValue, open, close, confirm };
}
