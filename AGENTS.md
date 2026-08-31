# Aether

## Documentation

- Every public code surface in every language MUST have documentation appropriate to that language, including exported APIs, types, functions, methods, fields, constants, modules, and packages.
- Documentation and comments MUST explain behavior, constraints, and non-obvious rationale; do not restate code, and update them when behavior changes.

## Design and Maintainability

- Prefer the simplest clear design that solves the current requirement. Avoid speculative abstractions, dead code, and unnecessary dependencies.
- Keep changes focused. Do not mix unrelated refactors, formatting churn, or behavior changes into a feature or fix.
- Preserve existing conventions within the surrounding package or module unless there is a documented reason to change them.
- Treat compatibility as a requirement: do not break existing callers, persisted data, configuration, or user-visible behavior without an explicit migration plan.

## Correctness and Testing

- Every behavior change MUST add or update tests that exercise the observable contract, including important boundary conditions and failure paths.
- Every bug fix MUST include a regression test unless the behavior cannot be tested meaningfully; explain that exception in the change description.
- Tests MUST verify behavior rather than implementation details and MUST NOT be weakened, deleted, skipped, or made order-dependent merely to pass.
- Handle errors explicitly. Do not silently ignore failures, discard diagnostic context, or use panic/force-unwrap behavior for recoverable conditions.
- Before completing a change, run the repository's relevant formatter, linter, type checker, build, and test commands. Use the commands documented in `Makefile` and package scripts.

## Security

- Treat all external input as untrusted. Validate it at the server or other trust boundary using allowlists, type/format checks, and sensible size/range limits.
- Client-side validation is for user experience only and MUST NOT replace server-side validation or authorization.
- Use parameterized database queries and context-appropriate output encoding; do not construct commands, SQL, or HTML by concatenating untrusted input.
- Never commit credentials, tokens, private keys, or sensitive personal data. Do not log secrets or sensitive request data.

## Go

- New and modified Go source MUST be formatted with `gofmt`, pass `go vet`, and follow idiomatic Go naming and error-handling conventions.
- Every new or modified Go source line MUST be no longer than 100 characters. Refactor or split long expressions; keep unavoidable long literals intact only when splitting would change their value.
- Prefer standard-library solutions and small, explicit interfaces. Add a dependency only when its benefit justifies its long-term maintenance and security cost.
- Pass `context.Context` through I/O and request paths, honor cancellation, and avoid leaking goroutines, timers, files, or database resources.

## TypeScript and React

- Keep TypeScript type-safe; do not introduce `any` or type assertions to bypass errors without a documented, justified boundary.
- Components MUST remain accessible: use semantic HTML, labels, keyboard support, and appropriate names/roles for interactive controls.
- Keep the `pnpm-lock.yaml` file synchronized with dependency changes; use the repository's `pnpm` scripts rather than introducing another package manager.

## Generated and Configuration Files

- Do not hand-edit generated files. Change their source or generator and regenerate them.
- Configuration and migration changes MUST account for existing deployments and include a safe rollback or migration path when applicable.
