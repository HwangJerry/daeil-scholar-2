// MarkdownEditor — split-view Markdown editor with image drag-drop and paste support
import MDEditor from '@uiw/react-md-editor';
import { uploadImage } from './uploadImage.ts';
import { isImageFile } from './isImageFile.ts';

interface Props {
  value: string;
  onChange: (val: string) => void;
}

export function MarkdownEditor({ value, onChange }: Props) {
  const handleDrop = async (e: React.DragEvent) => {
    const file = e.dataTransfer.files[0];
    if (!file || !isImageFile(file)) return;
    e.preventDefault();
    const url = await uploadImage(file);
    onChange(value + `\n![이미지](${url})\n`);
  };

  const handlePaste = async (e: React.ClipboardEvent) => {
    const items = e.clipboardData.items;
    for (const item of items) {
      if (item.type.startsWith('image/')) {
        e.preventDefault();
        const file = item.getAsFile();
        if (file) {
          const url = await uploadImage(file);
          onChange(value + `\n![이미지](${url})\n`);
        }
        return;
      }
    }
  };

  return (
    <div onDrop={(e) => void handleDrop(e)} onPaste={(e) => void handlePaste(e)}>
      <MDEditor
        value={value}
        onChange={(val) => onChange(val ?? '')}
        height={500}
        preview="live"
      />
    </div>
  );
}
