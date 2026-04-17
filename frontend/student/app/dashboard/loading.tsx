export default function DashboardLoading() {
  return (
    <div className="space-y-6 animate-pulse">
      <div className="space-y-2">
        <div className="h-7 w-64 rounded bg-gray-200" />
        <div className="h-4 w-80 rounded bg-gray-200" />
      </div>
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <div
            key={i}
            className="rounded-xl border border-[var(--color-border)] bg-white p-5 shadow-sm"
          >
            <div className="h-4 w-24 rounded bg-gray-200" />
            <div className="mt-3 h-8 w-16 rounded bg-gray-200" />
          </div>
        ))}
      </div>
      <div className="grid gap-6 lg:grid-cols-2">
        {Array.from({ length: 4 }).map((_, i) => (
          <div
            key={i}
            className="rounded-xl border border-[var(--color-border)] bg-white p-5 shadow-sm"
          >
            <div className="h-5 w-32 rounded bg-gray-200 mb-4" />
            <div className="space-y-3">
              {Array.from({ length: 3 }).map((__, j) => (
                <div key={j} className="h-14 rounded-lg bg-gray-100" />
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
