// WipGate — Temporary "Work in Progress" overlay gating the app behind an admin code
import { useRef, useState, type FormEvent } from 'react';
import { Card } from '../ui/Card';
import { Input } from '../ui/Input';
import { Button } from '../ui/Button';

const STORAGE_KEY = 'wip-unlock';
const EXPECTED_CODE = (import.meta.env.VITE_WIP_ADMIN_CODE ?? '') as string;

function readUnlocked(): boolean {
  if (!EXPECTED_CODE) return true;
  try {
    return sessionStorage.getItem(STORAGE_KEY) === '1';
  } catch {
    return false;
  }
}

function persistUnlocked() {
  try {
    sessionStorage.setItem(STORAGE_KEY, '1');
  } catch {
    // ignore (Safari private mode, storage disabled, etc.)
  }
}

export function WipGate({ children }: { children: React.ReactNode }) {
  const [unlocked, setUnlocked] = useState<boolean>(readUnlocked);
  const [code, setCode] = useState('');
  const [error, setError] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  if (unlocked) return <>{children}</>;

  const handleSubmit = (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (code === EXPECTED_CODE) {
      persistUnlocked();
      setUnlocked(true);
      return;
    }
    setError(true);
    requestAnimationFrame(() => {
      inputRef.current?.focus();
      inputRef.current?.select();
    });
  };

  return (
    <div className="fixed inset-0 z-[100] flex items-center justify-center bg-background p-4 animate-fade-in">
      <Card variant="elevated" padding="lg" className="w-full max-w-md animate-fade-in-up">
        <h1 className="text-2xl font-semibold text-text-primary">사이트 점검 중</h1>
        <p className="mt-3 text-sm text-text-secondary leading-relaxed">
          현재 사이트를 새롭게 단장하고 있습니다. 잠시만 기다려주세요.
        </p>
        <form onSubmit={handleSubmit} className="mt-6 space-y-3">
          <label htmlFor="wip-code" className="block text-sm text-text-secondary">
            관리자 코드
          </label>
          <Input
            id="wip-code"
            ref={inputRef}
            type="password"
            autoComplete="off"
            autoFocus
            value={code}
            onChange={(e) => {
              setCode(e.target.value);
              if (error) setError(false);
            }}
            aria-invalid={error}
            aria-describedby={error ? 'wip-code-error' : undefined}
          />
          {error && (
            <p id="wip-code-error" role="alert" className="text-sm text-error-text">
              코드가 올바르지 않습니다.
            </p>
          )}
          <Button type="submit" className="w-full">
            입장하기
          </Button>
        </form>
      </Card>
    </div>
  );
}
