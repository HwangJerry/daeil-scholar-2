// ErasurePreviewTable — One table's affected rows, with before/after values for anonymized columns.
import type { ErasurePreviewRow, ErasurePreviewTable as PreviewTable } from '../api/accountErasurePreview';
import { tableLabel } from './accountErasureLabels';

const ACTION_LABELS: Record<PreviewTable['action'], string> = {
  delete: '행 삭제',
  anonymize: '익명화 · 행은 남기고 표시된 칸만 변경',
};

function CellValue({ value, masked }: { value: string | null | undefined; masked: boolean }) {
  if (value === null || value === undefined) return <span className="text-cool-gray">NULL</span>;
  if (value === '') return <span className="text-cool-gray">(빈 값)</span>;
  return <span className={masked ? 'italic text-cool-gray' : 'break-all'}>{value}</span>;
}

function PreviewCell({ row, index, changed, masked }: { row: ErasurePreviewRow; index: number; changed: boolean; masked: boolean }) {
  if (!changed || !row.after) return <CellValue value={row.before[index]} masked={masked} />;
  return (
    <span className="flex flex-col gap-1">
      <del className="text-cool-gray"><CellValue value={row.before[index]} masked={masked} /></del>
      <ins className="font-semibold text-dark-slate no-underline">→ <CellValue value={row.after[index]} masked={masked} /></ins>
    </span>
  );
}

export function ErasurePreviewTable({ table }: { table: PreviewTable }) {
  const changed = new Set(table.changedColumns);
  const masked = new Set(table.maskedColumns);
  const hasNotes = table.rows.some((row) => row.note);
  const hidden = table.count - table.rows.length;
  return (
    <details className="rounded-lg border border-border-light">
      <summary className="cursor-pointer p-3 text-sm text-dark-slate">
        <span className="font-semibold">{tableLabel(table.table)}</span>{' '}
        <span className="font-mono text-xs text-cool-gray">{table.table}</span> · {ACTION_LABELS[table.action]} · {table.count}건
      </summary>
      <div className="space-y-2 p-3">
        {table.action === 'anonymize' && <p className="text-xs text-cool-gray">변경되는 칸: {table.changedColumns.join(', ')}</p>}
        <div className="overflow-x-auto">
          <table className="min-w-full border-collapse text-left text-xs">
            <thead>
              <tr>
                {table.columns.map((column) => (
                  <th key={column} scope="col" className="border-b border-border-light px-2 py-1 font-mono font-semibold text-dark-slate">
                    {column}{changed.has(column) && ' ✎'}
                  </th>
                ))}
                {hasNotes && <th scope="col" className="border-b border-border-light px-2 py-1 font-semibold text-dark-slate">처리</th>}
              </tr>
            </thead>
            <tbody>
              {table.rows.map((row, rowIndex) => (
                <tr key={rowIndex} className="align-top">
                  {table.columns.map((column, index) => (
                    <td key={column} className="border-b border-border-light px-2 py-1">
                      <PreviewCell row={row} index={index} changed={changed.has(column)} masked={masked.has(column)} />
                    </td>
                  ))}
                  {hasNotes && <td className="border-b border-border-light px-2 py-1 text-dark-slate">{row.note}</td>}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {hidden > 0 && <p className="text-xs text-cool-gray">화면에는 {table.rows.length}건만 표시합니다. 나머지 {hidden}건도 같은 방식으로 처리되며 승인 확인값에 모두 포함됩니다.</p>}
        {table.maskedColumns.length > 0 && <p className="text-xs text-cool-gray">보안값과 쪽지 본문은 표시하지 않습니다: {table.maskedColumns.join(', ')}</p>}
      </div>
    </details>
  );
}
