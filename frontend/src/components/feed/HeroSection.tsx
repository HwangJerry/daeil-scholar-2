// Hero banner displaying the latest pinned/top notice with large image
import { Link } from 'react-router-dom';
import { useHeroNotice } from '../../hooks/useHeroNotice';

export function HeroSection() {
  const { data: hero, isLoading, isError } = useHeroNotice();

  if (isLoading) {
    return <div className="h-56 rounded-xl skeleton-shimmer md:h-72" />;
  }

  if (isError || !hero) return null;

  return (
    <Link
      to={`/post/${hero.seq}`}
      className="group relative block overflow-hidden rounded-xl bg-text-primary animate-fade-in-up"
    >
      {hero.thumbnailUrl ? (
        <img
          src={hero.thumbnailUrl}
          alt={hero.subject}
          className="h-56 w-full object-cover opacity-80 transition-transform duration-300 group-hover:scale-[1.03] md:h-72"
        />
      ) : (
        <div className="h-56 bg-gradient-to-br from-primary to-violet-600 md:h-72" />
      )}
      <div className="absolute inset-0 bg-gradient-to-t from-black/60 via-black/20 to-transparent" />
      <div className="absolute bottom-0 left-0 right-0 p-5 animate-fade-in-up stagger-2">
        <h2 className="mb-1 text-lg font-bold text-white md:text-xl">{hero.subject}</h2>
        {hero.summary && (
          <p className="line-clamp-2 text-[13px] text-white/70">{hero.summary}</p>
        )}
        <div className="mt-2 flex items-center gap-3 text-xs text-white/50">
          <span>{hero.regDate}</span>
          <span>조회 {hero.hit}</span>
        </div>
      </div>
    </Link>
  );
}
