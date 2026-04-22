// KakaoProfileImagePreview — Circular avatar preview for the Kakao-sourced profile image on the signup form.
import { User } from 'lucide-react';

interface KakaoProfileImagePreviewProps {
  imageUrl: string | null;
}

export function KakaoProfileImagePreview({ imageUrl }: KakaoProfileImagePreviewProps) {
  return (
    <div className="mb-4 flex flex-col items-center gap-2">
      <div className="h-20 w-20 overflow-hidden rounded-full border border-border bg-surface-muted">
        {imageUrl ? (
          <img
            src={imageUrl}
            alt="카카오 프로필 이미지"
            className="h-full w-full object-cover"
            referrerPolicy="no-referrer"
          />
        ) : (
          <div className="flex h-full w-full items-center justify-center text-text-placeholder">
            <User size={32} />
          </div>
        )}
      </div>
      <p className="text-xs text-text-placeholder">
        {imageUrl ? '카카오 프로필 이미지가 적용됩니다' : '프로필 이미지가 설정되지 않았습니다'}
      </p>
    </div>
  );
}
