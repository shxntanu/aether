import { ApiError } from "./api";

/** Options controlling the bounded retry used only for the initial session GET. */
export interface InitialSessionOptions {
  /** Delay between transient attempts; production uses five seconds. */
  retryDelayMs?: number;
  /** Maximum time spent waiting before the final recoverable error. */
  maxElapsedMs?: number;
  /** Called once after the first transient failure so the UI can show wake-up state. */
  onTransientFailure?: () => void;
  /** Cancels pending wake-up work when the owning view unmounts. */
  signal?: AbortSignal;
  /** Injectable timer used by deterministic tests. */
  sleep?: (delayMs: number, signal?: AbortSignal) => Promise<void>;
  /** Injectable monotonic clock used to enforce the total wall-time budget. */
  now?: () => number;
}

/** Retries only an initial session read while a sleeping backend becomes ready. */
export async function loadInitialSession<T>(
  request: (signal?: AbortSignal) => Promise<T>,
  options: InitialSessionOptions = {},
): Promise<T> {
  const retryDelayMs = options.retryDelayMs ?? 5_000;
  const maxElapsedMs = options.maxElapsedMs ?? 90_000;
  const wait = options.sleep ?? sleep;
  const now = options.now ?? (() => performance.now());
  const deadline = now() + maxElapsedMs;
  let wakeStateReported = false;

  for (;;) {
    options.signal?.throwIfAborted();
    const attempt = signalUntil(options.signal, Math.max(0, deadline - now()));
    try {
      return await request(attempt.signal);
    } catch (error) {
      options.signal?.throwIfAborted();
      if (!isTransientSessionFailure(error) || now() >= deadline) {
        throw error;
      }
      if (!wakeStateReported) {
        options.onTransientFailure?.();
        wakeStateReported = true;
      }
      const delayMs = Math.min(retryDelayMs, deadline - now());
      await wait(delayMs, options.signal);
    } finally {
      attempt.dispose();
    }
  }
}

function isTransientSessionFailure(error: unknown): boolean {
  if (!(error instanceof ApiError)) return false;
  if (error.status === 401 || error.status === 403) return false;
  if (error.status === 0 || [502, 503, 504].includes(error.status)) return true;
  return error.status >= 200 && error.status < 300 &&
    error.code === "unexpected_response";
}

function signalUntil(parent: AbortSignal | undefined, delayMs: number) {
  const controller = new AbortController();
  const onParentAbort = () => controller.abort(parent?.reason);
  if (parent?.aborted) onParentAbort();
  else parent?.addEventListener("abort", onParentAbort, { once: true });
  const timer = globalThis.setTimeout(
    () => controller.abort(new DOMException("Session deadline exceeded", "TimeoutError")),
    Math.max(0, Math.ceil(delayMs)),
  );
  return {
    signal: controller.signal,
    dispose: () => {
      globalThis.clearTimeout(timer);
      parent?.removeEventListener("abort", onParentAbort);
    },
  };
}

function sleep(delayMs: number, signal?: AbortSignal): Promise<void> {
  return new Promise((resolve, reject) => {
    const onAbort = () => {
      globalThis.clearTimeout(timer);
      reject(signal?.reason);
    };
    const timer = globalThis.setTimeout(() => {
      signal?.removeEventListener("abort", onAbort);
      resolve();
    }, delayMs);
    signal?.addEventListener("abort", onAbort, { once: true });
  });
}
