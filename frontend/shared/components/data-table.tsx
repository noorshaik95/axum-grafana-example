/**
 * Slate Glass Design System - DataTable Component
 *
 * A standardized data table component for displaying tabular data consistently
 * across all portals. Uses slate-colored styling with type-safe columns and data.
 *
 * Features:
 * - Type-safe columns and data props
 * - Slate-colored header styling
 * - Alternating row colors using slate palette
 * - Subtle slate hover highlight
 * - Responsive design for mobile
 *
 * Requirements: 6.1, 6.2, 6.3, 6.4
 */

import * as React from 'react'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '../utils'

/**
 * DataTable container variant styles using class-variance-authority
 *
 * Base styles include:
 * - Glassmorphism with backdrop blur
 * - Slate-colored borders
 * - Rounded corners using design system radius
 * - Overflow handling for responsive design
 *
 * Requirements: 6.1
 */
export const dataTableVariants = cva(
  [
    'rounded-xl',
    'bg-card/70 dark:bg-card/50',
    'backdrop-blur-[12px]',
    'border border-card-border',
    'text-card-foreground',
    'shadow-lg',
    'overflow-hidden',
  ],
  {
    variants: {
      /**
       * Size variants for different table sizes
       */
      size: {
        default: '',
        sm: '',
        lg: '',
        full: 'w-full',
      },
    },
    defaultVariants: {
      size: 'default',
    },
  }
)

/**
 * Table header variant styles
 *
 * Uses slate colors for consistent header styling
 *
 * Requirements: 6.2
 */
export const tableHeaderVariants = cva([
  'bg-muted/50 dark:bg-slate-800/50',
  'border-b border-border',
])

/**
 * Table header cell variant styles
 *
 * Requirements: 6.2
 */
export const tableHeaderCellVariants = cva([
  'px-4 py-3',
  'text-left',
  'text-sm font-semibold',
  'text-foreground',
  'whitespace-nowrap',
])

/**
 * Table row variant styles
 *
 * Includes alternating row colors and hover highlight
 *
 * Requirements: 6.3, 6.4
 */
export const tableRowVariants = cva(
  ['border-b border-border/50', 'transition-colors duration-150'],
  {
    variants: {
      /**
       * Alternating row colors using slate palette
       * Requirements: 6.3
       */
      striped: {
        even: 'bg-transparent',
        odd: 'bg-muted/30 dark:bg-slate-800/30',
      },
      /**
       * Hover highlight variant
       * Requirements: 6.4
       */
      hover: {
        default: 'hover:bg-muted/50 dark:hover:bg-slate-700/50',
        none: '',
      },
    },
    defaultVariants: {
      striped: 'even',
      hover: 'default',
    },
  }
)

/**
 * Table cell variant styles
 */
export const tableCellVariants = cva(['px-4 py-3', 'text-sm', 'text-foreground'])

/**
 * Column definition interface
 *
 * Provides type-safe column configuration for the DataTable.
 *
 * Requirements: 6.1
 *
 * @template T - The type of data objects in the table
 */
export interface Column<T> {
  /**
   * The key of the data object to display in this column
   */
  key: keyof T

  /**
   * The header text to display for this column
   */
  header: string

  /**
   * Optional custom render function for cell content
   * @param value - The value at the column key
   * @param row - The full row data object
   * @returns React node to render in the cell
   */
  render?: (value: T[keyof T], row: T) => React.ReactNode

  /**
   * Optional CSS class name for the column cells
   */
  className?: string

  /**
   * Optional CSS class name for the header cell
   */
  headerClassName?: string

  /**
   * Optional alignment for the column
   * @default "left"
   */
  align?: 'left' | 'center' | 'right'
}

/**
 * DataTable component props
 *
 * Requirements: 6.1 - DataTable SHALL accept columns and data props with type safety
 *
 * @template T - The type of data objects in the table
 */
export interface DataTableProps<T>
  extends
    Omit<React.HTMLAttributes<HTMLDivElement>, 'children'>,
    VariantProps<typeof dataTableVariants> {
  /**
   * Column definitions for the table
   * Each column specifies the key, header, and optional render function
   */
  columns: Column<T>[]

  /**
   * Array of data objects to display in the table
   */
  data: T[]

  /**
   * Optional key extractor function for row keys
   * @default Uses index as key
   */
  getRowKey?: (row: T, index: number) => string | number

  /**
   * Whether to show alternating row colors
   * @default true
   */
  striped?: boolean

  /**
   * Whether to show hover highlight on rows
   * @default true
   */
  hoverable?: boolean

  /**
   * Optional empty state content when data is empty
   */
  emptyState?: React.ReactNode

  /**
   * Optional loading state
   */
  loading?: boolean

  /**
   * Optional loading content
   */
  loadingContent?: React.ReactNode
}

