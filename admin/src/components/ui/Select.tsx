// Dropdown select field with consistent admin styling
import * as React from "react";
import { cn } from "../../lib/utils.ts";

const Select = React.forwardRef<HTMLSelectElement, React.SelectHTMLAttributes<HTMLSelectElement>>(
  ({ className, children, ...props }, ref) => {
    return (
      <select
        className={cn(
          "flex h-10 w-full rounded-xl border border-border bg-white px-3 py-2 text-sm",
          "focus:outline-none focus:ring-2 focus:ring-royal-indigo focus:border-transparent",
          "disabled:cursor-not-allowed disabled:opacity-50",
          className,
        )}
        ref={ref}
        {...props}
      >
        {children}
      </select>
    );
  }
);
Select.displayName = "Select";

export { Select };
