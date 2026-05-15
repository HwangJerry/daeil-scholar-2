// errorReporter — forwards frontend JS errors to the backend debug-agent relay endpoint.
// Fire-and-forget: errors in the reporter itself are silently swallowed to avoid recursion.

export function reportFrontendError(
  message: string,
  stack: string,
  url: string,
  component: string,
): void {
  try {
    fetch('/api/admin/errors/report', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ message, stack, url, component }),
      // keepalive so the request completes even if the page is unloading
      keepalive: true,
    }).catch(() => {/* ignore network errors */});
  } catch {
    // noop — never throw from an error reporter
  }
}
