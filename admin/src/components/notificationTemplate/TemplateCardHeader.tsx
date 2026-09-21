// TemplateCardHeader — identity, status badges, and revision metadata for one template
import { Badge } from '../ui/Badge.tsx';
import type { NotificationTemplateView } from '../../types/notificationTemplate.ts';
import { formatTemplateDateTime, formatTemplateOperator } from './formatTemplateDateTime.ts';

const CHANNEL_LABELS = { sms: '문자', push: '푸시' } as const;

interface TemplateCardHeaderProps {
  template: NotificationTemplateView;
}

export function TemplateCardHeader({ template }: TemplateCardHeaderProps) {
  return (
    <header>
      <div className="flex flex-wrap items-center gap-2">
        <h3 className="font-semibold text-dark-slate">{template.displayName}</h3>
        <Badge variant="muted">{CHANNEL_LABELS[template.channel]}</Badge>
        {template.isDefault && <Badge variant="default">기본값 사용 중</Badge>}
      </div>
      <p className="mt-1.5 text-sm text-cool-gray">{template.description}</p>

      {template.invalid && (
        <p role="alert" className="mt-2 rounded-xl bg-error-subtle px-3 py-2 text-sm text-error-text">
          저장된 문구가 유효하지 않아 기본 문구로 발송 중입니다.
        </p>
      )}

      <p className="mt-2 text-xs text-cool-gray">
        버전 {template.version} · 최종 수정 {formatTemplateDateTime(template.updatedAt)}
        {template.updatedAt ? ` · ${formatTemplateOperator(template.updatedBy)}` : ''}
      </p>
    </header>
  );
}
