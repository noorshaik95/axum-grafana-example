'use client';

import { useState } from 'react';
import Link from 'next/link';
import { BookOpen, Headphones, MessageCircle, Eye } from 'lucide-react';

type LearningStyle = 'visual' | 'hands-on' | 'discussion' | 'reading';

interface LearningStyleCard {
  key: LearningStyle;
  icon: React.ElementType;
  title: string;
  description: string;
}

const LEARNING_STYLES: LearningStyleCard[] = [
  {
    key: 'visual',
    icon: Eye,
    title: 'Visual',
    description: 'Diagrams, videos, and visual explanations help me learn best.',
  },
  {
    key: 'hands-on',
    icon: Headphones,
    title: 'Hands-on',
    description: 'I learn through practice, labs, and building projects.',
  },
  {
    key: 'discussion',
    icon: MessageCircle,
    title: 'Discussion',
    description: 'Talking through concepts with peers and professors clarifies ideas.',
  },
  {
    key: 'reading',
    icon: BookOpen,
    title: 'Reading',
    description: 'Textbooks and written materials are my primary learning source.',
  },
];

const STEPS = ['Welcome', 'Learning style', 'Courses', 'Done'];

export default function OnboardingPage() {
  const [step, setStep] = useState(1);
  const [selectedStyles, setSelectedStyles] = useState<Set<LearningStyle>>(new Set());
  const [name, setName] = useState('');

  const toggleStyle = (key: LearningStyle) => {
    setSelectedStyles((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  };

  return (
    <div
      className="w-full max-w-md rounded-2xl p-8"
      style={{
        background: '#fff',
        border: '1px solid var(--border)',
        boxShadow: 'var(--shadow-md)',
      }}
    >
      {/* Step indicator */}
      <div className="flex items-center gap-2 mb-8">
        {STEPS.map((s, i) => {
          const idx = i + 1;
          const isDone = idx < step;
          const isCurrent = idx === step;
          return (
            <div key={s} className="flex items-center gap-2">
              <div
                className="flex items-center justify-center rounded-full text-xs font-semibold"
                style={{
                  width: 24,
                  height: 24,
                  background: isDone
                    ? 'var(--forest-500)'
                    : isCurrent
                      ? 'var(--forest-700)'
                      : 'var(--paper)',
                  color: isDone || isCurrent ? '#fff' : 'var(--muted)',
                  border: `1.5px solid ${isDone || isCurrent ? 'transparent' : 'var(--border)'}`,
                }}
              >
                {isDone ? '✓' : idx}
              </div>
              {i < STEPS.length - 1 && (
                <div
                  style={{
                    height: 1.5,
                    width: 24,
                    background: isDone ? 'var(--forest-400)' : 'var(--border)',
                  }}
                />
              )}
            </div>
          );
        })}
        <span className="ml-2 text-xs" style={{ color: 'var(--muted)' }}>
          {step}/{STEPS.length}
        </span>
      </div>

      {/* Step 1: Welcome */}
      {step === 1 && (
        <div className="space-y-5">
          <div>
            <h1 className="serif text-2xl" style={{ color: 'var(--ink)' }}>
              Welcome to Slate
            </h1>
            <p className="text-sm mt-2" style={{ color: 'var(--muted)' }}>
              Let&apos;s get you set up. This takes about 2 minutes.
            </p>
          </div>
          <div>
            <label className="text-sm font-medium block mb-1.5" style={{ color: 'var(--ink)' }}>
              What should we call you?
            </label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Your first name"
              className="w-full rounded-lg px-4 py-2.5 text-sm focus:outline-none"
              style={{
                border: '1px solid var(--border)',
                background: 'var(--paper)',
                color: 'var(--ink)',
              }}
            />
          </div>
          <button
            onClick={() => setStep(2)}
            disabled={!name.trim()}
            className="btn-primary w-full disabled:opacity-40"
          >
            Continue →
          </button>
        </div>
      )}

      {/* Step 2: Learning style */}
      {step === 2 && (
        <div className="space-y-5">
          <div>
            <p className="mono text-xs font-semibold mb-1" style={{ color: 'var(--forest-600)' }}>
              Step 2 of 4
            </p>
            <h2 className="serif text-2xl" style={{ color: 'var(--ink)' }}>
              How do you learn best?
            </h2>
            <p className="text-sm mt-1" style={{ color: 'var(--muted)' }}>
              Select all that apply — we&apos;ll personalize your experience.
            </p>
          </div>
          <div className="grid grid-cols-2 gap-3">
            {LEARNING_STYLES.map((ls) => {
              const Icon = ls.icon;
              const isSelected = selectedStyles.has(ls.key);
              return (
                <button
                  key={ls.key}
                  onClick={() => toggleStyle(ls.key)}
                  className="flex flex-col items-start gap-3 p-4 rounded-xl text-left transition-all"
                  style={{
                    background: isSelected ? 'var(--forest-50)' : 'var(--paper)',
                    border: `2px solid ${isSelected ? 'var(--forest-500)' : 'var(--border)'}`,
                  }}
                >
                  <div
                    className="flex items-center justify-center rounded-lg"
                    style={{
                      width: 36,
                      height: 36,
                      background: isSelected ? 'var(--forest-700)' : '#fff',
                      border: '1px solid var(--border)',
                    }}
                  >
                    <Icon
                      style={{
                        width: 16,
                        height: 16,
                        color: isSelected ? '#fff' : 'var(--muted)',
                      }}
                    />
                  </div>
                  <div>
                    <p
                      className="text-sm font-semibold"
                      style={{ color: isSelected ? 'var(--forest-800)' : 'var(--ink)' }}
                    >
                      {ls.title}
                    </p>
                    <p className="text-xs mt-0.5" style={{ color: 'var(--muted)' }}>
                      {ls.description}
                    </p>
                  </div>
                </button>
              );
            })}
          </div>
          <div className="flex gap-3">
            <button
              onClick={() => setStep(1)}
              className="btn-secondary flex-1"
              style={{ fontSize: 14 }}
            >
              Back
            </button>
            <button
              onClick={() => setStep(3)}
              className="btn-primary flex-1"
              style={{ fontSize: 14 }}
            >
              Continue →
            </button>
          </div>
        </div>
      )}

      {/* Step 3: Courses */}
      {step === 3 && (
        <div className="space-y-5">
          <div>
            <p className="mono text-xs font-semibold mb-1" style={{ color: 'var(--forest-600)' }}>
              Step 3 of 4
            </p>
            <h2 className="serif text-2xl" style={{ color: 'var(--ink)' }}>
              Explore your courses
            </h2>
            <p className="text-sm mt-1" style={{ color: 'var(--muted)' }}>
              You&apos;re enrolled in these courses for this semester.
            </p>
          </div>
          <div className="space-y-2">
            {[
              { code: 'CS 101', title: 'Intro to Computer Science', prof: 'Prof. Martinez' },
              { code: 'CS 201', title: 'Data Structures', prof: 'Prof. Lee' },
              { code: 'MATH 202', title: 'Calculus II', prof: 'Prof. Williams' },
            ].map((c) => (
              <div
                key={c.code}
                className="flex items-center gap-3 p-3 rounded-xl"
                style={{ background: 'var(--forest-50)', border: '1px solid var(--forest-200)' }}
              >
                <span
                  className="text-xs font-bold px-2 py-0.5 rounded"
                  style={{ background: 'var(--forest-700)', color: '#fff' }}
                >
                  {c.code}
                </span>
                <div className="flex-1">
                  <p className="text-sm font-medium" style={{ color: 'var(--ink)' }}>
                    {c.title}
                  </p>
                  <p className="text-xs" style={{ color: 'var(--muted)' }}>
                    {c.prof}
                  </p>
                </div>
              </div>
            ))}
          </div>
          <div className="flex gap-3">
            <button
              onClick={() => setStep(2)}
              className="btn-secondary flex-1"
              style={{ fontSize: 14 }}
            >
              Back
            </button>
            <button
              onClick={() => setStep(4)}
              className="btn-primary flex-1"
              style={{ fontSize: 14 }}
            >
              Continue →
            </button>
          </div>
        </div>
      )}

      {/* Step 4: Done */}
      {step === 4 && (
        <div className="space-y-5 text-center">
          <div
            className="flex items-center justify-center rounded-full mx-auto"
            style={{
              width: 64,
              height: 64,
              background: 'var(--forest-100)',
              border: '3px solid var(--forest-300)',
            }}
          >
            <span style={{ fontSize: 28 }}>🌱</span>
          </div>
          <div>
            <h2 className="serif text-2xl" style={{ color: 'var(--ink)' }}>
              You&apos;re all set, {name}!
            </h2>
            <p className="text-sm mt-2" style={{ color: 'var(--muted)' }}>
              Your Slate portal is ready. Start with today&apos;s tasks.
            </p>
          </div>
          <Link href="/today" className="btn-primary block text-center w-full">
            Go to Today →
          </Link>
        </div>
      )}
    </div>
  );
}
