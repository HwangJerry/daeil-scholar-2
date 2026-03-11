// NotificationBell — Bell icon with unread badge dot and click-outside dropdown panel
import { useState, useRef, useEffect, useCallback } from 'react';
import { Bell } from 'lucide-react';
import { useBadges } from '../../hooks/useBadges';
import { NotificationPanel } from './NotificationPanel';

export function NotificationBell() {
  const [isOpen, setIsOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  const { unreadNotifications } = useBadges();

  const handleToggle = () => {
    setIsOpen((prev) => !prev);
  };

  const handleClose = useCallback(() => {
    setIsOpen(false);
  }, []);

  useEffect(() => {
    if (!isOpen) return;

    const handleClickOutside = (e: MouseEvent) => {
      if (
        containerRef.current &&
        !containerRef.current.contains(e.target as Node)
      ) {
        setIsOpen(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [isOpen]);

  return (
    <div ref={containerRef} className="relative">
      <button
        type="button"
        onClick={handleToggle}
        className="relative flex items-center justify-center rounded-md p-1.5 text-text-placeholder hover:text-text-primary transition-colors"
        aria-label="알림"
      >
        <Bell size={20} strokeWidth={1.8} />
        {unreadNotifications > 0 && (
          <span className="absolute top-0.5 right-0.5 h-2 w-2 rounded-full bg-error" />
        )}
      </button>

      {isOpen && <NotificationPanel onClose={handleClose} />}
    </div>
  );
}
