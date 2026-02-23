// FeedCardSkeleton — Shimmer skeleton matching the unified feed card layout
import { Bone } from '../ui/Skeleton';

export function FeedCardSkeleton() {
  return (
    <div className="rounded-2xl bg-surface border border-border-subtle overflow-hidden">
      <div className="px-5 pt-5 pb-4 space-y-3">
        {/* Meta: plain-text category · date */}
        <div className="flex items-center gap-2">
          <Bone className="h-3 w-8" />
          <Bone className="h-3 w-14" />
        </div>

        {/* Title */}
        <div className="space-y-2">
          <Bone className="h-4 w-4/5" />
          <Bone className="h-4 w-3/5" />
        </div>

        {/* Summary */}
        <div className="space-y-1.5">
          <Bone className="h-3 w-full" />
          <Bone className="h-3 w-full" />
          <Bone className="h-3 w-2/3" />
        </div>
      </div>

      {/* Full-bleed image placeholder */}
      <Bone className="h-40 w-full" />

      <div className="border-t border-border-subtle mx-5" />

      {/* Engagement: like + comment left, views right */}
      <div className="flex items-center px-5 py-2.5">
        <div className="flex gap-4">
          <Bone className="h-3 w-10" />
          <Bone className="h-3 w-10" />
        </div>
        <Bone className="h-3 w-10 ml-auto" />
      </div>
    </div>
  );
}
