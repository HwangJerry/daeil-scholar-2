// PostBody — Renders post thumbnail image and HTML content
import { HtmlContent } from '../common/HtmlContent';
import { FileAttachments } from './FileAttachments';
import type { FileAttachment } from '../../types/api';

interface PostBodyProps {
  thumbnailUrl: string | null;
  subject: string;
  contentHtml: string;
  files: FileAttachment[];
}

export function PostBody({ thumbnailUrl, subject, contentHtml, files }: PostBodyProps) {
  return (
    <>
      {thumbnailUrl && (
        <img
          src={thumbnailUrl}
          alt={subject}
          className="mb-4 w-full aspect-video rounded-lg object-cover"
        />
      )}

      <HtmlContent html={contentHtml} className="text-text-secondary" />

      <FileAttachments files={files} />
    </>
  );
}
