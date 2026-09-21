// TemplatePreview — shows the draft as a member would receive it, samples filled in
import { renderWithSamples } from '../../lib/notificationTemplateText.ts';
import type {
  NotificationChannel,
  NotificationTemplatePlaceholder,
} from '../../types/notificationTemplate.ts';

interface TemplatePreviewProps {
  channel: NotificationChannel;
  title: string;
  body: string;
  placeholders: NotificationTemplatePlaceholder[];
}

export function TemplatePreview({ channel, title, body, placeholders }: TemplatePreviewProps) {
  const previewTitle = renderWithSamples(title, placeholders);
  const previewBody = renderWithSamples(body, placeholders);

  return (
    <section aria-label="미리보기" className="mt-4">
      <h4 className="mb-1.5 text-xs font-semibold text-cool-gray">미리보기</h4>
      {channel === 'sms' ? (
        <SmsPreviewBubble body={previewBody} />
      ) : (
        <PushPreviewCard title={previewTitle} body={previewBody} />
      )}
      <p className="mt-1.5 text-xs text-cool-gray">
        치환 항목은 예시 값으로 채워 보여 줍니다. 실제 발송 시에는 회원별 값으로 바뀝니다.
      </p>
    </section>
  );
}

function SmsPreviewBubble({ body }: { body: string }) {
  return (
    <div className="rounded-2xl bg-background p-3">
      <p
        data-testid="template-preview-body"
        className="max-w-[85%] whitespace-pre-wrap break-words rounded-2xl rounded-bl-sm bg-white px-3 py-2 text-sm text-dark-slate shadow-sm"
      >
        {body}
      </p>
    </div>
  );
}

function PushPreviewCard({ title, body }: { title: string; body: string }) {
  return (
    <div className="rounded-2xl bg-background p-3">
      <div className="rounded-xl bg-white px-3 py-2.5 shadow-sm">
        <p
          data-testid="template-preview-title"
          className="whitespace-pre-wrap break-words text-sm font-bold text-dark-slate"
        >
          {title}
        </p>
        <p
          data-testid="template-preview-body"
          className="mt-0.5 whitespace-pre-wrap break-words text-sm text-cool-gray"
        >
          {body}
        </p>
      </div>
    </div>
  );
}
