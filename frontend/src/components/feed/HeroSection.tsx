// HeroSection — Pinned notice hero card with dark navy editorial gradient
import { Link } from 'react-router-dom';
import { Pin } from 'lucide-react';
import { cn } from '../../lib/utils';
import { useHeroNotice } from '../../hooks/useHeroNotice';

export function HeroSection() {
  const { data: hero, isLoading, isError } = useHeroNotice();

  if (isLoading) {
    return <div className="min-h-[260px] rounded-[20px] skeleton-shimmer" />;
  }

  if (isError || !hero) return null;

  return (
    <Link
      to={`/post/${hero.seq}`}
      className={cn(
        'group relative block overflow-hidden rounded-[20px] animate-fade-in-up',
        'bg-gradient-to-br from-hero-from via-hero-via to-hero-to',
        'min-h-[260px]'
      )}
    >
      {/* Ambient glow decoration */}
      <div className="absolute top-0 right-0 w-64 h-64 rounded-full bg-white/5 -translate-y-1/2 translate-x-1/2 blur-3xl pointer-events-none" />

      {/* Thumbnail overlay */}
      {hero.thumbnailUrl && (
        <>
          <img
            src={hero.thumbnailUrl}
            alt={hero.subject}
            className="absolute inset-0 h-full w-full object-cover opacity-20 transition-transform duration-300 group-hover:scale-[1.03]"
          />
          <div className="absolute inset-0 bg-gradient-to-t from-hero-from/90 via-hero-from/40 to-transparent" />
        </>
      )}

      {/* Content */}
      <div className="relative flex flex-col justify-end h-full min-h-[260px] p-6">
        {/* Top label */}
        <div className="flex items-center gap-2 mb-auto pt-1">
          <span className="flex items-center gap-1.5 text-white/60 text-xs font-medium">
            <Pin size={12} className="text-white/50" />
            고정 공지
          </span>
        </div>

        {/* Title and meta */}
        <div className="mt-4">
          <h2 className="mb-2 text-2xl font-bold text-white font-serif leading-snug line-clamp-2 group-hover:text-white/90 transition-colors duration-150">
            {hero.subject}
          </h2>
          {hero.summary && (
            <p className="mb-3 line-clamp-2 text-[13px] text-white/55 leading-relaxed">{hero.summary}</p>
          )}
          <div className="flex items-center gap-3 text-xs text-white/40">
            <span>{hero.regDate}</span>
            <span>조회 {hero.hit}</span>
          </div>
        </div>
      </div>
    </Link>
  );
}
