import { Slot } from "@radix-ui/react-slot";
import { cva, type VariantProps } from "class-variance-authority";
import type { ButtonHTMLAttributes } from "react";

import { cn } from "../../lib/utils";

const buttonVariants = cva("ui-button", {
  variants: {
    variant: {
      default: "ui-button--default",
      outline: "ui-button--outline",
      ghost: "ui-button--ghost",
      destructive: "ui-button--destructive",
    },
    size: {
      default: "ui-button--default-size",
      sm: "ui-button--sm",
      icon: "ui-button--icon",
      "icon-sm": "ui-button--icon-sm",
    },
  },
  defaultVariants: { variant: "default", size: "default" },
});

/** Props for the shared shadcn-style button primitive. */
export interface ButtonProps
  extends ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  /** Renders the button styles onto the child element when true. */
  asChild?: boolean;
}

/** A compact action button with consistent focus and disabled states. */
export function Button({ className, variant, size, asChild = false, ...props }: ButtonProps) {
  const Comp = asChild ? Slot : "button";
  return <Comp className={cn(buttonVariants({ variant, size, className }))} {...props} />;
}
