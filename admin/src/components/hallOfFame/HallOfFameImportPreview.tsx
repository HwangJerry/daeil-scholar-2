// HallOfFameImportPreview — reviewable diff of an uploaded spreadsheet before it is committed
import { useState, type ReactNode } from 'react';
import type {
  HallImportIssue,
  HallImportPreview,
  HallImportRow,
  HallImportUpdatedRow,
} from '../../types/hallOfFame.ts';
import { Badge } from '../ui/Badge.tsx';
import {
  formatImportValue,
  formatTier,
  getImportFieldLabel,
  HALL_KIND_LABELS,
} from './hallOfFameLabels.ts';

const LONG_SECTION_THRESHOLD = 20;

function describeRow(row: HallImportRow) {
  const parts = [
    HALL_KIND_LABELS[row.kind],
    row.cohort === null ? null : `${row.cohort}기`,
    row.department,
    formatTier(row.tierManWon),
  ];
  return parts.filter(Boolean).join(' · ');
}

interface HallOfFameImportPreviewProps {
  preview: HallImportPreview;
  removeIds: Set<number>;
  disabled: boolean;
  onRemoveToggle: (id: number, checked: boolean) => void;
}

export function HallOfFameImportPreview({
  preview,
  removeIds,
  disabled,
  onRemoveToggle,
}: HallOfFameImportPreviewProps) {
  return (
    <div className="space-y-4">
      {preview.alreadyCommitted && (
        <p className="text-sm text-cool-gray">
          이미 반영한 파일입니다. 현재 명단과 비교한 결과를 확인해 주세요.
        </p>
      )}
      <div className="flex flex-wrap gap-2">
        <Badge variant="success">추가 {preview.added.length.toLocaleString()}</Badge>
        <Badge variant="default">수정 {preview.updated.length.toLocaleString()}</Badge>
        <Badge variant="muted">변경 없음 {preview.unchangedCount.toLocaleString()}</Badge>
        {preview.manualSkipped > 0 && (
          <Badge variant="muted">수동 수정 보호 {preview.manualSkipped.toLocaleString()}</Badge>
        )}
        <Badge variant="warning">삭제 후보 {preview.removalCandidates.length.toLocaleString()}</Badge>
        <Badge variant="warning">경고 {preview.warnings.length.toLocaleString()}</Badge>
        <Badge variant="danger">오류 {preview.errors.length.toLocaleString()}</Badge>
      </div>
      {preview.manualSkipped > 0 && (
        <p className="text-sm text-cool-gray">수동 수정한 항목은 덮어쓰거나 삭제하지 않습니다.</p>
      )}

      <IssueSection title="오류" issues={preview.errors} tone="error" />
      <IssueSection title="경고" issues={preview.warnings} tone="warning" />

      <PreviewSection title="추가" count={preview.added.length}>
        <ul className="divide-y divide-border-light">
          {preview.added.map((row, index) => (
            <li key={row.rowIndex ?? index} className="flex flex-wrap justify-between gap-2 px-3 py-2">
              <span className="text-dark-slate">{row.displayName}</span>
              <span className="text-cool-gray">{describeRow(row)}</span>
            </li>
          ))}
        </ul>
      </PreviewSection>

      <PreviewSection title="수정" count={preview.updated.length}>
        <ul className="divide-y divide-border-light">
          {preview.updated.map((row) => <UpdatedRow key={row.id} row={row} />)}
        </ul>
      </PreviewSection>

      <PreviewSection
        title="삭제 후보"
        count={preview.removalCandidates.length}
        description="엑셀에 없는 항목입니다. 체크한 항목만 삭제됩니다."
        defaultOpen
      >
        <ul className="divide-y divide-border-light">
          {preview.removalCandidates.map((row) => (
            <li key={row.id} className="px-3 py-2">
              <label className="flex cursor-pointer flex-wrap items-center gap-2">
                <input
                  type="checkbox"
                  aria-label={`${row.displayName} 삭제`}
                  checked={removeIds.has(row.id)}
                  disabled={disabled}
                  onChange={(event) => onRemoveToggle(row.id, event.target.checked)}
                  className="h-4 w-4 rounded border-border accent-royal-indigo"
                />
                <span className="text-dark-slate">{row.displayName}</span>
                <span className="text-cool-gray">{describeRow(row)}</span>
              </label>
            </li>
          ))}
        </ul>
      </PreviewSection>
    </div>
  );
}

function UpdatedRow({ row }: { row: HallImportUpdatedRow }) {
  const changes = row.changes.flatMap((change) => {
    const label = getImportFieldLabel(change.field);
    return label ? [{ ...change, label }] : [];
  });

  return (
    <li className="px-3 py-2">
      <p className="text-dark-slate">{row.displayName}</p>
      {changes.map((change) => (
        <p key={change.field} className="mt-0.5 text-cool-gray">
          {change.label}: {formatImportValue(change.field, change.before)}
          {' → '}
          <span className="text-dark-slate">{formatImportValue(change.field, change.after)}</span>
        </p>
      ))}
    </li>
  );
}

interface PreviewSectionProps {
  title: string;
  count: number;
  description?: string;
  defaultOpen?: boolean;
  children: ReactNode;
}

function PreviewSection({ title, count, description, defaultOpen, children }: PreviewSectionProps) {
  const [isOpen, setIsOpen] = useState(defaultOpen || count <= LONG_SECTION_THRESHOLD);
  if (count === 0) return null;

  return (
    <details
      open={isOpen}
      onToggle={(event) => setIsOpen(event.currentTarget.open)}
      className="rounded-xl border border-border-light text-sm"
    >
      <summary className="cursor-pointer px-3 py-2 font-medium text-dark-slate">
        {title} {count.toLocaleString()}건
      </summary>
      {description && <p className="px-3 pb-2 text-xs text-cool-gray">{description}</p>}
      {children}
    </details>
  );
}

interface IssueSectionProps {
  title: string;
  issues: HallImportIssue[];
  tone: 'error' | 'warning';
}

function IssueSection({ title, issues, tone }: IssueSectionProps) {
  if (issues.length === 0) return null;
  const toneClass = tone === 'error'
    ? 'bg-error-subtle text-error-text'
    : 'bg-warning-subtle text-warning-text';

  return (
    <div className={`rounded-xl px-3 py-2 text-sm ${toneClass}`}>
      <p className="font-medium">{title}</p>
      <ul className="mt-1 space-y-0.5">
        {issues.map((issue, index) => (
          <li key={`${issue.rowIndex ?? 'file'}-${index}`}>
            {issue.rowIndex === null ? issue.message : `${issue.rowIndex}행 ${issue.message}`}
          </li>
        ))}
      </ul>
    </div>
  );
}
