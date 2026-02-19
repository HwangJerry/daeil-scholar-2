// Badge — semantic status indicator with color variants
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "../../lib/utils";

const badgeVariants = cva(
  "inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium",
  {
    variants: {
      variant: {
        default: "bg-primary-light text-primary",
        secondary: "bg-background text-text-tertiary",
        success: "bg-success-subtle text-success-text",
        warning: "bg-warning-subtle text-warning-text",
        destructive: "bg-error-light text-error",
        pinned: "bg-error-subtle text-error-text",
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
