// PostAuthor — post byline: the foundation emblem and official name, or the author's name
import { OFFICIAL_PROFILE_EMBLEM_SRC, OFFICIAL_PROFILE_NAME } from './officialProfile';

interface PostAuthorProps {
  regName: string;
  officialProfile: boolean;
}

export function PostAuthor({ regName, officialProfile }: PostAuthorProps) {
  if (!officialProfile) {
    return <span className="text-text-secondary font-medium">{regName}</span>;
  }
  return (
    <span className="inline-flex items-center gap-1.5 text-text-secondary font-medium">
      <img
        src={OFFICIAL_PROFILE_EMBLEM_SRC}
        alt=""
        aria-hidden="true"
        className="h-5 w-5 shrink-0 rounded-full object-cover"
      />
      {OFFICIAL_PROFILE_NAME}
    </span>
  );
}
