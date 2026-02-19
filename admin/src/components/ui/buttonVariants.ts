// buttonVariants — CVA variant definitions for Button component
import { cva } from "class-variance-authority";

export const buttonVariants = cva(
  "inline-flex items-center justify-center rounded-2xl text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-royal-indigo focus-visible:ring-offset-2 disabled:opacity-50 disabled:pointer-events-none ring-offset-background",
  {
    variants: {
      variant: {
        default: "bg-royal-indigo text-white hover:bg-royal-indigo/90 shadow-sm",
        destructive: "bg-error text-white hover:bg-error/90",
        outline: "border border-border hover:bg-border-light hover:text-dark-slate",
        secondary: "bg-soft-sky text-royal-indigo hover:bg-soft-sky/80",
        ghost: "hover:bg-border-light hover:text-dark-slate",
        link: "underline-offset-4 hover:underline text-royal-indigo",
      },
      size: {
        default: "h-10 py-2 px-4",
        sm: "h-9 px-3 rounded-xl",
        lg: "h-11 px-8 rounded-2xl",
        icon: "h-10 w-10",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
);
