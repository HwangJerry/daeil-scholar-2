// TemplateStoredText — what the server currently holds, shown beside a rejected draft
import type { NotificationTemplateView } from '../../types/notificationTemplate.ts';

interface TemplateStoredTextProps {
  template: NotificationTemplateView;
}

/**
 * After a concurrent edit the administrator needs to see what the other
 * administrator saved before deciding whether to keep their own draft, so the
 * stored text is shown next to it rather than replacing it.
 */
export function TemplateStoredText({ template }: TemplateStoredTextProps) {
  return (
    <section
      aria-label="서버 저장 문구"
      className="mt-4 rounded-xl border border-border-light bg-background p-3"
    >
      <h4 className="text-xs font-semibold text-cool-gray">
        서버 저장 문구 (버전 {template.version})
      </h4>
      {template.channel === 'push' && (
        <p className="mt-1.5 whitespace-pre-wrap break-words text-sm font-semibold text-dark-slate">
          {template.title}
        </p>
      )}
      <p className="mt-1 whitespace-pre-wrap break-words text-sm text-dark-slate">{template.body}</p>
    </section>
  );
}
