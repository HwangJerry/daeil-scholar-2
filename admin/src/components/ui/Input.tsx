// Text input field with consistent admin styling
import * as React from "react";
import { cn } from "../../lib/utils.ts";

const Input = React.forwardRef<HTMLInputElement, React.InputHTMLAttributes<HTMLInputElement>>(
  ({ className, ...props }, ref) => {
    return (
      <input
        className={cn(
          "flex h-10 w-full rounded-xl border border-border bg-white px-3 py-2 text-sm",
          "placeholder:text-cool-gray",
          "focus:outline-none focus:ring-2 focus:ring-royal-indigo focus:border-transparent",
          "disabled:cursor-not-allowed disabled:opacity-50",
          className,
        )}
        ref={ref}
        {...props}
      />
    );
  }
);
Input.displayName = "Input";

export { Input };
