// DonationImportSection — previews and commits Happy Sharing Excel donation rows
import { useCallback, useState } from 'react';
import { FileSpreadsheet, Loader2, Upload } from 'lucide-react';
import { ApiClientError } from '../../api/client.ts';
import { useCommitDonationImport, usePreviewDonationImport } from '../../hooks/useDonationImport.ts';
import { useToast } from '../../hooks/useToast.ts';
import { formatAmount } from '../../lib/formatAmount.ts';
import type {
  DonationImportCommitResult,
  DonationImportCommitRow,
  DonationImportPreviewResult,
  DonationImportPreviewRow,
  DonationImportRowStatus,
} from '../../types/api.ts';
import { Badge } from '../ui/Badge.tsx';
import { Button } from '../ui/Button.tsx';
import { Input } from '../ui/Input.tsx';

const EXCEL_FILE_EXTENSION = '.xlsx';

const STATUS_DETAILS: Record<
  DonationImportRowStatus,
  { label: string; variant: 'success' | 'warning' | 'muted' | 'danger' }
> = {
  matched: { label: '매칭', variant: 'success' },
  ambiguous: { label: '복수 후보', variant: 'warning' },
  unmatched: { label: '미매칭', variant: 'muted' },
  duplicate: { label: '중복', variant: 'danger' },
};

interface ImportRowSelection {
  selectedRowIndexes: Set<number>;
  accountUsrSeqInputs: Record<number, string>;
  unlinkedRowIndexes: Set<number>;
}

function createInitialSelection(preview: DonationImportPreviewResult): ImportRowSelection {
  const matchedRows = preview.rows.filter((row) => row.status === 'matched');
  return {
    selectedRowIndexes: new Set(matchedRows.map((row) => row.rowIndex)),
    accountUsrSeqInputs: Object.fromEntries(
      matchedRows.flatMap((row) => (
        row.matchedUsrSeq == null ? [] : [[row.rowIndex, String(row.matchedUsrSeq)]]
      )),
    ),
    unlinkedRowIndexes: new Set(),
  };
}

function getErrorMessage(error: unknown): string {
  if (error instanceof ApiClientError) return error.message;
  return '네트워크 상태를 확인하고 다시 시도해 주세요.';
}

