'use client';

import { useCallback, useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import {
  CommandPalette,
  type CommandPaletteResult,
} from '../../../shared/components/command-palette';
import { aiApi } from '@/lib/api';
import { STUDENT_ROUTE_CATALOG } from '@/lib/route-catalog';

function fallbackRouteResults(query: string): CommandPaletteResult[] {
  const q = query.trim().toLowerCase();
  if (!q) {
    return STUDENT_ROUTE_CATALOG.slice(0, 6).map((r) => ({
      id: r.path,
      label: r.label,
      route: r.path,
      group: r.group,
    }));
  }
  return STUDENT_ROUTE_CATALOG.filter((r) => r.label.toLowerCase().includes(q)).map((r) => ({
    id: r.path,
    label: r.label,
    route: r.path,
    group: r.group,
  }));
}

export function CommandPaletteHost() {
  const router = useRouter();
  const [open, setOpen] = useState(false);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        setOpen((v) => !v);
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, []);

  const handleSearch = useCallback(async (query: string): Promise<CommandPaletteResult[]> => {
    try {
      const { results } = await aiApi.queryCmdPalette({
        query,
        userContext: { role: 'student', courseIds: [] },
        routeCatalog: STUDENT_ROUTE_CATALOG.map((r) => ({
          path: r.path,
          label: r.label,
          group: r.group,
        })),
      });
      if (results && results.length > 0) {
        return results.map(
          (r: {
            id: string;
            label: string;
            description?: string;
            route?: string;
            group?: string;
          }) => ({
            id: r.id,
            label: r.label,
            description: r.description,
            route: r.route,
            group: r.group,
          })
        );
      }
      return fallbackRouteResults(query);
    } catch {
      return fallbackRouteResults(query);
    }
  }, []);

  return (
    <CommandPalette
      open={open}
      onOpenChange={setOpen}
      onSearch={handleSearch}
      routerPush={(path) => router.push(path)}
    />
  );
}
