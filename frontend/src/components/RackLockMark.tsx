import React from 'react';

/**
 * OS-Baka "Rack Lock" brand mark.
 *
 * A hard-cornered server chassis (bare-metal rack) whose interior negatives form a
 * top rack rail + two drive bays, with a fused keyhole as the focal point carrying
 * the full-disk-encryption (LUKS) story.
 *
 * Single path, fill="currentColor", fill-rule="evenodd" — every interior detail is a
 * true knockout, so it works on any surface. Color is inherited from the parent
 * (use `text-*` classes or `style={{ color }}`).
 *
 * Use the default mark at >=32px (sidebar header, login, docs); pass `variant="favicon"`
 * for <=24px contexts (fewer elements, bolder mass, single keyhole).
 */
interface RackLockMarkProps extends React.SVGProps<SVGSVGElement> {
  variant?: 'mark' | 'favicon';
}

const MARK_PATH =
  'M22 16 L78 16 A6 6 0 0 1 84 22 L84 78 A6 6 0 0 1 78 84 L22 84 A6 6 0 0 1 16 78 L16 22 A6 6 0 0 1 22 16 Z M30 24.5 A3.3 3.3 0 0 0 30 31.1 L70 31.1 A3.3 3.3 0 0 0 70 24.5 Z M30 36.5 A3.3 3.3 0 0 0 30 43.1 L45 43.1 A3.3 3.3 0 0 0 45 36.5 Z M55 36.5 A3.3 3.3 0 0 0 55 43.1 L70 43.1 A3.3 3.3 0 0 0 70 36.5 Z M50 50 A10.5 10.5 0 0 0 44.4 69.4 L42.1 77.2 A2.9 2.9 0 0 0 44.9 81 L55.1 81 A2.9 2.9 0 0 0 57.9 77.2 L55.6 69.4 A10.5 10.5 0 0 0 50 50 Z M50 55.3 A5.2 5.2 0 1 1 49.99 55.3 Z';

const FAVICON_PATH =
  'M22 16 L78 16 A6 6 0 0 1 84 22 L84 78 A6 6 0 0 1 78 84 L22 84 A6 6 0 0 1 16 78 L16 22 A6 6 0 0 1 22 16 Z M30 27 A4 4 0 0 0 30 35 L70 35 A4 4 0 0 0 70 27 Z M50 44 A12 12 0 0 0 43.6 66.2 L40.9 75.1 A3.2 3.2 0 0 0 44 79.5 L56 79.5 A3.2 3.2 0 0 0 59.1 75.1 L56.4 66.2 A12 12 0 0 0 50 44 Z M50 50.2 A6 6 0 1 1 49.99 50.2 Z';

export const RackLockMark: React.FC<RackLockMarkProps> = ({ variant = 'mark', ...props }) => (
  <svg viewBox="0 0 100 100" role="img" aria-label="OS-Baka" {...props}>
    <path fill="currentColor" fillRule="evenodd" d={variant === 'favicon' ? FAVICON_PATH : MARK_PATH} />
  </svg>
);
