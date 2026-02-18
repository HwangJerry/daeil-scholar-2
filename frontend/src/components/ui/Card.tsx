// Card — reusable card container with variant-based elevation and padding
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "../../lib/utils";

const cardVariants = cva(
  "rounded-xl bg-surface border transition-all duration-200",
  {
    variants: {
      variant: {
        default: "border-border-subtle shadow-card",
        elevated: "border-transparent shadow-md",
        interactive:
          "border-border-subtle shadow-card hover:shadow-card-hover hover:-translate-y-0.5 cursor-pointer",
        ghost: "border-transparent shadow-none bg-transparent",
      },
      padding: {
        none: "",
        sm: "p-3",
        md: "p-4",
        lg: "p-5",
      },
    },
    defaultVariants: {
      variant: "default",
      padding: "md",
    },
  }
);

interface CardProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof cardVariants> {}

export function Card({ className, variant, padding, ...props }: CardProps) {
  return (
    <div
      className={cn(cardVariants({ variant, padding, className }))}
      {...props}
    />
  );
}
