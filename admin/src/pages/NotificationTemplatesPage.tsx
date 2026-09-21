// NotificationTemplatesPage — administration of the SMS and push notification texts
import { MessageSquareText, RefreshCw } from 'lucide-react';
import { ErrorState } from '../components/ui/ErrorState.tsx';
import { TemplateCard } from '../components/notificationTemplate/TemplateCard.tsx';
import { useNotificationTemplates } from '../hooks/useNotificationTemplates.ts';
import type {
  NotificationChannel,
  NotificationTemplateView,
} from '../types/notificationTemplate.ts';

const CHANNEL_SECTIONS = [
  { channel: 'sms', label: '문자 (SMS)' },
  { channel: 'push', label: '푸시 알림' },
] as const satisfies readonly { channel: NotificationChannel; label: string }[];

interface ChannelSectionProps {
  channel: NotificationChannel;
  label: string;
  templates: NotificationTemplateView[];
  onReload: () => void;
}

export function NotificationTemplatesPage() {
  const templatesQuery = useNotificationTemplates();
  const templates = templatesQuery.data ?? [];
  const reload = () => void templatesQuery.refetch();

  return (
    <div className="space-y-6">
      <header>
        <div className="flex items-center gap-2">
          <MessageSquareText aria-hidden="true" className="h-5 w-5 text-royal-indigo" />
          <h2 className="text-xl font-bold text-dark-slate">알림 문구 관리</h2>
        </div>
        <p className="mt-2 text-sm text-cool-gray">
          회원에게 발송되는 문자와 푸시 알림의 문구를 수정합니다. 저장하면 다음 발송부터 바로
          적용되며, 치환 항목은 회원별 값으로 바뀝니다.
        </p>
      </header>

      <TemplateSections
        query={templatesQuery}
        templates={templates}
        onReload={reload}
      />
    </div>
  );
}

interface TemplateSectionsProps {
  query: ReturnType<typeof useNotificationTemplates>;
  templates: NotificationTemplateView[];
  onReload: () => void;
}

function TemplateSections({ query, templates, onReload }: TemplateSectionsProps) {
  if (query.isLoading) return <TemplatesLoading />;
  if (query.isError) return <TemplatesError onRetry={onReload} />;

  return (
    <>
      {CHANNEL_SECTIONS.map((section) => (
        <ChannelSection
          key={section.channel}
          channel={section.channel}
          label={section.label}
          templates={templates.filter((template) => template.channel === section.channel)}
          onReload={onReload}
        />
      ))}
    </>
  );
}

function TemplatesLoading() {
  return (
    <div className="rounded-2xl border border-border-light bg-surface p-6 shadow-sm">
      <div className="flex items-center gap-2 text-sm text-cool-gray">
        <RefreshCw aria-hidden="true" className="h-4 w-4" />
        알림 문구를 불러오는 중입니다.
      </div>
    </div>
  );
}

function TemplatesError({ onRetry }: { onRetry: () => void }) {
  return (
    <div className="rounded-2xl border border-border-light bg-surface shadow-sm">
      <ErrorState message="알림 문구를 불러오는 데 실패했습니다." onRetry={onRetry} />
    </div>
  );
}

function ChannelSection({ channel, label, templates, onReload }: ChannelSectionProps) {
  const sectionId = `notification-templates-${channel}`;

  return (
    <section aria-labelledby={sectionId}>
      <h3 id={sectionId} className="mb-3 text-base font-semibold text-dark-slate">
        {label}
      </h3>
      {templates.length === 0 ? (
        <p className="rounded-2xl border border-border-light bg-surface p-5 text-sm text-cool-gray shadow-sm">
          이 채널에 등록된 문구가 없습니다.
        </p>
      ) : (
        <div className="grid grid-cols-1 gap-4 xl:grid-cols-2">
          {templates.map((template) => (
            // Keyed by key alone: a card must survive a version change with the
            // administrator's unsaved draft and focus intact.
            <TemplateCard key={template.key} template={template} onReload={onReload} />
          ))}
        </div>
      )}
    </section>
  );
}
