// TemplateCard — one notification text's form, with live counters, preview, and save
import { useEffect, useRef, useState, type FormEvent } from 'react';
import { Button } from '../ui/Button.tsx';
import { useSaveNotificationTemplate } from '../../hooks/useSaveNotificationTemplate.ts';
import {
  isTemplateConflict,
  templateCardMessage,
  templateFieldErrors,
} from '../../lib/notificationTemplateErrors.ts';
import { insertPlaceholder } from '../../lib/notificationTemplateText.ts';
import { validateTemplateDraft } from '../../lib/notificationTemplateValidation.ts';
import type {
  NotificationTemplateDraft,
  NotificationTemplateView,
} from '../../types/notificationTemplate.ts';
import { TemplateCardHeader } from './TemplateCardHeader.tsx';
import { TemplateFields, type EditableField } from './TemplateFields.tsx';
import { TemplatePreview } from './TemplatePreview.tsx';
import { TemplateStoredText } from './TemplateStoredText.tsx';

// A row the migration has not created yet has no version to send back, and the
// server rejects the write outright, so the form is shown but cannot be saved.
const MISSING_ROW_MESSAGE =
  '서버에 저장 행이 없어 수정할 수 없습니다. 마이그레이션 적용 여부를 확인해 주세요.';

interface TemplateCardProps {
  template: NotificationTemplateView;
  /** Refetches the list after a concurrent edit was reported. */
  onReload: () => void;
}

export function TemplateCard({ template, onReload }: TemplateCardProps) {
  const storedText = { title: template.title, body: template.body };
  const [draft, setDraft] = useState<NotificationTemplateDraft>(storedText);
  // The chips insert into whichever field was last edited; the body is what an
  // administrator fills in first, so it is the default target.
  const [focusedField, setFocusedField] = useState<EditableField>('body');
  const titleRef = useRef<HTMLInputElement>(null);
  const bodyRef = useRef<HTMLTextAreaElement>(null);
  const previousStored = useRef(storedText);
  const saveTemplate = useSaveNotificationTemplate(template.displayName);

  // An untouched card follows the server, so another administrator's change
  // shows up on the next read; a card being edited keeps its draft, because
  // discarding typing the administrator has not saved is never the right move.
  useEffect(() => {
    const previous = previousStored.current;
    previousStored.current = { title: template.title, body: template.body };
    setDraft((current) =>
      current.title === previous.title && current.body === previous.body
        ? { title: template.title, body: template.body }
        : current,
    );
  }, [template.title, template.body]);

  const isPush = template.channel === 'push';
  const isPending = saveTemplate.isPending;
  // An SMS carries no title, so what gets validated is what gets sent: a title
  // left in the stored row must not make the card permanently unsaveable.
  const submission = { title: isPush ? draft.title : '', body: draft.body };
  const draftErrors = validateTemplateDraft(template, submission);
  const serverErrors = templateFieldErrors(saveTemplate.error);
  const errors = {
    title: draftErrors.title ?? serverErrors.title,
    body: draftErrors.body ?? serverErrors.body,
  };
  const cardMessage = templateCardMessage(saveTemplate.error);
  const hasConflict = isTemplateConflict(saveTemplate.error);
  const hasStoredVersion = template.version > 0;
  const isDirty = draft.title !== template.title || draft.body !== template.body;
  const canSave =
    isDirty && hasStoredVersion && !draftErrors.title && !draftErrors.body && !isPending;

  const updateDraft = (changes: Partial<NotificationTemplateDraft>) => {
    if (saveTemplate.isError) saveTemplate.reset();
    setDraft((current) => ({ ...current, ...changes }));
  };

  const handleInsertPlaceholder = (name: string) => {
    const target = focusedField === 'title' && isPush ? 'title' : 'body';
    const element = target === 'title' ? titleRef.current : bodyRef.current;
    const text = draft[target];
    const { text: nextText, caret } = insertPlaceholder(
      text,
      name,
      element?.selectionStart ?? text.length,
      element?.selectionEnd ?? text.length,
    );
    updateDraft({ [target]: nextText });
    // The caret has to be restored after React writes the new value back.
    window.requestAnimationFrame(() => {
      element?.focus();
      element?.setSelectionRange(caret, caret);
    });
  };

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!canSave) return;
    saveTemplate.mutate(
      { key: template.key, ...submission, expectedVersion: template.version },
      // The card is not remounted on save, so the draft is what clears the
      // dirty state: it adopts exactly the text the server echoed back.
      { onSuccess: (saved) => setDraft({ title: saved.title, body: saved.body }) },
    );
  };

  return (
    <article className="rounded-2xl border border-border-light bg-surface p-5 shadow-sm md:p-6">
      <TemplateCardHeader template={template} />

      <form className="mt-4" onSubmit={handleSubmit}>
        <TemplateFields
          template={template}
          draft={draft}
          errors={errors}
          disabled={isPending}
          titleRef={titleRef}
          bodyRef={bodyRef}
          onChange={updateDraft}
          onFocusField={setFocusedField}
          onInsertPlaceholder={handleInsertPlaceholder}
        />

        <TemplatePreview
          channel={template.channel}
          title={submission.title}
          body={submission.body}
          placeholders={template.placeholders}
        />

        {hasConflict && <TemplateStoredText template={template} />}

        {!hasStoredVersion && (
          <p role="alert" className="mt-4 text-sm text-error-text">
            {MISSING_ROW_MESSAGE}
          </p>
        )}
        {cardMessage && (
          <p role="alert" className="mt-4 text-sm text-error-text">
            {cardMessage}
          </p>
        )}

        <div className="mt-4 flex flex-wrap items-center gap-2">
          <Button type="submit" disabled={!canSave}>
            저장
          </Button>
          <Button
            type="button"
            variant="outline"
            disabled={isPending}
            onClick={() => updateDraft({ title: template.defaultTitle, body: template.defaultBody })}
          >
            기본값으로 되돌리기
          </Button>
          <Button
            type="button"
            variant="ghost"
            disabled={isPending || !isDirty}
            onClick={() => updateDraft(storedText)}
          >
            변경 취소
          </Button>
          {hasConflict && (
            <Button type="button" variant="outline" onClick={onReload}>
              새로고침
            </Button>
          )}
        </div>
      </form>
    </article>
  );
}
