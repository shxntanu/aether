import { Badge } from "@/components/ui/badge";

/** Renders the user-facing lifecycle badge for documents and members. */
export function Status({ status }: { status: string }) {
  if (status === "ready")
    return <Badge className="vault-status vault-status--ready">Ready</Badge>;
  if (status === "failed")
    return (
      <Badge className="vault-status vault-status--failed">
        Needs attention
      </Badge>
    );
  if (status === "deleted")
    return (
      <Badge className="vault-status vault-status--failed">Deleted</Badge>
    );
  return (
    <Badge className="vault-status vault-status--processing">
      Processing
    </Badge>
  );
}
