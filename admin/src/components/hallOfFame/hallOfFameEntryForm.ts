// hallOfFameEntryForm — form state, validation, and request mapping for the entry modal
import type { HallEntry, HallEntryInput, HallKind, HallTier } from '../../types/hallOfFame.ts';

const NAME_MAX_LENGTH = 100;
const DEPARTMENT_MAX_LENGTH = 30;
const ROLE_LABEL_MAX_LENGTH = 30;

export interface HallEntryFormState {
  displayName: string;
  kind: HallKind;
  cohort: string;
  department: string;
  tierManWon: string;
  contactName: string;
  roleLabel: string;
}

export type HallEntryFormResult =
  | { ok: true; input: HallEntryInput }
  | { ok: false; reason: string };

export function createHallEntryFormState(entry: HallEntry | null): HallEntryFormState {
  return {
    displayName: entry?.displayName ?? '',
    kind: entry?.kind ?? 'alumni',
    cohort: entry?.cohort == null ? '' : String(entry.cohort),
    department: entry?.department ?? '',
    tierManWon: entry?.tierManWon == null ? '' : String(entry.tierManWon),
    contactName: entry?.contactName ?? '',
    roleLabel: entry?.roleLabel ?? '',
  };
}

export function parseHallEntryForm(form: HallEntryFormState): HallEntryFormResult {
  const displayName = form.displayName.trim();
  const department = form.department.trim();
  const contactName = form.contactName.trim();
  const roleLabel = form.roleLabel.trim();
  const cohortText = form.cohort.trim();

  if (!displayName) return { ok: false, reason: '이름을 입력해 주세요.' };
  if (displayName.length > NAME_MAX_LENGTH) {
    return { ok: false, reason: `이름은 ${NAME_MAX_LENGTH}자 이하로 입력해 주세요.` };
  }
  if (cohortText && !/^[1-9]\d*$/.test(cohortText)) {
    return { ok: false, reason: '기수는 1 이상의 숫자로 입력해 주세요.' };
  }
  if (department.length > DEPARTMENT_MAX_LENGTH) {
    return { ok: false, reason: `과는 ${DEPARTMENT_MAX_LENGTH}자 이하로 입력해 주세요.` };
  }
  if (contactName.length > NAME_MAX_LENGTH) {
    return { ok: false, reason: `담당자는 ${NAME_MAX_LENGTH}자 이하로 입력해 주세요.` };
  }
  if (roleLabel.length > ROLE_LABEL_MAX_LENGTH) {
    return { ok: false, reason: `직함은 ${ROLE_LABEL_MAX_LENGTH}자 이하로 입력해 주세요.` };
  }

  return {
    ok: true,
    input: {
      displayName,
      kind: form.kind,
      cohort: cohortText ? Number(cohortText) : null,
      department: department || null,
      tierManWon: form.tierManWon ? (Number(form.tierManWon) as HallTier) : null,
      contactName: form.kind === 'org' && contactName ? contactName : null,
      roleLabel: form.kind === 'teacher' && roleLabel ? roleLabel : null,
    },
  };
}
