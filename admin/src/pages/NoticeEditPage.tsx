// NoticeEditPage — composes notice form hooks with editor or legacy HTML viewer
import { useParams, useNavigate } from 'react-router-dom';
import { ArrowLeft } from 'lucide-react';
import { Button } from '../components/ui/Button.tsx';
import { Input } from '../components/ui/Input.tsx';
import { MarkdownEditor } from '../components/editor/MarkdownEditor.tsx';
import { HtmlContent } from '../components/common/HtmlContent.tsx';
import { useNoticeDetail } from '../hooks/useNoticeDetail.ts';
import { useNoticeForm } from '../hooks/useNoticeForm.ts';
import { useNoticeMutations } from '../hooks/useNoticeMutations.ts';
import type { NoticeDetail } from '../types/api.ts';

export function NoticeEditPage() {
  const { seq } = useParams<{ seq: string }>();
  const { data: notice } = useNoticeDetail(seq);

  return (
    <NoticeEditForm key={notice?.seq ?? 'new'} seq={seq} notice={notice} />
  );
}

function NoticeEditForm({
  seq,
  notice,
}: {
  seq: string | undefined;
  notice: NoticeDetail | undefined;
}) {
  const navigate = useNavigate();
  const form = useNoticeForm(notice);
  const { save, isSaving } = useNoticeMutations(seq);

  const isLegacy = notice?.contentFormat === 'LEGACY';

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-3">
        <Button variant="ghost" size="icon" onClick={() => navigate('/notice')}>
          <ArrowLeft className="h-5 w-5" />
        </Button>
        <h2 className="text-xl font-bold text-dark-slate">
          {seq ? '공지 수정' : '공지 작성'}
        </h2>
      </div>

      {isLegacy ? (
        <div>
          <div className="mb-4 rounded-xl bg-warning-subtle p-4 text-sm text-warning-text">
            이 글은 기존 시스템에서 작성되었습니다.
            Markdown 에디터로 수정하려면 기존 내용을 복사하여 새 글로 작성해주세요.
          </div>
          <HtmlContent html={notice?.contentHtml ?? ''} />
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

          <label className="flex items-center gap-2 text-sm text-dark-slate">
            <input
              type="checkbox"
              checked={form.isPinned}
              onChange={(e) => form.setIsPinned(e.target.checked)}
              className="h-4 w-4 rounded border-border text-royal-indigo focus:ring-royal-indigo"
            />
            상단 고정
          </label>

          <div className="flex justify-end gap-3">
            <Button variant="outline" onClick={() => navigate('/notice')}>취소</Button>
            <Button
              onClick={() => save(form.subject, form.contentMd, form.isPinned)}
              disabled={isSaving || !form.isValid}
            >
              {isSaving ? '저장 중...' : '저장'}
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
