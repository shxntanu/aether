import { navigate } from "@/lib/vault";

/** Renders the Aether wordmark and routes clicks back to the library. */
export function Brand() {
  return (
    <a
      className="vault-brand"
      href="/library"
      aria-label="Aether archive home"
      onClick={(event) => {
        event.preventDefault();
        navigate("/library");
      }}
    >
      <span className="vault-brand__mark">A</span>
      <span className="vault-brand__name">AETHER</span>
    </a>
  );
}
