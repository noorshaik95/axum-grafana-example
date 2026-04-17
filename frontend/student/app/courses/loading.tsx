export default function CoursesLoading() {
  return (
    <div className="space-y-6 animate-pulse">
      <div className="space-y-2">
        <div className="h-7 w-32 rounded bg-gray-200" />
        <div className="h-4 w-56 rounded bg-gray-200" />
      </div>
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {Array.from({ length: 6 }).map((_, i) => (
          <div
            key={i}
            className="rounded-xl border border-[var(--color-border)] bg-white p-5 shadow-sm"
          >
            <div className="h-5 w-16 rounded bg-gray-200 mb-3" />
            <div className="h-4 w-full rounded bg-gray-200 mb-2" />
            <div className="h-3 w-32 rounded bg-gray-200 mb-4" />
            <div className="h-8 w-24 rounded bg-gray-200" />
          </div>
        ))}
      </div>
    </div>
  );
}
