// PhoneMatchBanner — Inline banner shown when a typed phone matches an existing member, offering to switch to merge mode.
import { Button } from '../ui/Button';

interface PhoneMatchBannerProps {
  matchedName: string;
  matchedFN?: string;
  onConfirm: () => void;
}

function maskName(name: string): string {
  if (!name) return '';
  if (name.length <= 1) return name;
  if (name.length === 2) return name[0] + '*';
  return name[0] + '*'.repeat(name.length - 2) + name[name.length - 1];
}

function MatchProfileLabel({ name, fn }: { name: string; fn?: string }) {
  const maskedName = maskName(name);
  const fnLabel = fn ? ` · ${fn}기` : '';

  return (
    <>
      {maskedName}
      {fnLabel}
    </>
  );
}

export function PhoneMatchBanner({ matchedName, matchedFN, onConfirm }: PhoneMatchBannerProps) {
  return (
    <div className="mb-4 rounded-lg bg-warning-subtle px-4 py-3 text-sm text-warning-text">
      <p className="font-medium">
        이미 가입된 번호입니다.
        <span className="ml-1 font-normal">
          (<MatchProfileLabel name={matchedName} fn={matchedFN} />)
        </span>
      </p>
      <p className="mt-1 text-xs text-warning-text/80">
        기존 계정에 카카오를 연결하고 회원 정보를 이어받으시겠어요?
      </p>
      <div className="mt-3 flex justify-end">
        <Button type="button" size="sm" onClick={onConfirm}>
          통합 회원가입으로 진행
        </Button>
      </div>
    </div>
  );
}

interface PhoneMatchBottomSheetContentProps {
  matchedName: string;
  matchedFN?: string;
  onConfirm: () => void;
  onContinue: () => void;
}

export function PhoneMatchBottomSheetContent({
  matchedName,
  matchedFN,
  onConfirm,
  onContinue,
}: PhoneMatchBottomSheetContentProps) {
  return (
    <div className="px-5 pb-[calc(1.25rem+env(safe-area-inset-bottom))] pt-2">
      <div className="space-y-3">
        <div>
          <h2 className="text-lg font-semibold text-text-primary">기존 계정을 찾았습니다</h2>
          <p className="mt-2 text-sm leading-6 text-text-secondary">
            입력한 번호와 일치하는 계정이 있습니다. 카카오 계정을 연결하고 기존 정보를 이어받을 수 있습니다.
          </p>
        </div>

        <div className="rounded-lg border border-warning-text/20 bg-warning-subtle px-4 py-3 text-sm text-warning-text">
          <span className="font-medium">
            <MatchProfileLabel name={matchedName} fn={matchedFN} />
          </span>
        </div>
      </div>

      <div className="mt-5 space-y-2">
        <Button type="button" className="w-full" onClick={onConfirm}>
          통합 회원가입으로 진행
        </Button>
        <Button type="button" variant="outline" className="w-full" onClick={onContinue}>
          신규 가입 계속하기
        </Button>
      </div>
    </div>
  );
}
