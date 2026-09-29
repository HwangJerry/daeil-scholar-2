// FeedCategorySortableRow — draggable feed category row: name + 기본 badge, post count, app tab switch, edit/delete
import type { CSSProperties } from 'react';
import { useSortable } from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import { GripVertical, Pencil, Trash2 } from 'lucide-react';
import { Badge } from '../ui/Badge.tsx';
import { Button } from '../ui/Button.tsx';
import { AppTabSwitch } from './AppTabSwitch.tsx';
import type { AdminFeedCategory } from '../../types/api.ts';

export const DEFAULT_DELETE_LABEL = '기본 카테고리는 삭제할 수 없습니다';
const DEFAULT_HIDE_LABEL = '기본 카테고리는 숨길 수 없습니다';

export interface FeedCategorySortableRowProps {
  cat: AdminFeedCategory;
  index: number;
  onEdit: () => void;
  onDelete: () => void;
  onToggleOpen: (open: boolean) => void;
  disabled: boolean;
}

export function FeedCategorySortableRow({ cat, index, onEdit, onDelete, onToggleOpen, disabled }: FeedCategorySortableRowProps) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: cat.seq,
    disabled,
  });
  const isDefault = cat.isDefault === 'Y';

  const style: CSSProperties = {
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
      <td className="px-4 py-3 text-dark-slate">
        <span className="inline-flex items-center gap-2">
          {cat.name}
          {isDefault && <Badge variant="default">기본</Badge>}
        </span>
      </td>
      <td className="px-4 py-3 text-center text-cool-gray">{cat.postCount}</td>
      <td className="px-4 py-3 text-center">
        <AppTabSwitch
          checked={cat.openYn === 'Y'}
          onChange={onToggleOpen}
          disabled={disabled || isDefault}
          label={`${cat.name} 앱 탭 노출`}
          title={isDefault ? DEFAULT_HIDE_LABEL : undefined}
        />
      </td>
      <td className="px-4 py-3 text-center">
        <div className="flex justify-center gap-1">
          <Button variant="ghost" size="icon" onClick={onEdit} disabled={disabled} aria-label="수정">
            <Pencil className="h-4 w-4" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            onClick={onDelete}
            disabled={disabled || isDefault}
            aria-label={isDefault ? DEFAULT_DELETE_LABEL : '삭제'}
            title={isDefault ? DEFAULT_DELETE_LABEL : undefined}
          >
            <Trash2 className="h-4 w-4 text-error-text" />
          </Button>
        </div>
      </td>
    </tr>
  );
}
