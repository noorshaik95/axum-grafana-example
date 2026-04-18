'use client';

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Clock, Calendar } from 'lucide-react';

interface Prof {
  id: string;
  name: string;
  initials: string;
  course: string;
  slots: { time: string; available: boolean }[];
}

const MOCK_PROFS: Prof[] = [
  {
    id: 'p1',
    name: 'Prof. Martinez',
    initials: 'PM',
    course: 'CS 101',
    slots: [
      { time: 'Today 2:00pm', available: true },
      { time: 'Today 2:15pm', available: false },
      { time: 'Today 2:30pm', available: true },
      { time: 'Fri 10:00am', available: true },
    ],
  },
  {
    id: 'p2',
    name: 'Prof. Williams',
    initials: 'PW',
    course: 'MATH 202',
    slots: [
      { time: 'Thu 3:00pm', available: true },
      { time: 'Thu 3:15pm', available: true },
      { time: 'Fri 11:00am', available: false },
    ],
  },
];

function useOfficeHours() {
  return useQuery({
    queryKey: ['office-hours-mock'],
    queryFn: async () => MOCK_PROFS,
  });
}

export default function OfficeHoursPage() {
  const { data: profs = [] } = useOfficeHours();
  const [selectedSlot, setSelectedSlot] = useState<{ profId: string; time: string } | null>(null);
  const [question, setQuestion] = useState('');
  const [booked, setBooked] = useState<string[]>([]);

  const handleBook = (profId: string) => {
    if (!selectedSlot || selectedSlot.profId !== profId) return;
    setBooked((prev) => [...prev, `${profId}-${selectedSlot.time}`]);
    setSelectedSlot(null);
    setQuestion('');
  };

  return (
    <div className="max-w-xl mx-auto space-y-5">
      <div>
        <h1 className="serif text-2xl" style={{ color: 'var(--ink)' }}>
          Office Hours
        </h1>
        <p className="text-sm mt-1" style={{ color: 'var(--muted)' }}>
          Book 15 min with a professor
        </p>
      </div>

      {profs.map((prof) => {
        const isSelected = selectedSlot?.profId === prof.id;
        return (
          <div
            key={prof.id}
            className="rounded-xl overflow-hidden"
            style={{ background: '#fff', border: '1px solid var(--border)' }}
          >
            <div
              className="flex items-center gap-4 px-5 py-4 border-b"
              style={{ borderColor: 'var(--border)' }}
            >
              <div
                className="flex items-center justify-center rounded-full font-semibold text-sm shrink-0"
                style={{
                  width: 44,
                  height: 44,
                  background: 'var(--forest-100)',
                  color: 'var(--forest-700)',
                  border: '2px solid var(--forest-200)',
                }}
              >
                {prof.initials}
              </div>
              <div>
                <p className="text-sm font-semibold" style={{ color: 'var(--ink)' }}>
                  {prof.name}
                </p>
                <p className="text-xs mt-0.5" style={{ color: 'var(--muted)' }}>
                  {prof.course}
                </p>
              </div>
            </div>

            <div className="p-5 space-y-4">
              {/* Slots */}
              <div>
                <p
                  className="text-xs font-semibold mb-2 uppercase tracking-wide"
                  style={{ color: 'var(--muted)' }}
                >
                  Available slots
                </p>
                <div className="flex flex-wrap gap-2">
                  {prof.slots.map((slot) => {
                    const isBookedSlot = booked.includes(`${prof.id}-${slot.time}`);
                    const isSel =
                      selectedSlot?.profId === prof.id && selectedSlot.time === slot.time;
                    return (
                      <button
                        key={slot.time}
                        disabled={!slot.available || isBookedSlot}
                        onClick={() => setSelectedSlot({ profId: prof.id, time: slot.time })}
                        className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-all"
                        style={{
                          background: isBookedSlot
                            ? 'var(--forest-100)'
                            : isSel
                              ? 'var(--forest-700)'
                              : slot.available
                                ? 'var(--forest-50)'
                                : '#f5f5f5',
                          color: isBookedSlot
                            ? 'var(--forest-600)'
                            : isSel
                              ? '#fff'
                              : slot.available
                                ? 'var(--forest-700)'
                                : 'var(--muted)',
                          border: `1px solid ${isSel ? 'var(--forest-700)' : slot.available ? 'var(--forest-200)' : 'var(--border)'}`,
                          cursor: slot.available && !isBookedSlot ? 'pointer' : 'not-allowed',
                          opacity: !slot.available ? 0.5 : 1,
                        }}
                      >
                        <Clock style={{ width: 11, height: 11 }} />
                        {isBookedSlot ? `${slot.time} ✓` : slot.time}
                      </button>
                    );
                  })}
                </div>
              </div>

              {/* Pre-fill question */}
              {isSelected && (
                <div className="space-y-3 pt-2 border-t" style={{ borderColor: 'var(--border)' }}>
                  <p
                    className="text-xs font-semibold uppercase tracking-wide"
                    style={{ color: 'var(--muted)' }}
                  >
                    What would you like to discuss?
                  </p>
                  <textarea
                    value={question}
                    onChange={(e) => setQuestion(e.target.value)}
                    placeholder="e.g. Stuck on PS4 problem 3, recursion base case..."
                    rows={3}
                    className="w-full rounded-lg p-3 text-sm resize-none focus:outline-none"
                    style={{
                      border: '1px solid var(--border)',
                      background: 'var(--paper)',
                      color: 'var(--ink)',
                    }}
                  />
                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => handleBook(prof.id)}
                      className="btn-primary"
                      style={{ fontSize: 13 }}
                    >
                      <Calendar
                        style={{ width: 13, height: 13, display: 'inline', marginRight: 5 }}
                      />
                      Book {selectedSlot?.time}
                    </button>
                    <button
                      onClick={() => setSelectedSlot(null)}
                      className="text-sm"
                      style={{ color: 'var(--muted)' }}
                    >
                      Cancel
                    </button>
                  </div>
                </div>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
}
