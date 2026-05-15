// SignupProfileImageEditor — Avatar preview + replace/remove controls for the pre-signup profile photo (token-gated upload).
import { useRef, useState } from 'react';
import { User } from 'lucide-react';
import { uploadSocialLinkPhoto } from '../../api/auth';
import { ApiClientError } from '../../api/client';

interface Props {
  token: string;
  imageUrl: string | null;
  onChange: (url: string | null) => void;
}

const ACCEPT = 'image/png,image/jpeg,image/webp,image/gif';
const MAX_BYTES = 5 * 1024 * 1024;

export function SignupProfileImageEditor({ token, imageUrl, onChange }: Props) {
  const inputRef = useRef<HTMLInputElement | null>(null);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState('');

  const openPicker = () => {
    if (uploading) return;
    inputRef.current?.click();
  };

  const handleFile = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    e.target.value = '';
    if (!file) return;
    if (file.size > MAX_BYTES) {
      setError('이미지는 5MB 이하만 업로드할 수 있습니다.');
      return;
    }
    setError('');
    setUploading(true);
    try {
      const res = await uploadSocialLinkPhoto(token, file);
      onChange(res.url);
    } catch (err) {
      if (err instanceof ApiClientError) {
        setError(err.message || '이미지 업로드에 실패했습니다.');
      } else {
        setError('이미지 업로드에 실패했습니다.');
      }
    } finally {
      setUploading(false);
    }
  };

  const handleRemove = () => {
    if (uploading) return;
    setError('');
    onChange(null);
  };

  return (
    <div className="mb-4 flex flex-col items-center gap-2">
      <button
        type="button"
        onClick={openPicker}
        disabled={uploading}
        className="group relative h-20 w-20 overflow-hidden rounded-full border border-border bg-surface-muted transition focus:outline-none focus:ring-2 focus:ring-primary/30 disabled:opacity-60"
        aria-label="프로필 이미지 변경"
      >
        {imageUrl ? (
          <img
            src={imageUrl}
            alt="프로필 이미지"
            className="h-full w-full object-cover"
            referrerPolicy="no-referrer"
          />
        ) : (
          <div className="flex h-full w-full items-center justify-center text-text-placeholder">
            <User size={32} />
          </div>
        )}
      </button>
      <div className="flex items-center gap-3 text-xs">
        <button
          type="button"
          onClick={openPicker}
          disabled={uploading}
          className="text-primary hover:underline disabled:text-text-placeholder disabled:no-underline"
        >
          {uploading ? '업로드 중...' : '사진 변경'}
        </button>
        {imageUrl && !uploading && (
          <>
            <span className="text-border">|</span>
            <button
              type="button"
              onClick={handleRemove}
              className="text-text-placeholder hover:text-error-text"
            >
              이미지 제거
            </button>
          </>
        )}
      </div>
      {error && <p className="text-xs text-error-text">{error}</p>}
      <input
        ref={inputRef}
        type="file"
        accept={ACCEPT}
        className="hidden"
        onChange={handleFile}
      />
    </div>
  );
}
