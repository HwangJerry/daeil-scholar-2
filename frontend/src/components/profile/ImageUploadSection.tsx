// ImageUploadSection — Image preview, file picker, and upload trigger for profile images
import { useRef, useState } from 'react';
import { Camera, Loader2 } from 'lucide-react';
import { cn } from '../../lib/utils';
import { api } from '../../api/client';

interface ImageUploadSectionProps {
  label: string;
  currentUrl: string | null;
  uploadUrl: string;
  onUploaded: (url: string) => void;
  shape?: 'square' | 'card';
}

export function ImageUploadSection({
  label,
  currentUrl,
  uploadUrl,
  onUploaded,
  shape = 'square',
}: ImageUploadSectionProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [previewUrl, setPreviewUrl] = useState<string | null>(currentUrl);
  const [isUploading, setIsUploading] = useState(false);
  const [errorMsg, setErrorMsg] = useState('');

  const isSquare = shape === 'square';
  const containerClass = isSquare
    ? 'relative h-20 w-20 rounded-full overflow-hidden'
    : 'relative h-28 w-48 rounded-xl overflow-hidden';

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    const objectUrl = URL.createObjectURL(file);
    setPreviewUrl(objectUrl);
    setErrorMsg('');
    setIsUploading(true);

    try {
      const formData = new FormData();
      formData.append('file', file);
      const result = await api.upload<{ url: string }>(uploadUrl, formData);
      onUploaded(result.url);
    } catch {
      setErrorMsg('업로드에 실패했습니다.');
      setPreviewUrl(currentUrl);
    } finally {
      setIsUploading(false);
      if (inputRef.current) inputRef.current.value = '';
    }
  };

  return (
    <div className="mb-4">
      <p className="mb-2 text-[13px] font-medium text-text-tertiary">{label}</p>
      <div className="flex items-end gap-3">
        <button
          type="button"
          onClick={() => inputRef.current?.click()}
          disabled={isUploading}
          className={cn(containerClass, 'bg-surface-alt border border-border focus:outline-none focus:ring-2 focus:ring-primary/20 group')}
          aria-label={`${label} 변경`}
        >
          {previewUrl ? (
            <img src={previewUrl} alt={label} className="h-full w-full object-cover" />
          ) : (
            <Camera size={isSquare ? 24 : 28} className="absolute inset-0 m-auto text-text-tertiary" />
          )}
          {isUploading ? (
            <div className="absolute inset-0 flex items-center justify-center bg-black/40">
              <Loader2 size={20} className="animate-spin text-white" />
            </div>
          ) : (
            <div className="absolute inset-0 flex items-center justify-center bg-black/0 group-hover:bg-black/30 transition-colors">
              <Camera size={16} className="text-white opacity-0 group-hover:opacity-100 transition-opacity" />
            </div>
          )}
        </button>

        <button
          type="button"
          onClick={() => inputRef.current?.click()}
          disabled={isUploading}
          className="text-xs text-primary underline-offset-2 hover:underline disabled:opacity-50"
        >
          변경
        </button>

        <input
          ref={inputRef}
          type="file"
          accept="image/*"
          className="sr-only"
          onChange={handleFileChange}
        />
      </div>
      {errorMsg && <p className="mt-1.5 text-xs text-error">{errorMsg}</p>}
    </div>
  );
}
