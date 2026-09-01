import type { TextareaHTMLAttributes } from "react";

import { cn } from "../../lib/utils";

/** Props for the shared shadcn-style textarea primitive. */
export type TextareaProps = TextareaHTMLAttributes<HTMLTextAreaElement>;

/** A monospace textarea for request bodies and raw API output. */
export function Textarea({ className, ...props }: TextareaProps) {
  return <textarea className={cn("ui-textarea", className)} {...props} />;
}
