// Multi-line text area with consistent admin styling
import * as React from "react";
import { cn } from "../../lib/utils.ts";

const Textarea = React.forwardRef<HTMLTextAreaElement, React.TextareaHTMLAttributes<HTMLTextAreaElement>>(
  ({ className, ...props }, ref) => {
    return (
      <textarea
        className={cn(
          "flex w-full rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm min-h-[80px]",
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
Textarea.displayName = "Textarea";

export { Textarea };
