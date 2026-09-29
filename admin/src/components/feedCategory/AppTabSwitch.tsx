// AppTabSwitch — accessible on/off switch for a feed category's app tab visibility (openYn)
export interface AppTabSwitchProps {
  checked: boolean;
  onChange: (checked: boolean) => void;
  disabled?: boolean;
  label: string;
  title?: string;
}

export function AppTabSwitch({ checked, onChange, disabled, label, title }: AppTabSwitchProps) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      aria-label={label}
      title={title}
      disabled={disabled}
      onClick={() => onChange(!checked)}
      className={`relative inline-flex h-6 w-11 shrink-0 items-center rounded-full transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-royal-indigo disabled:cursor-not-allowed disabled:opacity-50 ${
        checked ? 'bg-royal-indigo' : 'bg-border'
      }`}
    >
      <span
        aria-hidden="true"
        className={`inline-block h-5 w-5 rounded-full bg-white shadow-sm transition-transform ${
          checked ? 'translate-x-5' : 'translate-x-0.5'
        }`}
      />
    </button>
  );
}
