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

export function PhoneMatchBanner({ matchedName, matchedFN, onConfirm }: PhoneMatchBannerProps) {
  const maskedName = maskName(matchedName);
  const fnLabel = matchedFN ? ` · ${matchedFN}기` : '';

  return (
    <div className="mb-4 rounded-lg bg-warning-subtle px-4 py-3 text-sm text-warning-text">
      <p className="font-medium">
        이미 가입된 번호입니다.
        <span className="ml-1 font-normal">
          ({maskedName}
          {fnLabel})
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
