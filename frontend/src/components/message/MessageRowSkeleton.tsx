// MessageRowSkeleton — Loading placeholder for inbox and outbox message rows
import { Bone } from '../ui/Skeleton';

export function MessageRowSkeleton() {
  return (
    <div className="rounded-2xl border border-border-subtle p-4">
      <div className="flex items-start gap-3">
        <Bone className="w-[18px] h-[18px] flex-shrink-0 rounded mt-0.5" />
        <div className="flex-1 space-y-2">
          <Bone className="h-4 w-24" />
          <Bone className="h-3 w-full" />
          <Bone className="h-3 w-3/4" />
          <Bone className="h-2.5 w-16 mt-1" />
        </div>
      </div>
    </div>
  );
}
