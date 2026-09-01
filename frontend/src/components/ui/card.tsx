import type { HTMLAttributes } from "react";

import { cn } from "../../lib/utils";

/** Props for a shadcn-style surface container. */
export type CardProps = HTMLAttributes<HTMLDivElement>;

/** A bordered surface used to group related API operations. */
export function Card({ className, ...props }: CardProps) {
  return <div className={cn("ui-card", className)} {...props} />;
}

/** A card header with a title and optional supporting content. */
export function CardHeader({ className, ...props }: CardProps) {
  return <div className={cn("ui-card__header", className)} {...props} />;
}

/** A card title. */
export function CardTitle({ className, ...props }: CardProps) {
  return <h2 className={cn("ui-card__title", className)} {...props} />;
}

/** Supporting description text for a card. */
export function CardDescription({ className, ...props }: CardProps) {
  return <p className={cn("ui-card__description", className)} {...props} />;
}

/** The body area of a card. */
export function CardContent({ className, ...props }: CardProps) {
  return <div className={cn("ui-card__content", className)} {...props} />;
}

/** The action/footer area of a card. */
export function CardFooter({ className, ...props }: CardProps) {
  return <div className={cn("ui-card__footer", className)} {...props} />;
}
