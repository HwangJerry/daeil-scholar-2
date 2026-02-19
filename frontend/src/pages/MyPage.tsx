// MyPage — Profile overview with edit form toggle and account actions
import { useState } from 'react';
import { AuthGuard } from '../components/auth/AuthGuard';
import { ProfileHeader } from '../components/profile/ProfileHeader';
import { ProfileEditForm } from '../components/profile/ProfileEditForm';
import { AccountActions } from '../components/profile/AccountActions';

function MyPageContent() {
  const [showEdit, setShowEdit] = useState(false);

  if (showEdit) {
    return (
      <div className="px-4 py-6 space-y-4">
        <button
          onClick={() => setShowEdit(false)}
          className="text-sm text-text-tertiary hover:text-text-primary transition-colors duration-150"
        >
          &larr; 돌아가기
        </button>
        <ProfileEditForm />
      </div>
    );
  }

  return (
    <div className="space-y-6 pb-20">
      <ProfileHeader onEditClick={() => setShowEdit(true)} />
      <AccountActions />
    </div>
  );
}

export function MyPage() {
  return (
    <AuthGuard>
      <MyPageContent />
    </AuthGuard>
  );
}
