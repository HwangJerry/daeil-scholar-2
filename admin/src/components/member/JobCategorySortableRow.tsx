// JobCategorySortableRow — draggable table row for job category list with drag handle + read-only view
import { useSortable } from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import { GripVertical, Pencil, Trash2 } from 'lucide-react';
import { Button } from '../ui/Button.tsx';
import type { AdminJobCategory } from '../../types/api.ts';

export interface JobCategorySortableRowProps {
  cat: AdminJobCategory;
  index: number;
  onEdit: () => void;
  onDelete: () => void;
  disabled: boolean;
}

export function JobCategorySortableRow({ cat, index, onEdit, onDelete, disabled }: JobCategorySortableRowProps) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: cat.seq,
    disabled,
  });

  const style: React.CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
    boxShadow: isDragging ? '0 8px 24px rgba(79, 70, 229, 0.25)' : undefined,
    position: 'relative',
    zIndex: isDragging ? 10 : undefined,
  };

  return (
    <tr ref={setNodeRef} style={style} className="border-b border-border-light hover:bg-background">
      <td className="px-2 py-3 text-center">
        <button
          type="button"
          aria-label="순서 변경 핸들"
          className={`inline-flex h-7 w-7 items-center justify-center rounded text-cool-gray ${
            disabled ? 'pointer-events-none opacity-30' : 'cursor-grab hover:bg-background active:cursor-grabbing'
          }`}
          {...attributes}
          {...listeners}
          disabled={disabled}
        >
          <GripVertical className="h-4 w-4" />
        </button>
      </td>
      <td className="px-4 py-3 text-center text-cool-gray">{index}</td>
      <td className="px-4 py-3 text-dark-slate">{cat.name}</td>
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
