// DisclosureEditPage — composes disclosure form hooks with editor or legacy HTML viewer
import { useParams, useNavigate } from 'react-router-dom';
import { ArrowLeft, Trash2 } from 'lucide-react';
import { Button } from '../components/ui/Button.tsx';
import { Input } from '../components/ui/Input.tsx';
import { MarkdownEditor } from '../components/editor/MarkdownEditor.tsx';
import { HtmlContent } from '../components/common/HtmlContent.tsx';
import { ConfirmDialog } from '../components/ui/ConfirmDialog.tsx';
import { useDisclosureDetail } from '../hooks/useDisclosureDetail.ts';
import { useDisclosureForm } from '../hooks/useDisclosureForm.ts';
import { useDisclosureMutations } from '../hooks/useDisclosureMutations.ts';
import { useDisclosureDelete } from '../hooks/useDisclosureDelete.ts';
import { useNoticeAttachments } from '../hooks/useNoticeAttachments.ts';
import { AttachmentDropzone } from '../components/editor/AttachmentDropzone.tsx';
import { AttachmentList } from '../components/editor/AttachmentList.tsx';
import type { NoticeDetail } from '../types/api.ts';

export function DisclosureEditPage() {
  const { seq } = useParams<{ seq: string }>();
  const { data: notice } = useDisclosureDetail(seq);

  return (
    <DisclosureEditForm key={notice?.seq ?? 'new'} seq={seq} notice={notice} />
  );
}

function DisclosureEditForm({
  seq,
  notice,
}: {
  seq: string | undefined;
  notice: NoticeDetail | undefined;
}) {
  const navigate = useNavigate();
  const form = useDisclosureForm(notice);
  const { save, isSaving, deleteDisclosure } = useDisclosureMutations(seq);
  const del = useDisclosureDelete(deleteDisclosure);
  const att = useNoticeAttachments(notice?.files);

  const isLegacy = notice?.contentFormat === 'LEGACY';

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-3">
        <Button variant="ghost" size="icon" onClick={() => navigate('/disclosure')}>
          <ArrowLeft className="h-5 w-5" />
        </Button>
        <h2 className="text-xl font-bold text-dark-slate">
          {seq ? '공시 자료 수정' : '공시 자료 작성'}
        </h2>
      </div>

      {isLegacy ? (
        <div>
          <div className="mb-4 rounded-xl bg-warning-subtle p-4 text-sm text-warning-text">
            이 글은 기존 시스템에서 작성되었습니다.
            Markdown 에디터로 수정하려면 기존 내용을 복사하여 새 글로 작성해주세요.
          </div>
          <HtmlContent html={notice?.contentHtml ?? ''} />
          {seq && (
            <div className="mt-4 flex justify-start">
              <Button variant="ghost" className="text-error-text" onClick={() => del.openDialog(Number(seq))}>
                <Trash2 className="mr-1 h-4 w-4" />
                삭제
              </Button>
            </div>
          )}
        </div>
      ) : (
        <div className="space-y-4">
          <Input
            aria-label="제목"
            placeholder="제목을 입력하세요"
            value={form.subject}
            onChange={(e) => form.setSubject(e.target.value)}
            className="text-lg font-medium"
          />

          <MarkdownEditor value={form.contentMd} onChange={form.setContentMd} />

          <div className="space-y-2">
            <h3 className="text-sm font-medium text-dark-slate">첨부파일</h3>
            <AttachmentDropzone onUploaded={att.addUploaded} disabled={isSaving} />
            <AttachmentList
              attachments={att.attachments}
              onRemove={att.remove}
              disabled={isSaving}
            />
          </div>

          <div className="flex justify-between gap-3">
            {seq && (
              <Button variant="ghost" className="text-error-text" onClick={() => del.openDialog(Number(seq))}>
                <Trash2 className="mr-1 h-4 w-4" />
                삭제
              </Button>
            )}
            <div className="flex gap-3 ml-auto">
              <Button variant="outline" onClick={() => navigate('/disclosure')}>취소</Button>
              <Button
                onClick={() => save(form.subject, form.contentMd, att.fSeqs)}
                disabled={isSaving || !form.isValid}
              >
                {isSaving ? '저장 중...' : '저장'}
              </Button>
            </div>
          </div>
        </div>
      )}

      <ConfirmDialog
        open={del.dialogOpen}
        onOpenChange={(open) => { if (!open) del.closeDialog(); }}
        title="공시 자료 삭제"
        description="정말 삭제하시겠습니까?"
        confirmLabel="삭제"
        variant="destructive"
        onConfirm={del.handleConfirm}
      />
    </div>
  );
}
