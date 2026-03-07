import React from 'react';
import { ChevronLeft, ChevronRight } from 'lucide-react';

interface PaginationProps {
    page: number;
    totalPages: number;
    total: number;
    pageSize: number;
    onPageChange: (page: number) => void;
    onPageSizeChange?: (size: number) => void;
}

/**
 * Reusable pagination component with page navigation and optional page size selector.
 */
export const Pagination: React.FC<PaginationProps> = ({
    page,
    totalPages,
    total,
    pageSize,
    onPageChange,
    onPageSizeChange,
}) => {
    const startItem = Math.min((page - 1) * pageSize + 1, total);
    const endItem = Math.min(page * pageSize, total);

    // Generate visible page numbers (window of 5 around current)
    const getVisiblePages = (): number[] => {
        const pages: number[] = [];
        let start = Math.max(1, page - 2);
        let end = Math.min(totalPages, page + 2);

        // Always show 5 if possible
        if (end - start < 4) {
            if (start === 1) end = Math.min(5, totalPages);
            else start = Math.max(1, end - 4);
        }

        for (let i = start; i <= end; i++) pages.push(i);
        return pages;
    };

    if (totalPages <= 1 && total <= pageSize) return null;

    return (
        <div className="flex items-center justify-between py-4 px-1">
            {/* Left: item count */}
            <span className="text-sm text-gray-500 dark:text-gray-400">
                Showing <span className="font-medium text-gray-700 dark:text-gray-200">{startItem}–{endItem}</span>{' '}
                of <span className="font-medium text-gray-700 dark:text-gray-200">{total}</span>
            </span>

            {/* Center: page numbers */}
            <div className="flex items-center gap-1">
                <button
                    onClick={() => onPageChange(page - 1)}
                    disabled={page <= 1}
                    className="p-1.5 rounded-lg text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
                >
                    <ChevronLeft size={16} />
                </button>

                {getVisiblePages()[0] > 1 && (
                    <>
                        <button
                            onClick={() => onPageChange(1)}
                            className="px-2.5 py-1 rounded-lg text-sm text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
                        >
                            1
                        </button>
                        {getVisiblePages()[0] > 2 && (
                            <span className="px-1 text-gray-400">…</span>
                        )}
                    </>
                )}

                {getVisiblePages().map(p => (
                    <button
                        key={p}
                        onClick={() => onPageChange(p)}
                        className={`px-2.5 py-1 rounded-lg text-sm font-medium transition-colors ${p === page
                                ? 'bg-blue-600 text-white shadow-sm'
                                : 'text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'
                            }`}
                    >
                        {p}
                    </button>
                ))}

                {getVisiblePages().slice(-1)[0] < totalPages && (
                    <>
                        {getVisiblePages().slice(-1)[0] < totalPages - 1 && (
                            <span className="px-1 text-gray-400">…</span>
                        )}
                        <button
                            onClick={() => onPageChange(totalPages)}
                            className="px-2.5 py-1 rounded-lg text-sm text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
                        >
                            {totalPages}
                        </button>
                    </>
                )}

                <button
                    onClick={() => onPageChange(page + 1)}
                    disabled={page >= totalPages}
                    className="p-1.5 rounded-lg text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
                >
                    <ChevronRight size={16} />
                </button>
            </div>

            {/* Right: page size selector */}
            {onPageSizeChange && (
                <select
                    value={pageSize}
                    onChange={(e) => onPageSizeChange(Number(e.target.value))}
                    className="px-2 py-1 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 text-gray-600 dark:text-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500/20"
                >
                    {[25, 50, 100].map(size => (
                        <option key={size} value={size}>{size} / page</option>
                    ))}
                </select>
            )}
        </div>
    );
};
