import { CircleAlert, X } from "lucide-react";
import { useCallback, useEffect, useState } from "react";

import { api, type Member, type Session, type Tag } from "@/lib/api";
import {
  getDocumentIdFromPath,
  getErrorMessage,
  getRoute,
  navigate,
  type DocumentItem,
  type RouteName,
} from "@/lib/vault";

import { Brand } from "@/components/Brand";
import { ConnectionFailure } from "@/components/ConnectionFailure";
import { Inspector } from "@/components/Inspector";
import { LibraryWorkspace } from "@/components/LibraryWorkspace";
import { MembersPage } from "@/components/MembersPage";
import { Navigation } from "@/components/Navigation";
import { SignedOut } from "@/components/SignedOut";
import { TagsPage } from "@/components/TagsPage";
import { TopBar } from "@/components/TopBar";
import { TrashPage } from "@/components/TrashPage";
import { UploadDialog } from "@/components/UploadDialog";
import { UploadQueue } from "@/components/UploadQueue";
import { Button } from "@/components/ui/button";
import "./VaultApp.css";

/** Renders the Celestial Archive Garden vault and coordinates its API-backed flows. */
export default function VaultApp() {
  const [session, setSession] = useState<Session | null>(null);
  const [sessionState, setSessionState] = useState<
    "loading" | "ready" | "signed-out" | "disabled" | "error"
  >("loading");
  const [route, setRoute] = useState<RouteName>(() =>
    getRoute(window.location.pathname),
  );
  const [documents, setDocuments] = useState<DocumentItem[]>([]);
  const [tags, setTags] = useState<Tag[]>([]);
  const [members, setMembers] = useState<Member[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(() =>
    getDocumentIdFromPath(window.location.pathname),
  );
  const [query, setQuery] = useState("");
  const [navOpen, setNavOpen] = useState(false);
  const [uploadOpen, setUploadOpen] = useState(false);
  const [uploadFiles, setUploadFiles] = useState<File[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const closeNavigation = useCallback(() => setNavOpen(false), []);

  const loadDocuments = useCallback(async () => {
    setLoading(true);
    try {
      const result = await api.listDocuments();
      setDocuments(result.data.documents);
      setError("");
    } catch (requestError) {
      setError(getErrorMessage(requestError));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    const onPopState = () => {
      setRoute(getRoute(window.location.pathname));
      setSelectedId(getDocumentIdFromPath(window.location.pathname));
      setNavOpen(false);
    };
    window.addEventListener("popstate", onPopState);
    return () => window.removeEventListener("popstate", onPopState);
  }, []);

  const refreshMembers = useCallback(async () => {
    if (session?.member.role !== "admin") return;
    try {
      const result = await api.listMembers();
      setMembers(result.data.members);
    } catch (requestError) {
      setError(getErrorMessage(requestError));
    }
  }, [session]);

  useEffect(() => {
    void api
      .getSession()
      .then((next) => {
        setSession(next.data);
        setSessionState("ready");
      })
      .catch((requestError: unknown) => {
        const status =
          requestError &&
          typeof requestError === "object" &&
          "status" in requestError &&
          typeof requestError.status === "number"
            ? requestError.status
            : -1;
        if (status === 403) setSessionState("disabled");
        else if (status === 401) setSessionState("signed-out");
        else {
          setError(getErrorMessage(requestError));
          setSessionState("error");
        }
      });
  }, []);

  useEffect(() => {
    if (sessionState !== "ready") return;
    const refreshTags = async () => {
      try {
        const result = await api.listTags("", 100);
        setTags(result.data.tags);
      } catch (requestError) {
        setError(getErrorMessage(requestError));
      }
    };
    void refreshTags();
    void refreshMembers();
    void loadDocuments();
  }, [loadDocuments, refreshMembers, sessionState]);

  const selected =
    documents.find((item) => item.document.id === selectedId) ?? null;

  const updateMetadata = async (title: string, nextTags: string[]) => {
    if (!selected) return;
    try {
      const result = await api.updateMetadata(selected.document.id, {
        title,
        tags: nextTags,
        version: selected.document.version,
      });
      const updated: DocumentItem = result.data;
      setDocuments((current) =>
        current.map((item) =>
          item.document.id === selected.document.id ? updated : item,
        ),
      );
    } catch (requestError) {
      throw new Error(getErrorMessage(requestError));
    }
  };

  const deleteSelected = async () => {
    if (!selected) return;
    await api.deleteDocument(selected.document.id);
    setDocuments((current) =>
      current.map((item) =>
        item.document.id === selected.document.id
          ? { ...item, document: { ...item.document, status: "deleted" } }
          : item,
      ),
    );
    setSelectedId(null);
  };

  const restoreDocument = async (id: string) => {
    try {
      await api.restoreDocument(id);
      setDocuments((current) =>
        current.map((item) =>
          item.document.id === id
            ? { ...item, document: { ...item.document, status: "ready" } }
            : item,
        ),
      );
    } catch (requestError) {
      setError(getErrorMessage(requestError));
    }
  };

  const purgeDocument = async (id: string) => {
    try {
      await api.purgeDocument(id);
      setDocuments((current) =>
        current.filter((item) => item.document.id !== id),
      );
    } catch (requestError) {
      setError(getErrorMessage(requestError));
    }
  };

  const logout = async () => {
    try {
      await api.logout();
    } finally {
      setSession(null);
      setSessionState("signed-out");
    }
  };

  const completeUpload = useCallback(
    (item: DocumentItem, finished: boolean) => {
      setDocuments((current) => [item, ...current]);
      if (finished) {
        setUploadFiles([]);
        setUploadOpen(false);
      }
    },
    [],
  );

  const openDocument = (id: string) => {
    setSelectedId(id);
    navigate(`/documents/${encodeURIComponent(id)}`);
  };

  if (sessionState === "loading")
    return (
      <main className="vault-login">
        <section className="vault-login__sheet">
          <Brand />
          <p>Opening your archive…</p>
        </section>
      </main>
    );
  if (sessionState === "signed-out") return <SignedOut />;
  if (sessionState === "disabled") return <SignedOut disabled />;
  if (sessionState === "error") {
    return (
      <ConnectionFailure
        message={error || "The server could not be reached."}
      />
    );
  }
  if (!session) return <SignedOut />;

  const isWorkspace = route === "library" || route === "recent";
  return (
    <div className="vault-app">
      <TopBar
        session={session}
        onMenu={() => setNavOpen((current) => !current)}
        navOpen={navOpen}
        onSearch={setQuery}
        onUpload={() => setUploadOpen(true)}
        onLogout={() => void logout()}
      />
      <div className="vault-layout">
        <Navigation
          session={session}
          documents={documents}
          tags={tags}
          open={navOpen}
          onClose={closeNavigation}
        />
        <main className="vault-main">
          {error && (
            <div className="vault-notice" role="alert">
              <CircleAlert size={14} aria-hidden="true" /> {error}
              <Button
                variant="ghost"
                size="icon"
                className="vault-button vault-button--icon"
                type="button"
                aria-label="Dismiss error"
                onClick={() => setError("")}
              >
                <X size={14} aria-hidden="true" />
              </Button>
            </div>
          )}
          {isWorkspace && (
            <LibraryWorkspace
              route={route}
              documents={documents.filter(
                (item) => item.document.status !== "deleted",
              )}
              availableTags={tags}
              selectedId={selectedId}
              onSelect={openDocument}
              query={query}
              onUpload={() => setUploadOpen(true)}
              onRefresh={() => void loadDocuments()}
              loading={loading}
            />
          )}
          {route === "tags" && (
            <TagsPage
              tags={tags}
              documents={documents}
              onOpenTag={(tag) => {
                setQuery(tag.displayName);
                navigate("/library");
              }}
              onCreateTag={async (name) => {
                const created = await api.createTag(name);
                setTags((current) => [...current, created.data]);
              }}
            />
          )}
          {route === "trash" && (
            <TrashPage
              documents={documents}
              onRestore={restoreDocument}
              onPurge={purgeDocument}
            />
          )}
          {route === "members" && session.member.role === "admin" && (
            <MembersPage
              members={members}
              onAdd={async (email, role) => {
                const created = await api.createMember(email, role);
                setMembers((current) => [...current, created.data]);
              }}
              onUpdate={async (member) => {
                const updated = await api.updateMember(
                  member.id,
                  member.role,
                  member.status,
                );
                setMembers((current) =>
                  current.map((entry) =>
                    entry.id === updated.data.id ? updated.data : entry,
                  ),
                );
              }}
            />
          )}
        </main>
      </div>
      {selected && isWorkspace && (
        <Inspector
          key={selected.document.id}
          item={selected}
          onClose={() => {
            setSelectedId(null);
            navigate("/library");
          }}
          onUpdate={updateMetadata}
          onDelete={deleteSelected}
          onDownload={() => {
            window.location.href = api.getDocumentContentUrl(
              selected.document.id,
              { download: true },
            );
          }}
        />
      )}
      {uploadOpen && uploadFiles.length === 0 && (
        <UploadDialog
          onClose={() => setUploadOpen(false)}
          onUpload={setUploadFiles}
        />
      )}
      {uploadFiles.length > 0 && (
        <UploadQueue
          files={uploadFiles}
          onClose={() => {
            setUploadFiles([]);
            setUploadOpen(false);
          }}
          onComplete={completeUpload}
        />
      )}
    </div>
  );
}
