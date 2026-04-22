// JobCategoryInlineEditRow — inline edit table row for job category create/update (order managed separately via DnD)
import { Check, X } from 'lucide-react';
import { Button } from '../ui/Button.tsx';
import { Input } from '../ui/Input.tsx';
import type { AdminJobCategoryUpsert } from '../../types/api.ts';

export interface JobCategoryInlineEditRowProps {
  draft: AdminJobCategoryUpsert;
  onChange: (patch: Partial<AdminJobCategoryUpsert>) => void;
  onSave: () => void;
  onCancel: () => void;
  isSaving: boolean;
}

export function JobCategoryInlineEditRow({ draft, onChange, onSave, onCancel, isSaving }: JobCategoryInlineEditRowProps) {
  return (
    <tr className="border-b border-border-light bg-background">
      <td className="px-2 py-2 text-center text-cool-gray">—</td>
      <td className="px-4 py-2 text-center text-cool-gray">—</td>
      <td className="px-4 py-2">
        <Input
          value={draft.name}
          onChange={(e) => onChange({ name: e.target.value })}
          placeholder="카테고리 이름"
          className="h-8 text-sm"
          disabled={isSaving}
          autoFocus
        />
      </td>
      <td className="px-4 py-2">
        <select
          value={draft.openYn}
          onChange={(e) => onChange({ openYn: e.target.value as 'Y' | 'N' })}
          className="h-8 w-full rounded-xl border border-border bg-white px-2 text-sm focus:outline-none focus:ring-2 focus:ring-royal-indigo"
          disabled={isSaving}
        >
          <option value="Y">공개</option>
          <option value="N">비공개</option>
        </select>
      </td>
      <td className="px-4 py-2 text-center">
        <div className="flex justify-center gap-1">
          <Button variant="ghost" size="icon" onClick={onSave} disabled={isSaving} aria-label="저장">
            <Check className="h-4 w-4 text-success-text" />
          </Button>
          <Button variant="ghost" size="icon" onClick={onCancel} disabled={isSaving} aria-label="취소">
            <X className="h-4 w-4 text-cool-gray" />
          </Button>
        </div>
      </td>
    </tr>
  );
}
