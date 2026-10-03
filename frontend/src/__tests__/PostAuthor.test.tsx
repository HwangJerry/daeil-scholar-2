// PostAuthor.test — notice detail byline: official profile emblem + name vs. the author's name
import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { PostHeader } from '../components/post/PostHeader';

const header = { subject: '장학금 안내', regName: '홍길동', regDate: '2026-10-01T10:00:00+09:00', hit: 3 };

describe('post byline', () => {
  it('shows the foundation emblem and official name for an official post', () => {
    const { container } = render(<PostHeader {...header} regName="대일외고장학회" officialProfile />);

    expect(screen.getByText('대일외고장학회')).toBeInTheDocument();
    expect(container.querySelector('img[src="/logo.png"]')).not.toBeNull();
  });

  it('shows the official name even if regName disagrees', () => {
    render(<PostHeader {...header} officialProfile />);

    expect(screen.getByText('대일외고장학회')).toBeInTheDocument();
    expect(screen.queryByText('홍길동')).not.toBeInTheDocument();
  });

  it('shows the author name without the emblem when officialProfile is false or missing', () => {
    const { container, rerender } = render(<PostHeader {...header} officialProfile={false} />);
    expect(screen.getByText('홍길동')).toBeInTheDocument();
    expect(container.querySelector('img')).toBeNull();

    rerender(<PostHeader {...header} />);
    expect(screen.getByText('홍길동')).toBeInTheDocument();
    expect(container.querySelector('img')).toBeNull();
  });
});
