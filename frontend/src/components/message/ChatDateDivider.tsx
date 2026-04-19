// ChatDateDivider — Horizontal date label between message groups
export function ChatDateDivider({ label }: { label: string }) {
  return (
    <div className="flex items-center gap-3 my-4">
      <div className="flex-1 border-t border-border" />
      <span className="text-[11px] text-text-tertiary shrink-0">{label}</span>
      <div className="flex-1 border-t border-border" />
    </div>
  );
}
