// formatFileSize — formats a byte-string ("12345") into a human-readable size like "12.1KB"
export function formatFileSize(size: string): string {
  const bytes = Number(size);
  if (Number.isNaN(bytes)) return size;
  if (bytes < 1024) return `${bytes}B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)}KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)}MB`;
}
