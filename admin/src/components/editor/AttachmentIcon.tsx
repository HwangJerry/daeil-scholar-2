// AttachmentIcon — renders a lucide icon based on the attachment file extension
import { File, FileText, FileImage, FileSpreadsheet, FileArchive } from 'lucide-react';

interface AttachmentIconProps {
  fileName: string;
  className?: string;
}

export function AttachmentIcon({ fileName, className }: AttachmentIconProps) {
  const ext = fileName.split('.').pop()?.toLowerCase() ?? '';
  switch (ext) {
    case 'pdf':
    case 'doc':
    case 'docx':
    case 'hwp':
    case 'hwpx':
    case 'ppt':
    case 'pptx':
    case 'txt':
      return <FileText className={className} />;
    case 'jpg':
    case 'jpeg':
    case 'png':
    case 'gif':
    case 'webp':
      return <FileImage className={className} />;
    case 'xls':
    case 'xlsx':
    case 'csv':
      return <FileSpreadsheet className={className} />;
    case 'zip':
      return <FileArchive className={className} />;
    default:
      return <File className={className} />;
  }
}