export function DonationImportSection() {
  const addToast = useToast((state) => state.addToast);
  const previewMutation = usePreviewDonationImport();
  const commitMutation = useCommitDonationImport();
  const [file, setFile] = useState<File | null>(null);
  const [preview, setPreview] = useState<DonationImportPreviewResult | null>(null);
  const [selection, setSelection] = useState<ImportRowSelection>({
    selectedRowIndexes: new Set(),
    accountUsrSeqInputs: {},
    unlinkedRowIndexes: new Set(),
  });
  const [commitResult, setCommitResult] = useState<DonationImportCommitResult | null>(null);

  const handleFileSelected = (nextFile: File) => {
    setFile(nextFile);
    setPreview(null);
    setCommitResult(null);
  };

  const handlePreview = () => {
    if (!file) return;
    previewMutation.mutate(file, {
      onSuccess: (result) => {
        setPreview(result);
        setSelection(createInitialSelection(result));
        setCommitResult(null);
        addToast({
          variant: 'success',
          title: '엑셀 미리보기를 만들었습니다.',
          description: `총 ${result.rows.length.toLocaleString()}개 행을 확인해 주세요.`,
        });
      },
      onError: (error) => {
        addToast({
          variant: 'error',
          title: '미리보기 실패',
          description: getErrorMessage(error),
        });
      },
    });
  };

  const handleRowSelected = (rowIndex: number, checked: boolean) => {
    setSelection((current) => {
      const selectedRowIndexes = new Set(current.selectedRowIndexes);
      if (checked) selectedRowIndexes.add(rowIndex);
      else selectedRowIndexes.delete(rowIndex);
      return { ...current, selectedRowIndexes };
    });
  };

  const handleAccountUsrSeqChanged = (rowIndex: number, value: string) => {
    setSelection((current) => {
      const unlinkedRowIndexes = new Set(current.unlinkedRowIndexes);
      if (value !== '') unlinkedRowIndexes.delete(rowIndex);
      return {
        ...current,
        accountUsrSeqInputs: { ...current.accountUsrSeqInputs, [rowIndex]: value },
        unlinkedRowIndexes,
      };
    });
  };

  const handleUnlinkedChanged = (rowIndex: number, checked: boolean) => {
    setSelection((current) => {
      const unlinkedRowIndexes = new Set(current.unlinkedRowIndexes);
      if (checked) unlinkedRowIndexes.add(rowIndex);
      else unlinkedRowIndexes.delete(rowIndex);
      return {
        ...current,
        accountUsrSeqInputs: checked
          ? { ...current.accountUsrSeqInputs, [rowIndex]: '' }
          : current.accountUsrSeqInputs,
        unlinkedRowIndexes,
      };
    });
  };

  const handleCommit = () => {
    if (!preview) return;
    const selectedRows = preview.rows.filter((row) => selection.selectedRowIndexes.has(row.rowIndex));
    const invalidRows = selectedRows.filter((row) => {
      if (row.status === 'matched') return false;
      if (selection.unlinkedRowIndexes.has(row.rowIndex)) return false;
      const accountUsrSeq = Number(selection.accountUsrSeqInputs[row.rowIndex]);
      return !Number.isInteger(accountUsrSeq) || accountUsrSeq <= 0;
    });

    if (invalidRows.length > 0) {
      addToast({
        variant: 'error',
        title: '계정 연결 방법을 확인해 주세요.',
        description: `${invalidRows.map((row) => `${row.rowIndex}행`).join(', ')}에 회원 번호를 입력하거나 계정 연결 없이 반영을 선택하세요.`,
      });
      return;
    }

    const commitRows: DonationImportCommitRow[] = selectedRows.map((row) => {
      const accountUsrSeqInput = selection.accountUsrSeqInputs[row.rowIndex];
      const accountUsrSeq = selection.unlinkedRowIndexes.has(row.rowIndex)
        ? null
        : Number(accountUsrSeqInput || row.matchedUsrSeq);
      const isValidAccountUsrSeq = accountUsrSeq !== null
        && Number.isInteger(accountUsrSeq)
        && accountUsrSeq > 0;
      return {
        rowIndex: row.rowIndex,
        donorName: row.donorName,
        donorPhone: row.donorPhone,
        amount: row.amount,
        donationDate: row.donationDate,
        transactionNo: row.transactionNo,
        accountUsrSeq: isValidAccountUsrSeq ? accountUsrSeq : null,
      };
    });

    commitMutation.mutate(commitRows, {
      onSuccess: (result) => {
        setCommitResult(result);
        const successCount = result.rows.filter((row) => row.success).length;
        const failedRows = result.rows.filter((row) => !row.success);
        addToast({
          variant: failedRows.length === 0 ? 'success' : 'error',
          title: `기부 내역 반영: 성공 ${successCount}건, 실패 ${failedRows.length}건`,
          description: failedRows.length > 0
            ? failedRows.map((row) => `${row.rowIndex}행: ${row.errorMessage}`).join(' / ')
            : undefined,
        });
      },
      onError: (error) => {
        addToast({
          variant: 'error',
          title: '기부 내역 반영 실패',
          description: getErrorMessage(error),
        });
      },
    });
  };

  return (
    <section className="rounded-2xl border border-border-light bg-white p-6 shadow-sm">
      <div className="mb-4">
        <h3 className="font-semibold text-dark-slate">해피나눔 엑셀 기부 임포트</h3>
        <p className="mt-1 text-sm text-cool-gray">
          .xlsx 파일을 미리 확인한 뒤 반영할 행과 회원 계정 연결 여부를 선택하세요.
        </p>
      </div>

      <div className="space-y-3">
        <ExcelFileDropzone
          file={file}
          onFileSelected={handleFileSelected}
          disabled={previewMutation.isPending || commitMutation.isPending}
          onInvalidFile={() => addToast({
            variant: 'error',
            title: '엑셀 파일을 선택해 주세요.',
            description: '.xlsx 형식만 사용할 수 있습니다.',
          })}
        />
        <Button onClick={handlePreview} disabled={!file || previewMutation.isPending || commitMutation.isPending}>
          {previewMutation.isPending ? (
            <><Loader2 className="mr-2 h-4 w-4 animate-spin" />미리보기 생성 중...</>
          ) : '미리보기'}
        </Button>
      </div>

      {preview && (
        <PreviewTable
          preview={preview}
          selection={selection}
          onRowSelected={handleRowSelected}
          onAccountUsrSeqChanged={handleAccountUsrSeqChanged}
          onUnlinkedChanged={handleUnlinkedChanged}
          onCommit={handleCommit}
          isCommitting={commitMutation.isPending}
        />
      )}

      {commitResult && <CommitResultTable result={commitResult} />}
    </section>
  );
}

interface ExcelFileDropzoneProps {
  file: File | null;
  onFileSelected: (file: File) => void;
  onInvalidFile: () => void;
  disabled: boolean;
}

