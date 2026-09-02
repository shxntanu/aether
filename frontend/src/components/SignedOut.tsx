import { Shield } from "lucide-react";

import { Brand } from "@/components/Brand";

/** Renders the sign-in screen and disabled-account state. */
export function SignedOut({ disabled = false }: { disabled?: boolean }) {
  return (
    <main className="vault-login">
      <section className="vault-login__sheet">
        <Brand />
        <h1>
          {disabled ? "Access needs attention." : "Your private archive."}
        </h1>
        <p>
          {disabled
            ? "This account is not currently enabled for Aether. Contact an administrator to restore access."
            : "A considered home for the documents your team needs to keep close, with original files preserved."}
        </p>
        {!disabled && (
          <a
            className="vault-button vault-button--primary"
            href="/auth/google/start"
          >
            <Shield size={15} aria-hidden="true" /> Continue with Google
          </a>
        )}
        <p className="vault-login__note">
          Aether is private by default. Only allowlisted members can enter.
        </p>
      </section>
    </main>
  );
}
