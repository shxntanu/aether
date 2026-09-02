/** Renders the user-facing lifecycle badge for documents and members. */
export function Status({ status }: { status: string }) {
  if (status === "ready")
    return <span className="vault-status vault-status--ready">Ready</span>;
  if (status === "failed")
    return (
      <span className="vault-status vault-status--failed">Needs attention</span>
    );
  if (status === "deleted")
    return <span className="vault-status vault-status--failed">Deleted</span>;
  return (
    <span className="vault-status vault-status--processing">Processing</span>
  );
}
