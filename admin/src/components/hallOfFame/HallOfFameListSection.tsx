// HallOfFameListSection — filterable entry list with add, edit, toggle, and delete flows
import { useEffect, useState } from 'react';
import { Plus } from 'lucide-react';
import { useConfirmDialog } from '../../hooks/useConfirmDialog.ts';
import { useHallOfFameList, useHallOfFameMutations } from '../../hooks/useHallOfFameEntries.ts';
import { usePagination } from '../../hooks/usePagination.ts';
import type { HallEntry, HallEntryInput, HallKind, HallTier } from '../../types/hallOfFame.ts';
import { Button } from '../ui/Button.tsx';
import { ConfirmDialog } from '../ui/ConfirmDialog.tsx';
import { ErrorState } from '../ui/ErrorState.tsx';
import { Pagination } from '../ui/Pagination.tsx';
import { HallOfFameEntryDialog } from './HallOfFameEntryDialog.tsx';
import { HallOfFameFilters } from './HallOfFameFilters.tsx';
import { HallOfFameTable } from './HallOfFameTable.tsx';

const SEARCH_DEBOUNCE_MS = 300;

type EntryDialogTarget = { mode: 'create' } | { mode: 'edit'; entry: HallEntry };

export function HallOfFameListSection() {
  const { page, pageSize, setPage, resetPage } = usePagination();
  const [tier, setTier] = useState<HallTier | ''>('');
  const [kind, setKind] = useState<HallKind | ''>('');
  const [search, setSearch] = useState('');
  const [q, setQ] = useState('');
  const [dialogTarget, setDialogTarget] = useState<EntryDialogTarget | null>(null);
  const deleteDialog = useConfirmDialog<HallEntry>();

  const list = useHallOfFameList({ tier, kind, q }, page, pageSize);
  const { create, update, remove } = useHallOfFameMutations();

  useEffect(() => {
    const timer = setTimeout(() => {
      setQ(search);
      resetPage();
    }, SEARCH_DEBOUNCE_MS);
    return () => clearTimeout(timer);
  }, [search, resetPage]);

  const handleSubmit = (input: HallEntryInput) => {
    const close = () => setDialogTarget(null);
    if (dialogTarget?.mode === 'edit') {
      update.mutate({ id: dialogTarget.entry.id, patch: input }, { onSuccess: close });
      return;
    }
    create.mutate({ ...input, visible: true, consentPublic: true }, { onSuccess: close });
  };

  const handleDelete = () => {
    const entry = deleteDialog.confirm();
    if (entry) remove.mutate(entry.id);
  };

  const entries = list.data?.items ?? [];
  const total = list.data?.total ?? 0;
  const totalPages = Math.ceil(total / pageSize);
  const dialogEntry = dialogTarget?.mode === 'edit' ? dialogTarget.entry : null;

  return (
    <section className="space-y-4 rounded-2xl border border-border-light bg-white p-6 shadow-sm">
      <div className="flex items-center justify-between gap-3">
        <p className="text-sm text-cool-gray">
          {list.isLoading ? '불러오는 중...' : `총 ${total.toLocaleString()}명`}
        </p>
        <Button size="sm" onClick={() => setDialogTarget({ mode: 'create' })}>
          <Plus className="mr-1 h-4 w-4" />
          추가
        </Button>
      </div>

      <HallOfFameFilters
        tier={tier}
        kind={kind}
        search={search}
        onTierChange={(nextTier) => { setTier(nextTier); resetPage(); }}
        onKindChange={(nextKind) => { setKind(nextKind); resetPage(); }}
        onSearchChange={setSearch}
      />

      {list.isError ? (
        <ErrorState onRetry={() => void list.refetch()} />
      ) : (
        <>
          {!list.isLoading && entries.length === 0 ? (
            <p className="py-10 text-center text-sm text-cool-gray">조건에 맞는 항목이 없습니다.</p>
          ) : (
            <HallOfFameTable
              entries={entries}
              isUpdating={update.isPending}
              onVisibleChange={(entry, visible) => update.mutate({ id: entry.id, patch: { visible } })}
              onConsentChange={(entry, consentPublic) => (
                update.mutate({ id: entry.id, patch: { consentPublic } })
              )}
              onEdit={(entry) => setDialogTarget({ mode: 'edit', entry })}
              onDelete={deleteDialog.open}
            />
          )}
          <Pagination page={page} totalPages={totalPages} onPageChange={setPage} />
        </>
      )}

      {dialogTarget && (
        <HallOfFameEntryDialog
          key={dialogEntry?.id ?? 'new'}
          entry={dialogEntry}
          isSubmitting={create.isPending || update.isPending}
          onClose={() => setDialogTarget(null)}
          onSubmit={handleSubmit}
        />
      )}

      <ConfirmDialog
        open={deleteDialog.isOpen}
        onOpenChange={(open) => { if (!open) deleteDialog.close(); }}
        title="항목 삭제"
        description={`${deleteDialog.pendingValue?.displayName ?? ''} 항목을 명예의 전당에서 삭제합니다.`}
        confirmLabel="삭제"
        variant="destructive"
        isPending={remove.isPending}
        onConfirm={handleDelete}
      />
    </section>
  );
}
