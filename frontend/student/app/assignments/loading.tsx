export default function AssignmentsLoading() {
  return (
    <div className="space-y-6 animate-pulse">
      <div className="space-y-2">
        <div className="h-7 w-48 rounded bg-gray-200" />
        <div className="h-4 w-72 rounded bg-gray-200" />
      </div>
      <div className="rounded-xl border border-[var(--color-border)] bg-white shadow-sm">
        {Array.from({ length: 5 }).map((_, i) => (
          <div
            key={i}
            className="flex items-center justify-between p-5 border-b border-[var(--color-border)] last:border-0"
          >
            <div className="space-y-2">
              <div className="h-4 w-48 rounded bg-gray-200" />
              <div className="h-3 w-32 rounded bg-gray-200" />
            </div>
            <div className="h-6 w-20 rounded-full bg-gray-200" />
          </div>
        ))}
      </div>
    </div>
  );
}