function ExcelFileDropzone({ file, onFileSelected, onInvalidFile, disabled }: ExcelFileDropzoneProps) {
  const [isDragOver, setIsDragOver] = useState(false);

  const processFile = useCallback((selectedFile: File | undefined) => {
    if (!selectedFile) return;
    if (!selectedFile.name.toLowerCase().endsWith(EXCEL_FILE_EXTENSION)) {
      onInvalidFile();
      return;
    }
    onFileSelected(selectedFile);
  }, [onFileSelected, onInvalidFile]);

  const handleDrop = useCallback((event: React.DragEvent<HTMLLabelElement>) => {
    event.preventDefault();
    setIsDragOver(false);
    if (disabled) return;
    processFile(event.dataTransfer.files[0]);
  }, [disabled, processFile]);

  const stateClass = isDragOver
    ? 'border-royal-indigo bg-royal-indigo/5'
    : 'border-border hover:border-royal-indigo/50';

  return (
    <label
      className={`flex cursor-pointer select-none flex-col items-center justify-center gap-2 rounded-xl border-2 border-dashed px-4 py-6 text-sm transition-colors ${stateClass} ${disabled ? 'pointer-events-none opacity-60' : ''}`}
      onDragOver={(event) => { event.preventDefault(); setIsDragOver(true); }}
      onDragLeave={() => setIsDragOver(false)}
      onDrop={handleDrop}
    >
      <input
        type="file"
        accept={EXCEL_FILE_EXTENSION}
        className="hidden"
        disabled={disabled}
        onChange={(event) => {
          processFile(event.target.files?.[0]);
          event.target.value = '';
        }}
      />
      {file ? (
        <FileSpreadsheet className="h-5 w-5 text-royal-indigo" />
      ) : (
        <Upload className="h-5 w-5 text-text-tertiary" />
      )}
      <span className="text-dark-slate">
        {file ? file.name : '엑셀 파일을 드래그하거나 클릭하여 선택'}
      </span>
      <span className="text-xs text-text-placeholder">해피나눔 .xlsx 파일 지원</span>
    </label>
  );
}

interface PreviewTableProps {
  preview: DonationImportPreviewResult;
  selection: ImportRowSelection;
  onRowSelected: (rowIndex: number, checked: boolean) => void;
  onAccountUsrSeqChanged: (rowIndex: number, value: string) => void;
  onUnlinkedChanged: (rowIndex: number, checked: boolean) => void;
  onCommit: () => void;
  isCommitting: boolean;
}

function PreviewTable({
  preview,
  selection,
  onRowSelected,
  onAccountUsrSeqChanged,
  onUnlinkedChanged,
  onCommit,
  isCommitting,
}: PreviewTableProps) {
  const selectedCount = selection.selectedRowIndexes.size;

  return (
    <div className="mt-6 space-y-4">
      <div className="flex flex-wrap gap-2">
        <Badge variant="success">매칭 {preview.matchedCount.toLocaleString()}</Badge>
        <Badge variant="warning">복수 후보 {preview.ambiguousCount.toLocaleString()}</Badge>
        <Badge variant="muted">미매칭 {preview.unmatchedCount.toLocaleString()}</Badge>
        <Badge variant="danger">중복 {preview.duplicateCount.toLocaleString()}</Badge>
      </div>

      <div className="overflow-x-auto rounded-xl border border-border-light">
        <table className="w-full min-w-[1180px] text-sm">
          <thead>
            <tr className="border-b border-border-light text-left text-cool-gray">
              <th className="w-16 px-3 py-3 text-center font-medium">반영</th>
              <th className="w-16 px-3 py-3 font-medium">행</th>
              <th className="px-3 py-3 font-medium">이름</th>
              <th className="px-3 py-3 font-medium">전화</th>
              <th className="px-3 py-3 text-right font-medium">금액</th>
              <th className="px-3 py-3 font-medium">기부일</th>
              <th className="px-3 py-3 font-medium">거래번호</th>
              <th className="px-3 py-3 text-center font-medium">상태</th>
              <th className="min-w-60 px-3 py-3 font-medium">회원 계정 연결</th>
            </tr>
          </thead>
          <tbody>
            {preview.rows.map((row) => (
              <PreviewTableRow
                key={row.rowIndex}
                row={row}
                selected={selection.selectedRowIndexes.has(row.rowIndex)}
                accountUsrSeqInput={selection.accountUsrSeqInputs[row.rowIndex] ?? ''}
                unlinked={selection.unlinkedRowIndexes.has(row.rowIndex)}
                onSelected={onRowSelected}
                onAccountUsrSeqChanged={onAccountUsrSeqChanged}
                onUnlinkedChanged={onUnlinkedChanged}
                disabled={isCommitting}
              />
            ))}
          </tbody>
        </table>
      </div>

      <div className="flex items-center justify-between gap-4">
        <p className="text-sm text-cool-gray">선택 {selectedCount.toLocaleString()}개 행</p>
        <Button onClick={onCommit} disabled={selectedCount === 0 || isCommitting}>
          {isCommitting ? (
            <><Loader2 className="mr-2 h-4 w-4 animate-spin" />반영 중...</>
          ) : '확정 반영'}
        </Button>
      </div>
    </div>
  );
}

