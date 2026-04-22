// JobCategoryPage — drag-and-drop reorderable table for ALUMNI_JOB_CATEGORY admin management
import { useState } from 'react';
import { Plus } from 'lucide-react';
import {
  DndContext,
  KeyboardSensor,
  PointerSensor,
  closestCenter,
  useSensor,
  useSensors,
  type DragEndEvent,
} from '@dnd-kit/core';
import {
  SortableContext,
  arrayMove,
  sortableKeyboardCoordinates,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable';
import { Button } from '../components/ui/Button.tsx';
import { ConfirmDialog } from '../components/ui/ConfirmDialog.tsx';
import { ErrorState } from '../components/ui/ErrorState.tsx';
import { JobCategorySortableRow } from '../components/member/JobCategorySortableRow.tsx';
import { JobCategoryInlineEditRow } from '../components/member/JobCategoryInlineEditRow.tsx';
import { useJobCategoryList } from '../hooks/useJobCategoryList.ts';
import type { AdminJobCategory, AdminJobCategoryUpsert } from '../types/api.ts';

const BLANK_DRAFT: AdminJobCategoryUpsert = {
  name: '',
  openYn: 'Y',
};

interface EditState {
  seq: number | null;
  draft: AdminJobCategoryUpsert;
}

export function JobCategoryPage() {
  const {
    data,
    isLoading,
    isError,
    refetch,
    createCategory,
    updateCategory,
    deleteCategory,
    reorderCategories,
    isCreating,
    isUpdating,
    isDeleting,
  } = useJobCategoryList();

  const [edit, setEdit] = useState<EditState | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<AdminJobCategory | null>(null);

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 4 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  );

  const startEdit = (cat: AdminJobCategory) =>
    setEdit({ seq: cat.seq, draft: { name: cat.name, openYn: cat.openYn } });

  const startNew = () => setEdit({ seq: null, draft: { ...BLANK_DRAFT } });
  const cancelEdit = () => setEdit(null);

  const setDraft = (patch: Partial<AdminJobCategoryUpsert>) =>
    setEdit((prev) => (prev ? { ...prev, draft: { ...prev.draft, ...patch } } : prev));

  const handleSave = () => {
    if (!edit) return;
    if (edit.seq === null) {
      createCategory(edit.draft, { onSuccess: () => setEdit(null) });
    } else {
      updateCategory({ seq: edit.seq, body: edit.draft }, { onSuccess: () => setEdit(null) });
    }
  };

  const handleDelete = () => {
    if (!deleteTarget) return;
    deleteCategory(deleteTarget.seq, { onSuccess: () => setDeleteTarget(null) });
  };

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (!over || !data || active.id === over.id) return;
    const oldIndex = data.findIndex((c) => c.seq === active.id);
    const newIndex = data.findIndex((c) => c.seq === over.id);
    if (oldIndex < 0 || newIndex < 0) return;
    const newOrder = arrayMove(data, oldIndex, newIndex);
    reorderCategories(newOrder.map((c) => c.seq));
  };

  const isSaving = isCreating || isUpdating;
  const isNewRow = edit?.seq === null;
  const dragDisabled = edit !== null;
  const sortableIds = data?.map((c) => c.seq) ?? [];

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-bold text-dark-slate">직업 카테고리 관리</h2>
        <Button size="sm" onClick={startNew} disabled={edit !== null}>
          <Plus className="mr-1 h-4 w-4" />
          추가
        </Button>
      </div>

      <div className="overflow-x-auto rounded-2xl border border-border-light bg-white shadow-sm">
        <table className="w-full table-fixed text-sm">
          <thead>
            <tr className="border-b border-border-light text-left text-cool-gray">
              <th className="px-2 py-3 font-medium w-10 text-center" aria-label="순서 변경 핸들"></th>
              <th className="px-4 py-3 font-medium w-12 text-center">#</th>
              <th className="px-4 py-3 font-medium">이름</th>
              <th className="px-4 py-3 font-medium w-24 text-center">공개</th>
              <th className="px-4 py-3 font-medium w-28 text-center">액션</th>
            </tr>
          </thead>
          <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
            <SortableContext items={sortableIds} strategy={verticalListSortingStrategy}>
              <tbody aria-live="polite">
                {isNewRow && edit && (
                  <JobCategoryInlineEditRow
                    draft={edit.draft}
                    onChange={setDraft}
                    onSave={handleSave}
                    onCancel={cancelEdit}
                    isSaving={isSaving}
                  />
                )}

                {isError ? (
                  <ErrorState colSpan={5} onRetry={() => void refetch()} />
                ) : isLoading ? (
                  <tr><td colSpan={5} className="px-4 py-8 text-center text-cool-gray">로딩 중...</td></tr>
                ) : data?.length ? (
                  data.map((cat, idx) =>
                    edit?.seq === cat.seq ? (
                      <JobCategoryInlineEditRow
                        key={cat.seq}
                        draft={edit.draft}
                        onChange={setDraft}
                        onSave={handleSave}
                        onCancel={cancelEdit}
                        isSaving={isSaving}
                      />
                    ) : (
                      <JobCategorySortableRow
                        key={cat.seq}
                        cat={cat}
                        index={idx + 1}
                        onEdit={() => startEdit(cat)}
                        onDelete={() => setDeleteTarget(cat)}
                        disabled={dragDisabled}
                      />
                    )
                  )
                ) : (
                  !isNewRow && (
                    <tr><td colSpan={5} className="px-4 py-8 text-center text-cool-gray">카테고리가 없습니다.</td></tr>
                  )
                )}
              </tbody>
            </SortableContext>
          </DndContext>
        </table>
      </div>

      <ConfirmDialog
        open={deleteTarget !== null}
        onOpenChange={(open) => { if (!open) setDeleteTarget(null); }}
        title="카테고리 비활성화"
        description={`"${deleteTarget?.name}" 카테고리를 비활성화합니다. 연결된 회원 정보는 유지됩니다.`}
        confirmLabel="비활성화"
        variant="destructive"
        onConfirm={handleDelete}
        isPending={isDeleting}
      />
    </div>
  );
}
