import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { PostContent } from '../components/post/PostContent';
import type { NoticeDetail } from '../types/api';

vi.mock('../components/post/PostHeader', () => ({
  PostHeader: () => <header>공지 제목</header>,
}));
vi.mock('../components/post/PostBody', () => ({
  PostBody: () => <div>공지 본문</div>,
}));
vi.mock('../components/post/PostEngagement', () => ({
  PostEngagement: () => <div>PUBLIC_ENGAGEMENT_SHOULD_NOT_RENDER</div>,
}));

const post: NoticeDetail = {
  seq: 10,
  subject: '공지 제목',
  contentHtml: '<p>공지 본문</p>',
  contentFormat: 'MARKDOWN',
  summary: '요약',
  thumbnailUrl: null,
  regDate: '2026-07-29T00:00:00Z',
  regName: '장학회',
  hit: 3,
  likeCnt: 2,
  commentCnt: 1,
  userLiked: false,
  files: [],
};

describe('public notice detail', () => {
  it('renders notice content without likes or comments', () => {
    render(<PostContent post={post} />);

    expect(screen.getByText('공지 제목')).toBeInTheDocument();
    expect(screen.getByText('공지 본문')).toBeInTheDocument();
    expect(screen.queryByText('PUBLIC_ENGAGEMENT_SHOULD_NOT_RENDER')).not.toBeInTheDocument();
  });
});
