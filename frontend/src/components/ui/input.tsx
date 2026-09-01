import type { InputHTMLAttributes } from "react";

import { cn } from "../../lib/utils";

/** Props for the shared shadcn-style input primitive. */
export type InputProps = InputHTMLAttributes<HTMLInputElement>;

/** A styled text, file, or numeric input with accessible focus treatment. */
export function Input({ className, type, ...props }: InputProps) {
  return <input type={type} className={cn("ui-input", className)} {...props} />;
}
