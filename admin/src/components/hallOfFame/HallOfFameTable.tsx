// HallOfFameTable — entry rows with visibility and consent switches plus edit/delete actions
import { Pencil, Trash2 } from 'lucide-react';
import { cn } from '../../lib/utils.ts';
import type { HallEntry } from '../../types/hallOfFame.ts';
import { Badge } from '../ui/Badge.tsx';
import { Button } from '../ui/Button.tsx';
import { EMPTY_VALUE_LABEL, formatTier, HALL_KIND_LABELS, HALL_SOURCE_LABELS } from './hallOfFameLabels.ts';

interface HallOfFameTableProps {
  entries: HallEntry[];
  isUpdating: boolean;
  onVisibleChange: (entry: HallEntry, visible: boolean) => void;
  onConsentChange: (entry: HallEntry, consentPublic: boolean) => void;
  onEdit: (entry: HallEntry) => void;
  onDelete: (entry: HallEntry) => void;
}

export function HallOfFameTable({
  entries,
  isUpdating,
  onVisibleChange,
  onConsentChange,
  onEdit,
  onDelete,
}: HallOfFameTableProps) {
  return (
    <div className="overflow-x-auto rounded-xl border border-border-light">
      <table className="w-full min-w-[820px] text-sm">
        <thead>
          <tr className="border-b border-border-light text-left text-cool-gray">
            <th className="px-3 py-3 font-medium">이름</th>
            <th className="px-3 py-3 font-medium">종류</th>
            <th className="px-3 py-3 font-medium">기수</th>
            <th className="px-3 py-3 font-medium">과</th>
            <th className="px-3 py-3 font-medium">등급</th>
            <th className="px-3 py-3 font-medium">출처</th>
            <th className="px-3 py-3 text-center font-medium">노출</th>
            <th className="px-3 py-3 text-center font-medium">동의</th>
            <th className="px-3 py-3 text-right font-medium">관리</th>
          </tr>
        </thead>
        <tbody>
          {entries.map((entry) => (
            <tr key={entry.id} className="border-b border-border-light hover:bg-background">
              <td className="px-3 py-3 text-dark-slate">{entry.displayName}</td>
              <td className="px-3 py-3 text-cool-gray">{HALL_KIND_LABELS[entry.kind]}</td>
              <td className="px-3 py-3 text-cool-gray">{entry.cohort ?? EMPTY_VALUE_LABEL}</td>
              <td className="px-3 py-3 text-cool-gray">{entry.department ?? EMPTY_VALUE_LABEL}</td>
              <td className="px-3 py-3">
                <Badge variant={entry.tierManWon === null ? 'muted' : 'default'}>
                  {formatTier(entry.tierManWon)}
                </Badge>
              </td>
              <td className="px-3 py-3 text-cool-gray">{HALL_SOURCE_LABELS[entry.source]}</td>
              <td className="px-3 py-3 text-center">
                <EntrySwitch
                  label={`${entry.displayName} 노출`}
                  checked={entry.visible}
                  disabled={isUpdating}
                  onChange={(checked) => onVisibleChange(entry, checked)}
                />
              </td>
              <td className="px-3 py-3 text-center">
                <EntrySwitch
                  label={`${entry.displayName} 공개 동의`}
                  checked={entry.consentPublic}
                  disabled={isUpdating}
                  onChange={(checked) => onConsentChange(entry, checked)}
                />
              </td>
              <td className="px-3 py-3">
                <div className="flex justify-end gap-1">
                  <Button
                    variant="ghost"
                    size="icon"
                    aria-label={`${entry.displayName} 수정`}
                    onClick={() => onEdit(entry)}
                  >
                    <Pencil className="h-4 w-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    aria-label={`${entry.displayName} 삭제`}
                    onClick={() => onDelete(entry)}
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

interface EntrySwitchProps {
  label: string;
  checked: boolean;
  disabled: boolean;
  onChange: (checked: boolean) => void;
}

function EntrySwitch({ label, checked, disabled, onChange }: EntrySwitchProps) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      aria-label={label}
      disabled={disabled}
      onClick={() => onChange(!checked)}
      className={cn(
        'relative inline-flex h-6 w-10 shrink-0 items-center rounded-full transition-colors',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-royal-indigo',
        'disabled:cursor-not-allowed disabled:opacity-50',
        checked ? 'bg-royal-indigo' : 'bg-border',
      )}
    >
      <span
        className={cn(
          'inline-block h-4 w-4 rounded-full bg-white transition-transform',
          checked ? 'translate-x-5' : 'translate-x-1',
        )}
      />
    </button>
  );
}
