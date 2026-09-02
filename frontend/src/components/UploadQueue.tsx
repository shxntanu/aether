import { Upload, X } from "lucide-react";
import { useEffect, useState } from "react";

import { api } from "@/lib/api";
import type { DocumentItem } from "@/lib/vault";
import { getErrorMessage } from "@/lib/vault";

/** Uploads selected files sequentially and reports completed catalog records. */
export function UploadQueue({
  files,
  onClose,
  onComplete,
}: {
  files: File[];
  onClose: () => void;
  onComplete: (item: DocumentItem, finished: boolean) => void;
}) {
  const [index, setIndex] = useState(0);
  const [progress, setProgress] = useState<number | null>(null);
  const [status, setStatus] = useState("Uploading original…");
  const [failed, setFailed] = useState(false);
  const [attempt, setAttempt] = useState(0);
  const file = files[index];
  useEffect(() => {
    if (!file) return;
    let active = true;
    const upload = async () => {
      setFailed(false);
      setProgress(null);
      setStatus("Uploading original…");
      try {
        const result = await api.uploadDocument(
          file,
          {
            onProgress: (state) => {
              if (!active) return;
              if (state.state === "complete") setProgress(100);
              else if (state.percent !== null) setProgress(state.percent);
            },
          },
          `aether-${file.name}-${file.size}-${file.lastModified}`,
        );
        if (!active) return;
        setProgress(100);
        setStatus("Upload complete");
        onComplete(result.data, index === files.length - 1);
        if (index < files.length - 1)
          setIndex((current) => current + 1);
      } catch (error) {
        if (!active) return;
        setFailed(true);
        setStatus(getErrorMessage(error));
      }
    };
    void upload();
    return () => {
      active = false;
    };
  }, [attempt, file, files.length, index, onComplete]);
  if (!file) return null;
  return (
    <div className="vault-upload-queue" role="status" aria-live="polite">
      <div className="vault-upload-queue__head">
        <strong>Upload queue</strong>
        <span>
          {failed ? `${index + 1} failed` : `${index + 1} of ${files.length}`}
        </span>
        <button
          className="vault-upload-queue__close"
          type="button"
          aria-label="Close upload queue"
          onClick={onClose}
        >
          <X size={15} aria-hidden="true" />
        </button>
      </div>
      <div className="vault-upload-item">
        <Upload size={15} aria-hidden="true" />
        <span className="vault-upload-item__copy">
          <strong>{file.name}</strong>
          <span>
            {progress === null || failed ? status : `${status} ${progress}%`}
          </span>
          <span className="vault-upload-progress">
            <span
              style={{ width: progress === null ? "38%" : `${progress}%` }}
            />
          </span>
        </span>
        {failed && (
          <button
            className="vault-member-action"
            type="button"
            onClick={() => setAttempt((current) => current + 1)}
          >
            Retry
          </button>
        )}
      </div>
    </div>
  );
}
