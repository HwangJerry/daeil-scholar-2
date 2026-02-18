import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

/**
 * Combines multiple class names and merges conflicting Tailwind classes.
 * Uses clsx for conditional logic and tailwind-merge for conflict resolution.
 */
export function cn(...inputs: ClassValue[]) {
    return twMerge(clsx(inputs));
}
