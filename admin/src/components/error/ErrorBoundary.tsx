// ErrorBoundary — catches unhandled React errors and renders a fallback UI
import { Component, type ReactNode } from 'react';
import { AlertCircle } from 'lucide-react';

interface Props {
  children: ReactNode;
}

interface State {
  hasError: boolean;
}

export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError(): State {
    return { hasError: true };
  }

  render() {
    if (!this.state.hasError) {
      return this.props.children;
    }

    return (
      <div className="flex min-h-screen items-center justify-center bg-background p-6">
        <div className="w-full max-w-md rounded-2xl border border-error-border bg-error-subtle p-8 text-center shadow-sm">
          <AlertCircle className="mx-auto h-12 w-12 text-error-text" />
          <h2 className="mt-4 text-lg font-bold text-error-text">
            예기치 않은 오류가 발생했습니다
          </h2>
          <p className="mt-2 text-sm text-error-text/80">
            문제가 지속되면 관리자에게 문의해 주세요.
          </p>
          <div className="mt-6 flex flex-col items-center gap-3">
            <button
              onClick={() => window.location.reload()}
              className="rounded-xl bg-error-text px-6 py-2.5 text-sm font-medium text-white hover:bg-error-text/90"
            >
              새로고침
            </button>
            <a
              href="/admin"
              className="text-sm text-error-text underline-offset-4 hover:underline"
            >
              대시보드로 이동
            </a>
          </div>
        </div>
      </div>
    );
  }
}
