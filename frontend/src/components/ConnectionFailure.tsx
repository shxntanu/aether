import { Brand } from "@/components/Brand";

/** Renders the recoverable connection failure state for the vault shell. */
export function ConnectionFailure({ message }: { message: string }) {
  return (
    <main className="vault-login">
      <section className="vault-login__sheet">
        <Brand />
        <h1>Vault unavailable.</h1>
        <p>{message}</p>
        <button
          className="vault-button vault-button--primary"
          type="button"
          onClick={() => window.location.reload()}
        >
          Try again
        </button>
      </section>
    </main>
  );
}