interface PreviewTableRowProps {
  row: DonationImportPreviewRow;
  selected: boolean;
  accountUsrSeqInput: string;
  unlinked: boolean;
  onSelected: (rowIndex: number, checked: boolean) => void;
  onAccountUsrSeqChanged: (rowIndex: number, value: string) => void;
  onUnlinkedChanged: (rowIndex: number, checked: boolean) => void;
  disabled: boolean;
}

function PreviewTableRow({
  row,
  selected,
  accountUsrSeqInput,
  unlinked,
  onSelected,
  onAccountUsrSeqChanged,
  onUnlinkedChanged,
  disabled,
}: PreviewTableRowProps) {
  const isDuplicate = row.status === 'duplicate';
  const isMatched = row.status === 'matched';
  const statusDetail = STATUS_DETAILS[row.status];

  return (
    <tr className="border-b border-border-light align-top hover:bg-background">
      <td className="px-3 py-3 text-center">
        <input
          type="checkbox"
          aria-label={`${row.rowIndex}행 반영`}
          checked={selected}
          disabled={disabled || isDuplicate}
          onChange={(event) => onSelected(row.rowIndex, event.target.checked)}
          className="h-4 w-4 rounded border-gray-300 accent-indigo-600"
        />
      </td>
      <td className="px-3 py-3 text-cool-gray">{row.rowIndex}</td>
      <td className="px-3 py-3 text-dark-slate">{row.donorName || '—'}</td>
      <td className="px-3 py-3 text-cool-gray">{row.donorPhone || '—'}</td>
      <td className="px-3 py-3 text-right text-dark-slate">₩{formatAmount(row.amount)}</td>
      <td className="px-3 py-3 text-cool-gray">{row.donationDate || '—'}</td>
      <td className="px-3 py-3 font-mono text-xs text-cool-gray">{row.transactionNo || '—'}</td>
      <td className="px-3 py-3 text-center">
        <Badge variant={statusDetail.variant}>{statusDetail.label}</Badge>
        {row.note && <p className="mt-1 max-w-44 text-xs text-cool-gray">{row.note}</p>}
      </td>
      <td className="px-3 py-3">
        {isMatched ? (
          <div>
            <p className="text-dark-slate">{row.matchedName || '—'}</p>
            <p className="text-xs text-cool-gray">회원 번호 {row.matchedUsrSeq ?? '—'}</p>
          </div>
        ) : isDuplicate ? (
          <span className="text-cool-gray">반영할 수 없음</span>
        ) : (
          <div className="space-y-2">
            <Input
              type="number"
              min={1}
              step={1}
              aria-label={`${row.rowIndex}행 회원 번호`}
              placeholder="accountUsrSeq"
              value={accountUsrSeqInput}
              disabled={disabled || unlinked}
              onChange={(event) => onAccountUsrSeqChanged(row.rowIndex, event.target.value)}
              className="h-8"
            />
            <label className="flex cursor-pointer items-center gap-2 text-xs text-dark-slate">
              <input
                type="checkbox"
                checked={unlinked}
                disabled={disabled}
                onChange={(event) => onUnlinkedChanged(row.rowIndex, event.target.checked)}
                className="h-4 w-4 rounded border-gray-300 accent-indigo-600"
              />
              계정 연결 없이 반영
            </label>
          </div>
        )}
      </td>
    </tr>
  );
}

function CommitResultTable({ result }: { result: DonationImportCommitResult }) {
  const successCount = result.rows.filter((row) => row.success).length;
  const failedCount = result.rows.length - successCount;

  return (
    <div className="mt-6 space-y-3">
      <div className="flex items-center gap-2">
        <h4 className="font-medium text-dark-slate">반영 결과</h4>
        <Badge variant="success">성공 {successCount.toLocaleString()}</Badge>
        {failedCount > 0 && <Badge variant="danger">실패 {failedCount.toLocaleString()}</Badge>}
      </div>
      <div className="overflow-x-auto rounded-xl border border-border-light">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border-light text-left text-cool-gray">
              <th className="px-3 py-2 font-medium">엑셀 행</th>
              <th className="px-3 py-2 font-medium">결과</th>
              <th className="px-3 py-2 font-medium">주문 번호</th>
              <th className="px-3 py-2 font-medium">실패 사유</th>
            </tr>
          </thead>
          <tbody>
            {result.rows.map((row) => (
              <tr key={row.rowIndex} className="border-b border-border-light">
                <td className="px-3 py-2 text-dark-slate">{row.rowIndex}</td>
                <td className="px-3 py-2">
                  <Badge variant={row.success ? 'success' : 'danger'}>
                    {row.success ? '성공' : '실패'}
                  </Badge>
                </td>
                <td className="px-3 py-2 text-cool-gray">{row.orderSeq ?? '—'}</td>
                <td className="px-3 py-2 text-error-text">{row.errorMessage || '—'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
