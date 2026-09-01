import { useCallback, useEffect, useState, type ComponentProps, type FormEvent, type ReactNode } from "react";
import {
  Activity,
  ArrowUpRight,
  Check,
  CircleAlert,
  Database,
  FileText,
  LoaderCircle,
  LogOut,
  RefreshCw,
  ShieldCheck,
  Tags,
  Upload,
  UserRound,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import "./App.css";

type ApiResult = {
  ok: boolean;
  status: number;
  statusText: string;
  duration: number;
  body: string;
  headers: Record<string, string>;
};

type RequestOptions = RequestInit;

/** Sends a same-origin API request and preserves enough response detail to debug it. */
async function callApi(path: string, options: RequestOptions = {}): Promise<ApiResult> {
  const startedAt = performance.now();
  try {
    const response = await fetch(path, { credentials: "same-origin", ...options });
    const rawBody = await response.text();
    let body = rawBody;
    if (rawBody) {
      try {
        body = JSON.stringify(JSON.parse(rawBody) as unknown, null, 2);
      } catch {
        body = rawBody;
      }
    }
    return {
      ok: response.ok,
      status: response.status,
      statusText: response.statusText,
      duration: Math.round(performance.now() - startedAt),
      body,
      headers: Object.fromEntries(response.headers.entries()),
    };
  } catch (error) {
    return {
      ok: false,
      status: 0,
      statusText: "Network error",
      duration: Math.round(performance.now() - startedAt),
      body: error instanceof Error ? error.message : "The request could not be sent.",
      headers: {},
    };
  }
}

/** Holds loading and response state for one independently testable API area. */
function useApiRequest() {
  const [result, setResult] = useState<ApiResult | null>(null);
  const [loading, setLoading] = useState(false);

  const run = useCallback(async (path: string, options: RequestOptions = {}) => {
    setLoading(true);
    const nextResult = await callApi(path, options);
    setResult(nextResult);
    setLoading(false);
    return nextResult;
  }, []);

  return { result, loading, run };
}

function jsonOptions(method: string, payload: unknown): RequestInit {
  return {
    method,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  };
}

function ActionButton({ children, loading = false, ...props }: ComponentProps<typeof Button> & { loading?: boolean }) {
  return (
    <Button {...props} disabled={loading || props.disabled}>
      {loading && <LoaderCircle className="spin" aria-hidden="true" />}
      {children}
    </Button>
  );
}

function ResultBlock({ result }: { result: ApiResult | null }) {
  const statusClass = result ? (result.ok ? "result-block--ok" : "result-block--error") : "";
  return (
    <div className={`result-block ${statusClass}`} aria-live="polite">
      <div className="result-block__topline">
        <span className="result-block__label">Response</span>
        {result && (
          <span className="result-block__meta">
            {result.status ? `${result.status} ${result.statusText}` : result.statusText} · {result.duration} ms
          </span>
        )}
      </div>
      <pre>{result ? result.body || "(empty response body)" : "No request yet. Run the probe to see the response."}</pre>
      {result && Object.keys(result.headers).length > 0 && (
        <details className="response-headers">
          <summary>Response headers</summary>
          <pre>{JSON.stringify(result.headers, null, 2)}</pre>
        </details>
      )}
    </div>
  );
}

function Field({ id, label, hint, children }: { id?: string; label: string; hint?: string; children: ReactNode }) {
  return (
    <div className="field">
      <Label htmlFor={id}>{label}</Label>
      {children}
      {hint && <span className="field__hint">{hint}</span>}
    </div>
  );
}

function SectionHeading({ icon: Icon, title, description, anchor }: { icon: typeof Activity; title: string; description: string; anchor: string }) {
  return (
    <div className="section-heading" id={anchor}>
      <div className="section-heading__icon"><Icon aria-hidden="true" /></div>
      <div>
        <h2>{title}</h2>
        <p>{description}</p>
      </div>
    </div>
  );
}

/** Renders the practical API bench used to exercise Aether's current backend. */
function App() {
  const health = useApiRequest();
  const session = useApiRequest();
  const documents = useApiRequest();
  const documentLookup = useApiRequest();
  const upload = useApiRequest();
  const metadata = useApiRequest();
  const lifecycle = useApiRequest();
  const tags = useApiRequest();
  const tagCreate = useApiRequest();
  const members = useApiRequest();
  const memberCreate = useApiRequest();
  const memberUpdate = useApiRequest();

  const [documentTag, setDocumentTag] = useState("");
  const [documentMatch, setDocumentMatch] = useState("all");
  const [lookupId, setLookupId] = useState("");
  const [uploadFile, setUploadFile] = useState<File | null>(null);
  const [uploadTitle, setUploadTitle] = useState("");
  const [uploadTags, setUploadTags] = useState("");
  const [idempotencyKey, setIdempotencyKey] = useState(() => `bench-${Date.now()}`);
  const [metadataId, setMetadataId] = useState("");
  const [metadataTitle, setMetadataTitle] = useState("");
  const [metadataTags, setMetadataTags] = useState("");
  const [metadataVersion, setMetadataVersion] = useState("1");
  const [lifecycleId, setLifecycleId] = useState("");
  const [tagQuery, setTagQuery] = useState("");
  const [tagLimit, setTagLimit] = useState("20");
  const [newTag, setNewTag] = useState("");
  const [memberEmail, setMemberEmail] = useState("");
  const [memberRole, setMemberRole] = useState("member");
  const [memberId, setMemberId] = useState("");
  const [memberUpdateRole, setMemberUpdateRole] = useState("member");
  const [memberStatus, setMemberStatus] = useState("active");
  const runHealth = health.run;
  const runSession = session.run;

  useEffect(() => {
    void runHealth("/api/v1/health");
    void runSession("/api/v1/session");
  }, [runHealth, runSession]);

  const listDocuments = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const params = new URLSearchParams();
    if (documentTag.trim()) params.set("tag", documentTag.trim());
    if (documentTag.trim()) params.set("match", documentMatch);
    void documents.run(`/api/v1/documents${params.size ? `?${params.toString()}` : ""}`);
  };

  const getDocument = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (lookupId.trim()) void documentLookup.run(`/api/v1/documents/${encodeURIComponent(lookupId.trim())}`);
  };

  const submitUpload = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!uploadFile) return;
    const body = new FormData();
    body.append("file", uploadFile);
    body.append("title", uploadTitle);
    body.append("tags", uploadTags);
    void upload.run("/api/v1/documents", {
      method: "POST",
      headers: { "Idempotency-Key": idempotencyKey.trim() },
      body,
    });
  };

  const patchMetadata = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!metadataId.trim()) return;
    void metadata.run(
      `/api/v1/documents/${encodeURIComponent(metadataId.trim())}`,
      jsonOptions("PATCH", {
        title: metadataTitle,
        tags: metadataTags.split(",").map((tag) => tag.trim()).filter(Boolean),
        version: Number(metadataVersion) || 0,
      }),
    );
  };

  const runLifecycle = (method: string, suffix = "") => {
    if (lifecycleId.trim()) {
      void lifecycle.run(`/api/v1/documents/${encodeURIComponent(lifecycleId.trim())}${suffix}`, { method });
    }
  };

  const listTags = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const params = new URLSearchParams({ limit: tagLimit });
    if (tagQuery.trim()) params.set("q", tagQuery.trim());
    void tags.run(`/api/v1/tags?${params.toString()}`);
  };

  const createTag = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (newTag.trim()) void tagCreate.run("/api/v1/tags", jsonOptions("POST", { name: newTag.trim() }));
  };

  const createMember = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (memberEmail.trim()) {
      void memberCreate.run("/api/v1/admin/members", jsonOptions("POST", { email: memberEmail.trim(), role: memberRole }));
    }
  };

  const updateMember = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (memberId.trim()) {
      void memberUpdate.run(
        `/api/v1/admin/members/${encodeURIComponent(memberId.trim())}`,
        jsonOptions("PATCH", { role: memberUpdateRole, status: memberStatus }),
      );
    }
  };

  return (
    <main className="workbench">
      <header className="topbar">
        <a className="wordmark" href="#top" aria-label="Aether API bench home">
          <span className="wordmark__mark">A</span>
          <span>AETHER</span>
        </a>
        <div className="topbar__context"><span className="signal-dot" /> LOCAL BACKEND / API BENCH</div>
        <a className="topbar__link" href="/api/v1/health" target="_blank" rel="noreferrer">
          Raw health <ArrowUpRight aria-hidden="true" />
        </a>
      </header>

      <div className="layout" id="top">
        <aside className="rail">
          <div className="rail__intro">
            <span className="rail__eyebrow">Builder surface</span>
            <h1>See the system<br /><em>respond.</em></h1>
            <p>Small, direct probes for the backend foundations already in place.</p>
          </div>
          <nav aria-label="API bench sections">
            <a href="#connectivity"><Activity aria-hidden="true" /> Connectivity</a>
            <a href="#documents"><FileText aria-hidden="true" /> Documents</a>
            <a href="#tags"><Tags aria-hidden="true" /> Tags</a>
            <a href="#members"><UserRound aria-hidden="true" /> Admin members</a>
          </nav>
          <div className="rail__note"><span>BASE</span><code>/api/v1</code><small>Vite proxy → 127.0.0.1:8080</small></div>
        </aside>

        <div className="content">
          <div className="content__heading">
            <div>
              <p className="section-label">AETHER / DEVELOPMENT TOOL</p>
              <h2>API bench</h2>
            </div>
            <p className="content__hint">Responses stay visible here so a failed request is useful too.</p>
          </div>

          <section className="section" aria-labelledby="connectivity-title">
            <SectionHeading icon={Activity} title="Connectivity" description="Confirm the server and current browser session before probing protected routes." anchor="connectivity" />
            <div className="card-grid card-grid--two">
              <Card>
                <CardHeader>
                  <div className="card-title-row"><CardTitle id="connectivity-title">Health</CardTitle><span className={`status-chip ${health.result?.ok ? "status-chip--ok" : ""}`}><span />{health.result ? (health.result.ok ? "online" : "attention") : "checking"}</span></div>
                  <CardDescription><code>GET /api/v1/health</code></CardDescription>
                </CardHeader>
                <CardContent><ResultBlock result={health.result} /></CardContent>
                <CardFooter><ActionButton variant="outline" loading={health.loading} onClick={() => void health.run("/api/v1/health")}><RefreshCw aria-hidden="true" /> Refresh health</ActionButton></CardFooter>
              </Card>

              <Card>
                <CardHeader>
                  <div className="card-title-row"><CardTitle>Session</CardTitle><span className={`status-chip ${session.result?.ok ? "status-chip--ok" : ""}`}><span />{session.result?.ok ? "authenticated" : "protected"}</span></div>
                  <CardDescription><code>GET /api/v1/session</code> · browser cookies included</CardDescription>
                </CardHeader>
                <CardContent><ResultBlock result={session.result} /></CardContent>
                <CardFooter className="button-row">
                  <ActionButton variant="outline" loading={session.loading} onClick={() => void session.run("/api/v1/session")}><RefreshCw aria-hidden="true" /> Check session</ActionButton>
                  <Button asChild variant="ghost"><a href="/auth/google/start"><ShieldCheck aria-hidden="true" /> Google login</a></Button>
                  <Button variant="ghost" onClick={async () => { await session.run("/api/v1/logout", { method: "POST" }); await session.run("/api/v1/session"); }}><LogOut aria-hidden="true" /> Logout</Button>
                </CardFooter>
              </Card>
            </div>
          </section>

          <section className="section" aria-labelledby="documents-heading">
            <SectionHeading icon={FileText} title="Documents" description="Exercise listing, retrieval, upload, metadata concurrency, and lifecycle operations." anchor="documents" />
            <div className="card-grid card-grid--two">
              <Card>
                <CardHeader><CardTitle id="documents-heading">List documents</CardTitle><CardDescription><code>GET /api/v1/documents</code> · ready documents by default</CardDescription></CardHeader>
                <CardContent>
                  <form className="form-stack" onSubmit={listDocuments}>
                    <div className="form-row">
                      <Field id="document-tag" label="Tag filter"><Input id="document-tag" value={documentTag} onChange={(event) => setDocumentTag(event.target.value)} placeholder="e.g. insurance" /></Field>
                      <Field id="document-match" label="Match"><select id="document-match" className="ui-select" value={documentMatch} onChange={(event) => setDocumentMatch(event.target.value)}><option value="all">All tags</option><option value="any">Any tag</option></select></Field>
                    </div>
                    <ActionButton loading={documents.loading}>Run list request</ActionButton>
                  </form>
                  <ResultBlock result={documents.result} />
                </CardContent>
              </Card>

              <Card>
                <CardHeader><CardTitle>Get document</CardTitle><CardDescription><code>GET /api/v1/documents/:id</code></CardDescription></CardHeader>
                <CardContent>
                  <form className="form-stack" onSubmit={getDocument}>
                    <Field id="lookup-id" label="Document ID"><Input id="lookup-id" value={lookupId} onChange={(event) => setLookupId(event.target.value)} placeholder="doc_..." /></Field>
                    <ActionButton loading={documentLookup.loading}>Fetch document</ActionButton>
                  </form>
                  <ResultBlock result={documentLookup.result} />
                </CardContent>
              </Card>

              <Card>
                <CardHeader><CardTitle>Upload document</CardTitle><CardDescription><code>POST /api/v1/documents</code> · multipart upload</CardDescription></CardHeader>
                <CardContent>
                  <form className="form-stack" onSubmit={submitUpload}>
                    <Field id="upload-file" label="Original file" hint="PDF, JPEG, PNG, or WebP · max 50 MB"><Input id="upload-file" type="file" accept="application/pdf,image/jpeg,image/png,image/webp" onChange={(event) => setUploadFile(event.target.files?.[0] ?? null)} /></Field>
                    <div className="form-row"><Field id="upload-title" label="Title"><Input id="upload-title" value={uploadTitle} onChange={(event) => setUploadTitle(event.target.value)} placeholder="Optional display title" /></Field><Field id="upload-tags" label="Tags"><Input id="upload-tags" value={uploadTags} onChange={(event) => setUploadTags(event.target.value)} placeholder="comma, separated" /></Field></div>
                    <Field id="idempotency-key" label="Idempotency-Key"><Input id="idempotency-key" value={idempotencyKey} onChange={(event) => setIdempotencyKey(event.target.value)} /></Field>
                    <ActionButton loading={upload.loading}><Upload aria-hidden="true" /> Send upload</ActionButton>
                  </form>
                  <ResultBlock result={upload.result} />
                </CardContent>
              </Card>

              <Card>
                <CardHeader><CardTitle>Patch metadata</CardTitle><CardDescription><code>PATCH /api/v1/documents/:id</code> · optimistic version check</CardDescription></CardHeader>
                <CardContent>
                  <form className="form-stack" onSubmit={patchMetadata}>
                    <div className="form-row"><Field id="metadata-id" label="Document ID"><Input id="metadata-id" value={metadataId} onChange={(event) => setMetadataId(event.target.value)} placeholder="doc_..." /></Field><Field id="metadata-version" label="Expected version"><Input id="metadata-version" type="number" min="0" value={metadataVersion} onChange={(event) => setMetadataVersion(event.target.value)} /></Field></div>
                    <Field id="metadata-title" label="Title"><Input id="metadata-title" value={metadataTitle} onChange={(event) => setMetadataTitle(event.target.value)} /></Field>
                    <Field id="metadata-tags" label="Tags"><Input id="metadata-tags" value={metadataTags} onChange={(event) => setMetadataTags(event.target.value)} placeholder="comma, separated" /></Field>
                    <ActionButton loading={metadata.loading}>Patch metadata</ActionButton>
                  </form>
                  <ResultBlock result={metadata.result} />
                </CardContent>
              </Card>
            </div>

            <Card>
              <CardHeader><CardTitle>Document lifecycle</CardTitle><CardDescription>Delete, restore, or purge one document. Purge requires an administrator and an expired retention window.</CardDescription></CardHeader>
              <CardContent>
                <div className="lifecycle-row"><Field id="lifecycle-id" label="Document ID"><Input id="lifecycle-id" value={lifecycleId} onChange={(event) => setLifecycleId(event.target.value)} placeholder="doc_..." /></Field><div className="button-row button-row--bottom"><Button variant="outline" disabled={lifecycle.loading} onClick={() => runLifecycle("DELETE")}><Database aria-hidden="true" /> Soft delete</Button><Button variant="outline" disabled={lifecycle.loading} onClick={() => runLifecycle("POST", "/restore")}><Check aria-hidden="true" /> Restore</Button><Button variant="destructive" disabled={lifecycle.loading} onClick={() => runLifecycle("DELETE", "/purge")}><CircleAlert aria-hidden="true" /> Purge</Button></div></div>
                <ResultBlock result={lifecycle.result} />
              </CardContent>
            </Card>
          </section>

          <section className="section" aria-labelledby="tags-heading">
            <SectionHeading icon={Tags} title="Tags" description="Check autocomplete behavior and create reusable case-insensitive tags." anchor="tags" />
            <div className="card-grid card-grid--two">
              <Card>
                <CardHeader><CardTitle id="tags-heading">List tags</CardTitle><CardDescription><code>GET /api/v1/tags</code></CardDescription></CardHeader>
                <CardContent><form className="form-stack" onSubmit={listTags}><div className="form-row"><Field id="tag-query" label="Search"><Input id="tag-query" value={tagQuery} onChange={(event) => setTagQuery(event.target.value)} placeholder="q" /></Field><Field id="tag-limit" label="Limit"><Input id="tag-limit" type="number" min="1" max="100" value={tagLimit} onChange={(event) => setTagLimit(event.target.value)} /></Field></div><ActionButton loading={tags.loading}>List tags</ActionButton></form><ResultBlock result={tags.result} /></CardContent>
              </Card>
              <Card>
                <CardHeader><CardTitle>Create tag</CardTitle><CardDescription><code>POST /api/v1/tags</code></CardDescription></CardHeader>
                <CardContent><form className="form-stack" onSubmit={createTag}><Field id="new-tag" label="Name"><Input id="new-tag" value={newTag} onChange={(event) => setNewTag(event.target.value)} placeholder="e.g. tax" /></Field><ActionButton loading={tagCreate.loading}>Create tag</ActionButton></form><ResultBlock result={tagCreate.result} /></CardContent>
              </Card>
            </div>
          </section>

          <section className="section" aria-labelledby="members-heading">
            <SectionHeading icon={UserRound} title="Admin members" description="These routes require an authenticated administrator session." anchor="members" />
            <div className="card-grid card-grid--two">
              <Card>
                <CardHeader><CardTitle id="members-heading">Member list</CardTitle><CardDescription><code>GET /api/v1/admin/members</code></CardDescription></CardHeader>
                <CardContent><ActionButton loading={members.loading} onClick={() => void members.run("/api/v1/admin/members")}>Load members</ActionButton><ResultBlock result={members.result} /></CardContent>
              </Card>
              <Card>
                <CardHeader><CardTitle>Add member</CardTitle><CardDescription><code>POST /api/v1/admin/members</code></CardDescription></CardHeader>
                <CardContent><form className="form-stack" onSubmit={createMember}><div className="form-row"><Field id="member-email" label="Email"><Input id="member-email" type="email" value={memberEmail} onChange={(event) => setMemberEmail(event.target.value)} placeholder="person@example.com" /></Field><Field id="member-role" label="Role"><select id="member-role" className="ui-select" value={memberRole} onChange={(event) => setMemberRole(event.target.value)}><option value="member">Member</option><option value="admin">Admin</option></select></Field></div><ActionButton loading={memberCreate.loading}>Add member</ActionButton></form><ResultBlock result={memberCreate.result} /></CardContent>
              </Card>
              <Card>
                <CardHeader><CardTitle>Update member</CardTitle><CardDescription><code>PATCH /api/v1/admin/members/:id</code></CardDescription></CardHeader>
                <CardContent><form className="form-stack" onSubmit={updateMember}><Field id="member-id" label="Member ID"><Input id="member-id" value={memberId} onChange={(event) => setMemberId(event.target.value)} placeholder="member_..." /></Field><div className="form-row"><Field id="member-update-role" label="Role"><select id="member-update-role" className="ui-select" value={memberUpdateRole} onChange={(event) => setMemberUpdateRole(event.target.value)}><option value="member">Member</option><option value="admin">Admin</option></select></Field><Field id="member-status" label="Status"><select id="member-status" className="ui-select" value={memberStatus} onChange={(event) => setMemberStatus(event.target.value)}><option value="active">Active</option><option value="disabled">Disabled</option></select></Field></div><ActionButton loading={memberUpdate.loading}>Update member</ActionButton></form><ResultBlock result={memberUpdate.result} /></CardContent>
              </Card>
            </div>
          </section>

          <footer className="footer"><span><span className="signal-dot" /> Requests use same-origin credentials.</span><span>Aether backend foundations / v1</span></footer>
        </div>
      </div>
    </main>
  );
}

export default App;
