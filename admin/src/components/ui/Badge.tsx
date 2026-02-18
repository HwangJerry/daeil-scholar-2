// CVA-based badge component for status indicators, tier labels, and format tags
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "../../lib/utils.ts";

const badgeVariants = cva(
  "inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium transition-colors",
  {
    variants: {
      variant: {
        default: "bg-royal-indigo/10 text-royal-indigo",
        success: "bg-emerald-100 text-emerald-700",
        warning: "bg-amber-100 text-amber-700",
        danger: "bg-red-100 text-red-700",
        muted: "bg-slate-100 text-cool-gray",
        premium: "bg-gradient-to-r from-amber-400 to-orange-500 text-white",
        gold: "bg-amber-100 text-amber-800",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  }
);

interface BadgeProps
  extends React.HTMLAttributes<HTMLSpanElement>,
    VariantProps<typeof badgeVariants> {}

export function Badge({ className, variant, ...props }: BadgeProps) {
  return (
    <span className={cn(badgeVariants({ variant, className }))} {...props} />
  );
}
