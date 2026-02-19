// ErrorState — reusable error display with retry button, supports table row wrapping
import { AlertCircle } from 'lucide-react';
import { Button } from './Button.tsx';

interface ErrorStateProps {
  message?: string;
  onRetry?: () => void;
  colSpan?: number;
}

function ErrorContent({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <div className="flex flex-col items-center gap-3 py-8">
      <AlertCircle className="h-8 w-8 text-error-text" />
      <p className="text-sm text-error-text">{message}</p>
      {onRetry && (
        <Button variant="outline" size="sm" onClick={onRetry}>
          다시 시도
        </Button>
      )}
    </div>
  );
}

export function ErrorState({ message = '데이터를 불러오는 데 실패했습니다.', onRetry, colSpan }: ErrorStateProps) {
  if (colSpan != null) {
    return (
      <tr>
        <td colSpan={colSpan}>
          <ErrorContent message={message} onRetry={onRetry} />
        </td>
      </tr>
    );
  }

  return <ErrorContent message={message} onRetry={onRetry} />;
}
