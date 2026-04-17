export default function GradesLoading() {
  return (
    <div className="space-y-6 animate-pulse">
      <div className="space-y-2">
        <div className="h-7 w-24 rounded bg-gray-200" />
        <div className="h-4 w-64 rounded bg-gray-200" />
      </div>
      <div className="grid gap-4 sm:grid-cols-3">
        {Array.from({ length: 3 }).map((_, i) => (
          <div
            key={i}
            className="rounded-xl border border-[var(--color-border)] bg-white p-5 shadow-sm"
          >
            <div className="h-4 w-20 rounded bg-gray-200" />
            <div className="mt-3 h-8 w-12 rounded bg-gray-200" />
          </div>
        ))}
      </div>
      <div className="grid gap-4 sm:grid-cols-2">
        {Array.from({ length: 4 }).map((_, i) => (
          <div key={i} className="h-44 rounded-xl border border-[var(--color-border)] bg-white" />
        ))}
      </div>
    </div>
  );
}
