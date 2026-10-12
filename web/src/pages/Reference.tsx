import { SpecHeader } from "../components/SpecHeader";

/** Reference tab: the Scalar viewer, isolated in an iframe (see src/reference.tsx). */
export default function Reference({ specKey, version }: { specKey: string; version: string }) {
  const src = `/reference.html?${new URLSearchParams({ key: specKey, version })}`;
  return (
    <div className="reference-page">
      <SpecHeader specKey={specKey} version={version} tab="reference" />
      <iframe className="reference-frame" title={`${specKey}@${version} API reference`} src={src} key={src} />
      <p className="muted small">
        <a href={src} target="_blank" rel="noreferrer">
          Open full screen ↗
        </a>
      </p>
    </div>
  );
}
