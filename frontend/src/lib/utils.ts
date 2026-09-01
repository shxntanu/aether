import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

/** Combines conditional class names using the shadcn component convention. */
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}
