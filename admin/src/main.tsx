// Admin SPA entry point — mounts the React app into the DOM
import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import './index.css';
import App from './App.tsx';
import { reportFrontendError } from './lib/errorReporter.ts';

// Global JS error handler (syntax errors, unhandled exceptions outside React tree)
window.onerror = (msg, _src, _line, _col, err) => {
  reportFrontendError(
    typeof msg === 'string' ? msg : 'unhandled error',
    err?.stack ?? '',
    window.location.href,
    'window.onerror',
  );
};

// Global unhandled promise rejection handler
window.onunhandledrejection = (event: PromiseRejectionEvent) => {
  const reason = event.reason;
  const message = reason instanceof Error ? reason.message : String(reason ?? 'unhandled rejection');
  const stack = reason instanceof Error ? (reason.stack ?? '') : '';
  reportFrontendError(message, stack, window.location.href, 'window.onunhandledrejection');
};

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
