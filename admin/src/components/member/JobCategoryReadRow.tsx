// JobCategoryReadRow — read-only table row for job category list
import { Pencil, Trash2 } from 'lucide-react';
import { Button } from '../ui/Button.tsx';
import type { AdminJobCategory } from '../../types/api.ts';

export interface JobCategoryReadRowProps {
  cat: AdminJobCategory;
  index: number;
  onEdit: () => void;
  onDelete: () => void;
  disabled: boolean;
}

export function JobCategoryReadRow({ cat, index, onEdit, onDelete, disabled }: JobCategoryReadRowProps) {
  return (
    <tr className="border-b border-border-light hover:bg-background">
      <td className="px-4 py-3 text-center text-cool-gray">{index}</td>
      <td className="px-4 py-3 text-dark-slate">{cat.name}</td>
      <td className="px-4 py-3 text-center text-cool-gray">{cat.index}</td>
      <td className="px-4 py-3 text-center">
        <span className={cat.openYn === 'Y' ? 'text-success-text' : 'text-cool-gray'}>
          {cat.openYn === 'Y' ? '공개' : '비공개'}
        </span>
      </td>
      <td className="px-4 py-3 text-center">
        <div className="flex justify-center gap-1">
          <Button variant="ghost" size="icon" onClick={onEdit} disabled={disabled} aria-label="편집">
            <Pencil className="h-4 w-4" />
          </Button>
          <Button variant="ghost" size="icon" onClick={onDelete} disabled={disabled} aria-label="삭제">
            <Trash2 className="h-4 w-4 text-error-text" />
          </Button>
        </div>
      </td>
    </tr>
  );
}
