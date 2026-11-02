// HallOfFameEntryDialog — modal form for adding or editing a Hall of Fame entry
import { useState, type FormEvent } from 'react';
import * as Dialog from '@radix-ui/react-dialog';
import { Loader2 } from 'lucide-react';
import {
  HALL_KINDS,
  HALL_TIERS,
  type HallEntry,
  type HallEntryInput,
  type HallKind,
} from '../../types/hallOfFame.ts';
import { Button } from '../ui/Button.tsx';
import { Input } from '../ui/Input.tsx';
import { Select } from '../ui/Select.tsx';
import {
  createHallEntryFormState,
  parseHallEntryForm,
  type HallEntryFormState,
} from './hallOfFameEntryForm.ts';
import {
  formatTier,
  HALL_KIND_LABELS,
  ROLE_LABEL_FIELD_LABEL,
  UNLISTED_TIER_LABEL,
} from './hallOfFameLabels.ts';

interface HallOfFameEntryDialogProps {
  entry: HallEntry | null;
  isSubmitting: boolean;
  onClose: () => void;
  onSubmit: (input: HallEntryInput) => void;
}

/** Mount with a `key` per target so the form state starts from that entry. */
export function HallOfFameEntryDialog({
  entry,
  isSubmitting,
  onClose,
  onSubmit,
}: HallOfFameEntryDialogProps) {
  const [form, setForm] = useState(() => createHallEntryFormState(entry));
  const [formError, setFormError] = useState<string | null>(null);
  const isEditing = entry !== null;

  const setField = <K extends keyof HallEntryFormState>(field: K, value: HallEntryFormState[K]) => {
    setForm((current) => ({ ...current, [field]: value }));
  };

  const handleSubmit = (event: FormEvent) => {
    event.preventDefault();
    const result = parseHallEntryForm(form);
    if (!result.ok) {
      setFormError(result.reason);
      return;
    }
    setFormError(null);
    onSubmit(result.input);
  };

  return (
    <Dialog.Root open onOpenChange={(open) => { if (!open && !isSubmitting) onClose(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-50 bg-black/40" />
        <Dialog.Content className="fixed left-1/2 top-1/2 z-50 max-h-[90vh] w-[calc(100%-2rem)] max-w-lg -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-2xl bg-white p-6 shadow-xl">
          <Dialog.Title className="text-lg font-bold text-dark-slate">
            {isEditing ? '명예의 전당 항목 수정' : '명예의 전당 항목 추가'}
          </Dialog.Title>
          <Dialog.Description className="mt-1 text-sm text-cool-gray">
            전화번호는 입력하거나 저장하지 않습니다.
          </Dialog.Description>

          <form className="mt-5 space-y-4" noValidate onSubmit={handleSubmit}>
            <FormField id="hall-entry-name" label="이름">
              <Input
                id="hall-entry-name"
                value={form.displayName}
                onChange={(event) => setField('displayName', event.target.value)}
              />
            </FormField>

            <div className="grid grid-cols-2 gap-3">
              <FormField id="hall-entry-kind" label="종류">
                <Select
                  id="hall-entry-kind"
                  value={form.kind}
                  onChange={(event) => setField('kind', event.target.value as HallKind)}
                >
                  {HALL_KINDS.map((kind) => (
                    <option key={kind} value={kind}>{HALL_KIND_LABELS[kind]}</option>
                  ))}
                </Select>
              </FormField>
              <FormField id="hall-entry-tier" label="등급">
                <Select
                  id="hall-entry-tier"
                  value={form.tierManWon}
                  onChange={(event) => setField('tierManWon', event.target.value)}
                >
                  <option value="">{UNLISTED_TIER_LABEL}</option>
                  {HALL_TIERS.map((tier) => (
                    <option key={tier} value={tier}>{formatTier(tier)}</option>
                  ))}
                </Select>
              </FormField>
              <FormField id="hall-entry-cohort" label="기수">
                <Input
                  id="hall-entry-cohort"
                  inputMode="numeric"
                  value={form.cohort}
                  onChange={(event) => setField('cohort', event.target.value)}
                />
              </FormField>
              <FormField id="hall-entry-department" label="과">
                <Input
                  id="hall-entry-department"
                  value={form.department}
                  onChange={(event) => setField('department', event.target.value)}
                />
              </FormField>
            </div>

            {form.kind === 'org' && (
              <FormField id="hall-entry-contact" label="담당자 (비공개)">
                <Input
                  id="hall-entry-contact"
                  value={form.contactName}
                  onChange={(event) => setField('contactName', event.target.value)}
                />
              </FormField>
            )}

            {form.kind === 'teacher' && (
              <FormField id="hall-entry-role-label" label={ROLE_LABEL_FIELD_LABEL}>
                <Input
                  id="hall-entry-role-label"
                  value={form.roleLabel}
                  onChange={(event) => setField('roleLabel', event.target.value)}
                />
              </FormField>
            )}

            {entry?.source === 'excel' && (
              <p className="text-xs text-cool-gray">
                수정한 항목은 이후 엑셀 가져오기가 덮어쓰지 않습니다.
              </p>
            )}
            {formError && <p role="alert" className="text-sm text-error-text">{formError}</p>}

            <div className="flex justify-end gap-3">
              <Dialog.Close asChild>
                <Button type="button" variant="outline" disabled={isSubmitting}>취소</Button>
              </Dialog.Close>
              <Button type="submit" disabled={isSubmitting}>
                {isSubmitting && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                {isEditing ? '수정 저장' : '추가'}
              </Button>
            </div>
          </form>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}

function FormField({ id, label, children }: { id: string; label: string; children: React.ReactNode }) {
  return (
    <div className="space-y-1.5">
      <label htmlFor={id} className="block text-sm font-medium text-dark-slate">{label}</label>
      {children}
    </div>
  );
}
