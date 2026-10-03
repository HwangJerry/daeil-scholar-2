// NoticeOfficialProfileCheckbox — opt-in to publish a notice under the foundation's official profile
interface Props {
  checked: boolean;
  onChange: (checked: boolean) => void;
  disabled?: boolean;
}

export function NoticeOfficialProfileCheckbox({ checked, onChange, disabled }: Props) {
  return (
    <label className="flex items-center gap-2 text-sm text-dark-slate">
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        disabled={disabled}
        className="h-4 w-4 rounded border-border text-royal-indigo focus:ring-royal-indigo"
      />
      공식 프로필 사용 (대일외고장학회 닉네임+아이콘)
    </label>
  );
}
