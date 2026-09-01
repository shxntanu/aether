import type { LabelHTMLAttributes } from "react";

import { cn } from "../../lib/utils";

/** Props for the shared shadcn-style form label. */
export type LabelProps = LabelHTMLAttributes<HTMLLabelElement>;

/** An associated form label with consistent type and spacing. */
export function Label({ className, ...props }: LabelProps) {
  return <label className={cn("ui-label", className)} {...props} />;
}