/**
 * Get alignment class based on column alignment
 */
function getAlignmentClass(align?: 'left' | 'center' | 'right'): string {
  switch (align) {
    case 'center':
      return 'text-center'
    case 'right':
      return 'text-right'
    default:
      return 'text-left'
  }
}

/**
 * DataTable Component
 *
 * A type-safe data table component with slate-themed styling for displaying
 * tabular data consistently across all portals.
 *
 * Features:
 * - Type-safe columns and data props (Requirement 6.1)
 * - Slate-colored header styling (Requirement 6.2)
 * - Alternating row colors using slate palette (Requirement 6.3)
 * - Subtle slate hover highlight (Requirement 6.4)
 *
 * @example
 * ```tsx
 * interface User {
 *   id: number;
 *   name: string;
 *   email: string;
 *   role: string;
 * }
 *
 * const columns: Column<User>[] = [
 *   { key: 'name', header: 'Name' },
 *   { key: 'email', header: 'Email' },
 *   { key: 'role', header: 'Role', render: (value) => <Badge>{value}</Badge> },
 * ];
 *
 * const users: User[] = [
 *   { id: 1, name: 'John Doe', email: 'john@example.com', role: 'Admin' },
 *   { id: 2, name: 'Jane Smith', email: 'jane@example.com', role: 'User' },
 * ];
 *
 * <DataTable
 *   columns={columns}
 *   data={users}
 *   getRowKey={(row) => row.id}
 * />
 * ```
 */
function DataTableInner<T>(
  {
    className,
    columns,
    data,
    size,
    getRowKey,
    striped = true,
    hoverable = true,
    emptyState,
    loading = false,
    loadingContent,
    ...props
  }: DataTableProps<T>,
  ref: React.ForwardedRef<HTMLDivElement>
) {
  // Default empty state
  const defaultEmptyState = (
    <div className="px-4 py-8 text-center text-muted-foreground">No data available</div>
  )

  // Default loading content
  const defaultLoadingContent = (
    <div className="px-4 py-8 text-center text-muted-foreground">Loading...</div>
  )

  return (
    <div ref={ref} className={cn(dataTableVariants({ size, className }))} {...props}>
      <div className="overflow-x-auto">
        <table className="w-full">
          {/* Table Header - Requirement 6.2 */}
          <thead className={cn(tableHeaderVariants())}>
            <tr>
              {columns.map((column) => (
                <th
                  key={String(column.key)}
                  className={cn(
                    tableHeaderCellVariants(),
                    getAlignmentClass(column.align),
                    column.headerClassName
                  )}
                >
                  {column.header}
                </th>
              ))}
            </tr>
          </thead>

          {/* Table Body */}
          <tbody>
            {loading ? (
              <tr>
                <td colSpan={columns.length}>{loadingContent || defaultLoadingContent}</td>
              </tr>
            ) : data.length === 0 ? (
              <tr>
                <td colSpan={columns.length}>{emptyState || defaultEmptyState}</td>
              </tr>
            ) : (
              data.map((row, rowIndex) => {
                const rowKey = getRowKey ? getRowKey(row, rowIndex) : rowIndex

                return (
                  <tr
                    key={rowKey}
                    className={cn(
                      tableRowVariants({
                        striped: striped ? (rowIndex % 2 === 0 ? 'even' : 'odd') : 'even',
                        hover: hoverable ? 'default' : 'none',
                      })
                    )}
                  >
                    {columns.map((column) => {
                      const value = row[column.key]
                      const cellContent = column.render
                        ? column.render(value, row)
                        : String(value ?? '')

                      return (
                        <td
                          key={String(column.key)}
                          className={cn(
                            tableCellVariants(),
                            getAlignmentClass(column.align),
                            column.className
                          )}
                        >
                          {cellContent}
                        </td>
                      )
                    })}
                  </tr>
                )
              })
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}

/**
 * DataTable with forwardRef support
 *
 * Note: We use a wrapper pattern to support generics with forwardRef
 */
export const DataTable = React.forwardRef(DataTableInner) as <T>(
  props: DataTableProps<T> & { ref?: React.ForwardedRef<HTMLDivElement> }
) => React.ReactElement

/**
 * Type exports for external use
 */
export type DataTableSize = NonNullable<VariantProps<typeof dataTableVariants>['size']>
