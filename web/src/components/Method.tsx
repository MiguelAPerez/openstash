export function Method({ method }: { method: string }) {
  const m = method.toUpperCase();
  return <span className={`method method-${m}`}>{m}</span>;
}
