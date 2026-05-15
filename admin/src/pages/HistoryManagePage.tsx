// HistoryManagePage — inline CRUD table for scholarship history entries
import { useState } from 'react';
import { Plus, Pencil, Trash2, Check, X } from 'lucide-react';
import { Button } from '../components/ui/Button.tsx';
import { Input } from '../components/ui/Input.tsx';
import { ConfirmDialog } from '../components/ui/ConfirmDialog.tsx';
import { ErrorState } from '../components/ui/ErrorState.tsx';
import { useHistoryList } from '../hooks/useHistoryList.ts';
import type { HistoryEntry, HistoryUpsertRequest } from '../types/api.ts';

const BLANK: HistoryUpsertRequest = { eventDate: '', text: '', sortOrder: 0 };

interface EditState {
  seq: number | null; // null = new row
  draft: HistoryUpsertRequest;
}

function yearOf(date: string) {
  return date.slice(0, 4);
}

function displayDate(date: string) {
  // "YYYY-MM-DD" → "MM.DD"
  const parts = date.split('-');
  if (parts.length < 3) return date;
  return `${parts[1]}.${parts[2]}`;
}

export function HistoryManagePage() {
  const { data, isLoading, isError, refetch, createEntry, updateEntry, deleteEntry, isCreating, isUpdating, isDeleting } =
    useHistoryList();

  const [edit, setEdit] = useState<EditState | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<HistoryEntry | null>(null);

  const startNew = () => setEdit({ seq: null, draft: { ...BLANK } });
  const startEdit = (e: HistoryEntry) =>
    setEdit({ seq: e.heSeq, draft: { eventDate: e.eventDate, text: e.text, sortOrder: e.sortOrder } });
  const cancelEdit = () => setEdit(null);

  const setDraft = (patch: Partial<HistoryUpsertRequest>) =>
    setEdit((prev) => (prev ? { ...prev, draft: { ...prev.draft, ...patch } } : prev));

  const handleSave = () => {
    if (!edit) return;
    if (edit.seq === null) {
      createEntry(edit.draft, { onSuccess: () => setEdit(null) });
    } else {
      updateEntry({ seq: edit.seq, body: edit.draft }, { onSuccess: () => setEdit(null) });
    }
  };

  const handleDelete = () => {
    if (!deleteTarget) return;
    deleteEntry(deleteTarget.heSeq, { onSuccess: () => setDeleteTarget(null) });
  };

  const isSaving = isCreating || isUpdating;
  const isNewRow = edit?.seq === null;

  // Group entries by year for display
  const grouped: { year: string; entries: HistoryEntry[] }[] = [];
  const yearMap = new Map<string, HistoryEntry[]>();
  for (const e of data ?? []) {
    const y = yearOf(e.eventDate);
    if (!yearMap.has(y)) yearMap.set(y, []);
    yearMap.get(y)!.push(e);
  }
  for (const [year, entries] of yearMap) {
    grouped.push({ year, entries });
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-bold text-dark-slate">연혁 관리</h2>
        <Button size="sm" onClick={startNew} disabled={edit !== null}>
          <Plus className="mr-1 h-4 w-4" />
          항목 추가
        </Button>
      </div>

      <div className="overflow-x-auto rounded-2xl border border-border-light bg-white shadow-sm">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border-light text-left text-cool-gray">
              <th className="px-4 py-3 font-medium w-28">날짜</th>
              <th className="px-4 py-3 font-medium">내용</th>
              <th className="px-4 py-3 font-medium w-20 text-center">순서</th>
              <th className="px-4 py-3 font-medium w-24 text-center">액션</th>
            </tr>
          </thead>
          <tbody aria-live="polite">
            {/* New row */}
            {isNewRow && edit && (
              <InlineEditRow
                draft={edit.draft}
                onChange={setDraft}
                onSave={handleSave}
                onCancel={cancelEdit}
                isSaving={isSaving}
              />
            )}

            {isError ? (
              <ErrorState colSpan={4} onRetry={() => void refetch()} />
            ) : isLoading ? (
              <tr>
                <td colSpan={4} className="px-4 py-8 text-center text-cool-gray">로딩 중...</td>
              </tr>
            ) : grouped.length === 0 && !isNewRow ? (
              <tr>
                <td colSpan={4} className="px-4 py-8 text-center text-cool-gray">연혁이 없습니다.</td>
              </tr>
            ) : (
              grouped.map(({ year, entries }) => (
                <>
                  <tr key={`year-${year}`} className="bg-gray-50">
                    <td colSpan={4} className="px-4 py-2 font-semibold text-dark-slate">{year}년</td>
                  </tr>
                  {entries.map((entry) =>
                    edit?.seq === entry.heSeq ? (
                      <InlineEditRow
                        key={entry.heSeq}
                        draft={edit.draft}
                        onChange={setDraft}
                        onSave={handleSave}
                        onCancel={cancelEdit}
                        isSaving={isSaving}
                      />
                    ) : (
                      <tr key={entry.heSeq} className="border-t border-border-light hover:bg-gray-50">
                        <td className="px-4 py-3 text-cool-gray">{displayDate(entry.eventDate)}</td>
                        <td className="px-4 py-3">{entry.text}</td>
                        <td className="px-4 py-3 text-center text-cool-gray">{entry.sortOrder}</td>
                        <td className="px-4 py-3">
                          <div className="flex items-center justify-center gap-2">
                            <button
                              onClick={() => startEdit(entry)}
                              disabled={edit !== null}
                              className="text-cool-gray hover:text-dark-slate disabled:opacity-30"
                              aria-label="수정"
                            >
                              <Pencil size={15} />
                            </button>
                            <button
                              onClick={() => setDeleteTarget(entry)}
                              disabled={edit !== null}
                              className="text-cool-gray hover:text-red-500 disabled:opacity-30"
                              aria-label="삭제"
                            >
                              <Trash2 size={15} />
                            </button>
                          </div>
                        </td>
                      </tr>
                    )
                  )}
                </>
              ))
            )}
          </tbody>
        </table>
      </div>

      <ConfirmDialog
        open={deleteTarget !== null}
        onOpenChange={(open) => { if (!open) setDeleteTarget(null); }}
        title="연혁 삭제"
        description={`"${deleteTarget?.text}" 항목을 삭제합니다. 이 작업은 되돌릴 수 없습니다.`}
        confirmLabel="삭제"
        variant="destructive"
        onConfirm={handleDelete}
        isPending={isDeleting}
      />
    </div>
  );
}

