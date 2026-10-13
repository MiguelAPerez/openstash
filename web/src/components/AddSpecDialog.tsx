import { useEffect, useRef, useState, type FormEvent } from "react";
import { api } from "../lib/api";
import { href, navigate } from "../lib/route";
import { useSpecs } from "../lib/specs";

interface Props {
  open: boolean;
  prefillKey?: string;
  onClose: () => void;
  onAdded: (message: string) => void;
}

export function AddSpecDialog({ open, prefillKey, onClose, onAdded }: Props) {
  const ref = useRef<HTMLDialogElement>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const { groups } = useSpecs();

  useEffect(() => {
    const d = ref.current;
    if (!d) return;
    if (open && !d.open) {
      setError(null);
      d.showModal();
    } else if (!open && d.open) d.close();
  }, [open]);

  const submit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const form = new FormData(e.currentTarget);
    const get = (k: string) => String(form.get(k) ?? "").trim();
    setBusy(true);
    setError(null);
    try {
      const res = await api.addSpec({
        key: get("key"),
        from: get("from"),
        version: get("version") || undefined,
        endpoint: get("endpoint") || undefined,
      });
      onAdded(`Added ${res.meta.key}@${res.meta.version}: ${res.indexed} operations, ${res.schemas} schemas`);
      (e.target as HTMLFormElement).reset();
      onClose();
      // A new version of a key already stored extends its timeline: show what changed.
      const existing = groups.some((g) => g.key === res.meta.key);
      navigate(
        existing ? href.timeline(res.meta.key, { to: res.meta.version }) : href.explore(res.meta.key, res.meta.version),
      );
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <dialog className="dialog" ref={ref} onClose={onClose} aria-labelledby="add-title">
      <form className="dialog-body" onSubmit={submit}>
        <h2 id="add-title">Add a spec</h2>
        <p className="muted">
          The server fetches the URL (or reads the file path) and indexes it, same as <code>openstash add</code>. Adding
          a new version of an existing key extends its timeline.
        </p>
        <label className="field">
          <span>
            Key <em>required</em>
          </span>
          <input className="input" name="key" required placeholder="gitea" autoComplete="off" defaultValue={prefillKey} key={prefillKey} />
        </label>
        <label className="field">
          <span>
            Source <em>required — URL or server-side file path</em>
          </span>
          <input className="input" name="from" required placeholder="https://gitea.example/swagger.v1.json" autoComplete="off" />
        </label>
        <div className="field-row">
          <label className="field">
            <span>Version</span>
            <input className="input" name="version" placeholder="from info.version" autoComplete="off" />
          </label>
          <label className="field">
            <span>Endpoint</span>
            <input className="input" name="endpoint" placeholder="https://gitea.example/api/v1" autoComplete="off" />
          </label>
        </div>
        {error && (
          <p className="notice form-error" role="alert">
            {error}
          </p>
        )}
        <div className="dialog-actions">
          <button className="btn" type="button" onClick={onClose}>
            Cancel
          </button>
          <button className="btn btn-primary" type="submit" disabled={busy}>
            {busy ? "Adding…" : "Add spec"}
          </button>
        </div>
      </form>
    </dialog>
  );
}
