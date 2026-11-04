// ExcelFileDropzone — drag-and-drop or click picker restricted to .xlsx files
import { useCallback, useState, type DragEvent } from 'react';
import { FileSpreadsheet, Upload } from 'lucide-react';

const EXCEL_FILE_EXTENSION = '.xlsx';

interface ExcelFileDropzoneProps {
  file: File | null;
  hint: string;
  onFileSelected: (file: File) => void;
  onInvalidFile: () => void;
  disabled: boolean;
}

export function ExcelFileDropzone({
  file,
  hint,
  onFileSelected,
  onInvalidFile,
  disabled,
}: ExcelFileDropzoneProps) {
  const [isDragOver, setIsDragOver] = useState(false);

  const processFile = useCallback((selectedFile: File | undefined) => {
    if (!selectedFile) return;
    if (!selectedFile.name.toLowerCase().endsWith(EXCEL_FILE_EXTENSION)) {
      onInvalidFile();
      return;
    }
    onFileSelected(selectedFile);
  }, [onFileSelected, onInvalidFile]);

  const handleDrop = useCallback((event: DragEvent<HTMLLabelElement>) => {
    event.preventDefault();
    setIsDragOver(false);
    if (disabled) return;
    processFile(event.dataTransfer.files[0]);
  }, [disabled, processFile]);

  const stateClass = isDragOver
    ? 'border-royal-indigo bg-royal-indigo/5'
    : 'border-border hover:border-royal-indigo/50';

  return (
    <label
      className={`flex cursor-pointer select-none flex-col items-center justify-center gap-2 rounded-xl border-2 border-dashed px-4 py-6 text-sm transition-colors ${stateClass} ${disabled ? 'pointer-events-none opacity-60' : ''}`}
      onDragOver={(event) => { event.preventDefault(); setIsDragOver(true); }}
      onDragLeave={() => setIsDragOver(false)}
      onDrop={handleDrop}
    >
      <input
        type="file"
        accept={EXCEL_FILE_EXTENSION}
        className="hidden"
        disabled={disabled}
        onChange={(event) => {
          processFile(event.target.files?.[0]);
          event.target.value = '';
        }}
      />
      {file ? (
        <FileSpreadsheet className="h-5 w-5 text-royal-indigo" />
      ) : (
        <Upload className="h-5 w-5 text-text-tertiary" />
      )}
      <span className="text-dark-slate">
        {file ? file.name : '엑셀 파일을 드래그하거나 클릭하여 선택'}
      </span>
      <span className="text-xs text-text-placeholder">{hint}</span>
    </label>
  );
}