// ── Inline edit row ───────────────────────────────────────────────────────────

interface InlineEditRowProps {
  draft: HistoryUpsertRequest;
  onChange: (patch: Partial<HistoryUpsertRequest>) => void;
  onSave: () => void;
  onCancel: () => void;
  isSaving: boolean;
}

function InlineEditRow({ draft, onChange, onSave, onCancel, isSaving }: InlineEditRowProps) {
  return (
    <tr className="border-t border-border-light bg-blue-50">
      <td className="px-2 py-2">
        <Input
          type="date"
          value={draft.eventDate}
          onChange={(e) => onChange({ eventDate: e.target.value })}
          className="w-full"
        />
      </td>
      <td className="px-2 py-2">
        <Input
          value={draft.text}
          onChange={(e) => onChange({ text: e.target.value })}
          placeholder="내용을 입력하세요"
          className="w-full"
        />
      </td>
      <td className="px-2 py-2">
        <Input
          type="number"
          value={String(draft.sortOrder)}
          onChange={(e) => onChange({ sortOrder: Number(e.target.value) })}
          className="w-full text-center"
        />
      </td>
      <td className="px-2 py-2">
        <div className="flex items-center justify-center gap-2">
          <button
            onClick={onSave}
            disabled={isSaving}
            className="text-green-600 hover:text-green-700 disabled:opacity-50"
            aria-label="저장"
          >
            <Check size={16} />
          </button>
          <button
            onClick={onCancel}
            disabled={isSaving}
            className="text-cool-gray hover:text-dark-slate disabled:opacity-50"
            aria-label="취소"
          >
            <X size={16} />
          </button>
        </div>
      </td>
    </tr>
  );
}
