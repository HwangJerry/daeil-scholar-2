// isBlockedAttachment — client-side guard against executable file uploads
export const BLOCKED_ATTACHMENT_EXTS: readonly string[] = [
  'exe', 'sh', 'bat', 'js', 'php', 'com', 'scr', 'vbs', 'cmd', 'jar',
];

export function isBlockedAttachment(file: File): boolean {
  const dot = file.name.lastIndexOf('.');
  if (dot < 0) return false;
  const ext = file.name.slice(dot + 1).toLowerCase();
  return BLOCKED_ATTACHMENT_EXTS.includes(ext);
}
