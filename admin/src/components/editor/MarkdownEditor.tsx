// MarkdownEditor — split-view Markdown editor with image drag-drop and paste support
import MDEditor from '@uiw/react-md-editor';
import { uploadImage } from './uploadImage.ts';
import { isImageFile } from './isImageFile.ts';
import { useToast } from '../../hooks/useToast.ts';

interface Props {
  value: string;
  onChange: (val: string) => void;
}

export function MarkdownEditor({ value, onChange }: Props) {
  const addToast = useToast((s) => s.addToast);

  const handleDrop = async (e: React.DragEvent) => {
    const file = e.dataTransfer.files[0];
    if (!file || !isImageFile(file)) return;
    e.preventDefault();
    try {
      const url = await uploadImage(file);
      onChange(value + `\n![이미지](${url})\n`);
    } catch {
      addToast({ variant: 'error', title: '이미지 업로드 실패', description: '다시 시도해 주세요.' });
    }
  };

  const handlePaste = async (e: React.ClipboardEvent) => {
    const items = e.clipboardData.items;
    for (const item of items) {
      if (item.type.startsWith('image/')) {
        e.preventDefault();
        const file = item.getAsFile();
        if (file) {
          try {
            const url = await uploadImage(file);
            onChange(value + `\n![이미지](${url})\n`);
          } catch {
            addToast({ variant: 'error', title: '이미지 업로드 실패', description: '다시 시도해 주세요.' });
          }
        }
        return;
      }
    }
  };

  return (
    <div
      onDragOver={(e) => e.preventDefault()}
      onDrop={(e) => void handleDrop(e)}
      onPaste={(e) => void handlePaste(e)}
    >
      <MDEditor
        value={value}
        onChange={(val) => onChange(val ?? '')}
        height={500}
        preview="live"
      />
    </div>
  );
}
