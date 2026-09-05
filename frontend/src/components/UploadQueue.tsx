import { Upload, X } from "lucide-react";
import { useEffect, useState, type CSSProperties } from "react";

import { api } from "@/lib/api";
import type { DocumentItem } from "@/lib/vault";
import { getErrorMessage } from "@/lib/vault";
import { Button } from "@/components/ui/button";

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
    let completionTimer: number | undefined;
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
              setProgress(state.percent);
              if (state.state === "processing") setStatus("Finalizing in vault…");
              else if (state.state === "uploading") setStatus("Uploading original…");
            },
          },
          `aether-${file.name}-${file.size}-${file.lastModified}`,
        );
        if (!active) return;
        setProgress(100);
        setStatus("Upload complete");
        if (index === files.length - 1) {
          completionTimer = window.setTimeout(
            () => onComplete(result.data, true),
            650,
          );
        } else {
          onComplete(result.data, false);
          setIndex((current) => current + 1);
        }
      } catch (error) {
        if (!active) return;
        setFailed(true);
        setStatus(getErrorMessage(error));
      }
    };
    void upload();
    return () => {
      active = false;
      if (completionTimer !== undefined) window.clearTimeout(completionTimer);
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
        <Button
          variant="ghost"
          size="icon"
          className="vault-upload-queue__close"
          type="button"
          aria-label="Close upload queue"
          onClick={onClose}
        >
          <X size={15} aria-hidden="true" />
        </Button>
      </div>
      <div className="vault-upload-item">
        <Upload size={15} aria-hidden="true" />
        <span className="vault-upload-item__copy">
          <strong>{file.name}</strong>
          <span>
            {progress === null || failed ? status : `${status} ${progress}%`}
          </span>
          {!failed && (
            <span
              className={`vault-upload-progress ${
                progress === null ? "is-indeterminate" : ""
              }`}
              role="progressbar"
              aria-label={`Uploading ${file.name}`}
              aria-valuemin={0}
              aria-valuemax={100}
              aria-valuenow={progress === null ? undefined : progress}
              aria-valuetext={
                progress === null ? "Uploading" : `${progress}%`
              }
            >
              <span
                className="vault-upload-progress__fill"
                style={
                  {
                    "--vault-progress": progress === null ? 0 : progress / 100,
                  } as CSSProperties
                }
              />
              <span className="vault-upload-progress__label">
                {progress === null ? "Uploading" : `${progress}%`}
              </span>
            </span>
          )}
        </span>
        {failed && (
          <Button
            variant="ghost"
            size="sm"
            className="vault-member-action"
            type="button"
            onClick={() => setAttempt((current) => current + 1)}
          >
            Retry
          </Button>
        )}
      </div>
    </div>
  );
}
