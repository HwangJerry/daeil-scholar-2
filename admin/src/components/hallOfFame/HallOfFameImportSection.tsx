// HallOfFameImportSection — spreadsheet upload flow: select file, review preview, commit diff
import { useState } from 'react';
import { Loader2 } from 'lucide-react';
import { ApiClientError } from '../../api/client.ts';
import { useCommitHallImport, usePreviewHallImport } from '../../hooks/useHallOfFameImport.ts';
import { useToast } from '../../hooks/useToast.ts';
import type { HallImportCommitResult, HallImportPreview } from '../../types/hallOfFame.ts';
import { ExcelFileDropzone } from '../common/ExcelFileDropzone.tsx';
import { Button } from '../ui/Button.tsx';
import { ConfirmDialog } from '../ui/ConfirmDialog.tsx';
import { HallOfFameImportPreview } from './HallOfFameImportPreview.tsx';

function getErrorMessage(error: unknown) {
  return error instanceof ApiClientError && error.message
    ? error.message
    : '네트워크 상태를 확인하고 다시 시도해 주세요.';
}

export function HallOfFameImportSection() {
  const addToast = useToast((state) => state.addToast);
  const previewMutation = usePreviewHallImport();
  const commitMutation = useCommitHallImport();
  const [file, setFile] = useState<File | null>(null);
  const [preview, setPreview] = useState<HallImportPreview | null>(null);
  const [removeIds, setRemoveIds] = useState<Set<number>>(new Set());
  const [commitResult, setCommitResult] = useState<HallImportCommitResult | null>(null);
  const [isConfirmOpen, setIsConfirmOpen] = useState(false);

  const isBusy = previewMutation.isPending || commitMutation.isPending;

  const clearReview = () => {
    setPreview(null);
    setRemoveIds(new Set());
  };

  const handleFileSelected = (nextFile: File) => {
    setFile(nextFile);
    setCommitResult(null);
    clearReview();
  };

  const handlePreview = () => {
    if (!file) return;
    previewMutation.mutate(file, {
      onSuccess: (result) => {
        setPreview(result);
        setRemoveIds(new Set());
        setCommitResult(null);
      },
      onError: (error) => {
        addToast({ variant: 'error', title: '미리보기 실패', description: getErrorMessage(error) });
      },
    });
  };

  const handleRemoveToggle = (id: number, checked: boolean) => {
    setRemoveIds((current) => {
      const next = new Set(current);
      if (checked) next.add(id);
      else next.delete(id);
      return next;
    });
  };

  const handleCommit = () => {
    if (!preview) return;
    commitMutation.mutate(
      { previewToken: preview.previewToken, removeIds: [...removeIds] },
      {
        onSuccess: (result) => {
          setCommitResult(result);
          setFile(null);
          clearReview();
          addToast({ variant: 'success', title: '명예의 전당에 반영했습니다.' });
        },
        onError: (error) => {
          addToast({ variant: 'error', title: '반영 실패', description: getErrorMessage(error) });
        },
      },
    );
  };

  const hasErrors = (preview?.errors.length ?? 0) > 0;
  const changeCount = (preview?.added.length ?? 0) + (preview?.updated.length ?? 0) + removeIds.size;
  const canCommit = preview !== null && !hasErrors && changeCount > 0 && !isBusy;

  return (
    <section className="space-y-4 rounded-2xl border border-border-light bg-white p-6 shadow-sm">
      <div>
        <h3 className="font-semibold text-dark-slate">기부자 명단 업로드</h3>
        <p className="mt-1 text-sm text-cool-gray">
          .xlsx 파일을 미리 확인한 뒤 반영합니다. 같은 파일을 다시 올려도 변경은 생기지 않습니다.
        </p>
      </div>

      <ExcelFileDropzone
        file={file}
        hint="성함 | 연락처 | 기수 | 과 | 신규 8명 | 이동 10명 열의 .xlsx 파일"
        onFileSelected={handleFileSelected}
        disabled={isBusy}
        onInvalidFile={() => addToast({
          variant: 'error',
          title: '엑셀 파일을 선택해 주세요.',
          description: '.xlsx 형식만 사용할 수 있습니다.',
        })}
      />
      <Button onClick={handlePreview} disabled={!file || isBusy}>
        {previewMutation.isPending ? (
          <><Loader2 className="mr-2 h-4 w-4 animate-spin" />미리보기 생성 중...</>
        ) : '미리보기'}
      </Button>

      {preview && (
        <>
          <HallOfFameImportPreview
            preview={preview}
            removeIds={removeIds}
            disabled={isBusy}
            onRemoveToggle={handleRemoveToggle}
          />
          <div className="flex items-center justify-between gap-4">
            <p className="text-sm text-cool-gray">
              {hasErrors ? '오류를 수정한 파일을 다시 올려 주세요.' : null}
              {!hasErrors && changeCount === 0 ? '반영할 변경이 없습니다.' : null}
              {!hasErrors && changeCount > 0 ? `삭제 선택 ${removeIds.size.toLocaleString()}건` : null}
            </p>
            <Button onClick={() => setIsConfirmOpen(true)} disabled={!canCommit}>
              {commitMutation.isPending ? (
                <><Loader2 className="mr-2 h-4 w-4 animate-spin" />반영 중...</>
              ) : '반영'}
            </Button>
          </div>
        </>
      )}

      {commitResult && (
        <p role="status" className="rounded-xl bg-success-subtle px-3 py-2 text-sm text-success-text">
          반영 완료: 추가 {commitResult.addedCount.toLocaleString()}건,
          수정 {commitResult.updatedCount.toLocaleString()}건,
          삭제 {commitResult.removedCount.toLocaleString()}건
        </p>
      )}

      <ConfirmDialog
        open={isConfirmOpen}
        onOpenChange={setIsConfirmOpen}
        title="명단 반영"
        description={`추가 ${preview?.added.length ?? 0}건, 수정 ${preview?.updated.length ?? 0}건, 삭제 ${removeIds.size}건을 반영합니다.`}
        confirmLabel="반영"
        variant={removeIds.size > 0 ? 'destructive' : 'default'}
        onConfirm={handleCommit}
      />
    </section>
  );
}
