// FeedCategoryPage — 피드 카테고리 관리: drag-reorderable table with inline edit, app tab switch and move-on-delete
import { Plus } from 'lucide-react';
import { DndContext, KeyboardSensor, PointerSensor, closestCenter, useSensor, useSensors } from '@dnd-kit/core';
import { SortableContext, sortableKeyboardCoordinates, verticalListSortingStrategy } from '@dnd-kit/sortable';
import { Button } from '../components/ui/Button.tsx';
import { ErrorState } from '../components/ui/ErrorState.tsx';
import { FeedCategorySortableRow } from '../components/feedCategory/FeedCategorySortableRow.tsx';
import { FeedCategoryInlineEditRow } from '../components/feedCategory/FeedCategoryInlineEditRow.tsx';
import { FeedCategoryDeleteDialogs } from '../components/feedCategory/FeedCategoryDeleteDialogs.tsx';
import { FEED_CATEGORY_MAX_COUNT } from '../hooks/useFeedCategories.ts';
import { useFeedCategoryPageState } from '../hooks/useFeedCategoryPageState.ts';

const COLUMN_COUNT = 6;

export function FeedCategoryPage() {
  const page = useFeedCategoryPageState();
  const { categories, edit } = page;
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 4 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  );
  const dragDisabled = edit !== null || page.isSaving;

  return (
    <div className="space-y-4">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h2 className="text-xl font-bold text-dark-slate">피드 카테고리 관리</h2>
          <p className="mt-1 text-sm text-cool-gray">
            앱 피드의 탭과 공지 분류에 쓰입니다. 끌어서 순서를 바꾸면 앱 탭 순서도 바뀝니다.{' '}
            <span className="whitespace-nowrap">({categories.length} / 최대 {FEED_CATEGORY_MAX_COUNT}개)</span>
          </p>
        </div>
        <Button size="sm" onClick={page.startNew} disabled={edit !== null || page.isFull || page.isLoading}>
          <Plus className="mr-1 h-4 w-4" />
          추가
        </Button>
      </div>

      <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={page.handleDragEnd}>
        <div className="overflow-x-auto rounded-2xl border border-border-light bg-white shadow-sm">
          <table className="w-full table-fixed text-sm">
            <thead>
              <tr className="border-b border-border-light text-left text-cool-gray">
                <th className="px-2 py-3 font-medium w-10 text-center" aria-label="순서 변경 핸들"></th>
                <th className="px-4 py-3 font-medium w-12 text-center">#</th>
                <th className="px-4 py-3 font-medium">이름</th>
                <th className="px-4 py-3 font-medium w-24 text-center">게시글 수</th>
                <th className="px-4 py-3 font-medium w-28 text-center">앱 탭 노출</th>
                <th className="px-4 py-3 font-medium w-28 text-center">작업</th>
              </tr>
            </thead>
              <SortableContext items={categories.map((c) => c.seq)} strategy={verticalListSortingStrategy}>
                <tbody aria-live="polite">
                  {page.isError ? (
                    <ErrorState colSpan={COLUMN_COUNT} onRetry={() => void page.refetch()} />
                  ) : page.isLoading ? (
                    <tr><td colSpan={COLUMN_COUNT} className="px-4 py-8 text-center text-cool-gray">로딩 중...</td></tr>
                  ) : (
                    categories.map((cat, idx) =>
                      edit?.seq === cat.seq ? (
                        <FeedCategoryInlineEditRow
                          key={cat.seq}
                          draft={edit.draft}
                          isDefault={cat.isDefault === 'Y'}
                          postCount={cat.postCount}
                          onChange={page.setDraft}
                          onSave={page.save}
                          onCancel={page.cancelEdit}
                          isSaving={page.isSaving}
                        />
                      ) : (
                        <FeedCategorySortableRow
                          key={cat.seq}
                          cat={cat}
                          index={idx + 1}
                          onEdit={() => page.startEdit(cat)}
                          onDelete={() => page.requestDelete(cat)}
                          onToggleOpen={(open) => page.toggleOpen(cat, open)}
                          disabled={dragDisabled}
                        />
                      )
                    )
                  )}
                  {edit?.seq === null && (
                    <FeedCategoryInlineEditRow
                      draft={edit.draft}
                      isDefault={false}
                      postCount={null}
                      onChange={page.setDraft}
                      onSave={page.save}
                      onCancel={page.cancelEdit}
                      isSaving={page.isSaving}
                    />
                  )}
                </tbody>
              </SortableContext>
          </table>
        </div>
      </DndContext>

      {page.errorMessage && (
        <p role="alert" className="text-sm text-error-text">{page.errorMessage}</p>
      )}

      <FeedCategoryDeleteDialogs
        deleting={page.deleting}
        categories={categories}
        onConfirm={page.confirmDelete}
        onClose={page.closeDelete}
        isPending={page.isDeleting}
      />
    </div>
  );
}
