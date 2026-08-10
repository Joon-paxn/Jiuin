const disabledValues = new Set(['false', '0', 'off'])

/**
 * This experiment is intentionally enabled by default for visual review.
 * Set VITE_ENABLE_TEMPORARY_ANIMATION=false to render the application unchanged.
 */
export const temporaryAnimationEnabled = !disabledValues.has(
  String(import.meta.env.VITE_ENABLE_TEMPORARY_ANIMATION ?? '').toLowerCase(),
)
