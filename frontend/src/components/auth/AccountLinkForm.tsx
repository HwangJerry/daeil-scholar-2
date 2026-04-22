// AccountLinkForm — Mode router that dispatches social-signup to the appropriate form (new or merge).
import type { SocialLinkMode } from '../../types/api';
import { AccountLinkNewForm } from './AccountLinkNewForm';
import { AccountLinkMergeForm } from './AccountLinkMergeForm';

interface AccountLinkFormProps {
  token: string;
  mode: SocialLinkMode;
  initialPhone?: string;
}

export function AccountLinkForm({ token, mode, initialPhone }: AccountLinkFormProps) {
  if (mode === 'merge') {
    return <AccountLinkMergeForm token={token} initialPhone={initialPhone ?? ''} />;
  }
  return <AccountLinkNewForm token={token} />;
}
