// TemplateFields — the title and body inputs of a template card, with their counters and chips
import type { RefObject } from 'react';
import { Input } from '../ui/Input.tsx';
import { Textarea } from '../ui/Textarea.tsx';
import { runeLength } from '../../lib/notificationTemplateText.ts';
import {
  pushBodyLimit,
  pushTitleLimit,
  smsBodyBytes,
  smsByteLimit,
} from '../../lib/notificationTemplateValidation.ts';
import type {
  NotificationTemplateDraft,
  NotificationTemplateFieldErrors,
  NotificationTemplateView,
} from '../../types/notificationTemplate.ts';
import { PlaceholderChips } from './PlaceholderChips.tsx';
import { TemplateCounter } from './TemplateCounters.tsx';

const BODY_ROWS = 4;

export type EditableField = 'title' | 'body';

interface TemplateFieldsProps {
  template: NotificationTemplateView;
  draft: NotificationTemplateDraft;
  errors: NotificationTemplateFieldErrors;
  disabled: boolean;
  titleRef: RefObject<HTMLInputElement | null>;
  bodyRef: RefObject<HTMLTextAreaElement | null>;
  onChange: (changes: Partial<NotificationTemplateDraft>) => void;
  onFocusField: (field: EditableField) => void;
  onInsertPlaceholder: (name: string) => void;
}

export function TemplateFields({
  template,
  draft,
  errors,
  disabled,
  titleRef,
  bodyRef,
  onChange,
  onFocusField,
  onInsertPlaceholder,
}: TemplateFieldsProps) {
  const isPush = template.channel === 'push';
  const fieldId = (field: EditableField) => `${template.key}-${field}`;

  return (
    <fieldset disabled={disabled} className="space-y-4">
      {isPush && (
        <div>
          <FieldLabel htmlFor={fieldId('title')} label="제목">
            <TemplateCounter
              label="제목"
              used={runeLength(draft.title)}
              limit={pushTitleLimit(template)}
            />
          </FieldLabel>
          <Input
            id={fieldId('title')}
            ref={titleRef}
            value={draft.title}
            aria-invalid={errors.title ? true : undefined}
            onFocus={() => onFocusField('title')}
            onChange={(event) => onChange({ title: event.target.value })}
          />
          <FieldError message={errors.title} />
        </div>
      )}

      <div>
        <FieldLabel htmlFor={fieldId('body')} label="본문">
          {isPush ? (
            <TemplateCounter
              label="본문"
              used={runeLength(draft.body)}
              limit={pushBodyLimit(template)}
            />
          ) : (
            <TemplateCounter
              label="본문"
              used={smsBodyBytes(template, draft.body)}
              limit={smsByteLimit(template)}
              unit="바이트"
            />
          )}
        </FieldLabel>
        <Textarea
          id={fieldId('body')}
          ref={bodyRef}
          rows={BODY_ROWS}
          value={draft.body}
          aria-invalid={errors.body ? true : undefined}
          onFocus={() => onFocusField('body')}
          onChange={(event) => onChange({ body: event.target.value })}
        />
        <FieldError message={errors.body} />
        <PlaceholderChips
          placeholders={template.placeholders}
          disabled={disabled}
          onInsert={onInsertPlaceholder}
        />
      </div>
    </fieldset>
  );
}

function FieldLabel({
  htmlFor,
  label,
  children,
}: {
  htmlFor: string;
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="mb-1.5 flex items-center justify-between gap-2">
      <label htmlFor={htmlFor} className="text-sm font-medium text-dark-slate">
        {label}
      </label>
      {children}
    </div>
  );
}

/** Several reasons for one field arrive newline-separated, so they stack. */
function FieldError({ message }: { message?: string }) {
  if (!message) return null;
  return (
    <p role="alert" className="mt-1.5 whitespace-pre-line text-sm text-error-text">
      {message}
    </p>
  );
}
