// FeedCategoryInlineEditRow — inline create/rename row for a feed category (order is managed via drag)
import { Check, X } from 'lucide-react';
import { Button } from '../ui/Button.tsx';
import { Input } from '../ui/Input.tsx';
import { AppTabSwitch } from './AppTabSwitch.tsx';
import type { AdminFeedCategoryUpsert } from '../../types/api.ts';

export interface FeedCategoryInlineEditRowProps {
  draft: AdminFeedCategoryUpsert;
  isDefault: boolean;
  postCount: number | null;
  onChange: (patch: Partial<AdminFeedCategoryUpsert>) => void;
  onSave: () => void;
  onCancel: () => void;
  isSaving: boolean;
}

export function FeedCategoryInlineEditRow({ draft, isDefault, postCount, onChange, onSave, onCancel, isSaving }: FeedCategoryInlineEditRowProps) {
  return (
    <tr className="border-b border-border-light bg-background">
      <td className="px-2 py-2 text-center text-cool-gray">—</td>
      <td className="px-4 py-2 text-center text-cool-gray">—</td>
      <td className="px-4 py-2">
        <Input
          aria-label="카테고리 이름"
          value={draft.name}
          onChange={(e) => onChange({ name: e.target.value })}
          onKeyDown={(e) => {
            if (e.key === 'Enter') onSave();
            if (e.key === 'Escape') onCancel();
          }}
          placeholder="카테고리 이름 (1~8자)"
          className="h-8 text-sm"
          disabled={isSaving}
          autoFocus
        />
      </td>
      <td className="px-4 py-2 text-center text-cool-gray">{postCount ?? '—'}</td>
      <td className="px-4 py-2 text-center">
        <AppTabSwitch
          checked={draft.openYn === 'Y'}
          onChange={(open) => onChange({ openYn: open ? 'Y' : 'N' })}
          disabled={isSaving || isDefault}
          label="앱 탭 노출"
        />
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
