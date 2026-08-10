import type { PropsWithChildren } from 'react'

/**
 * Initial-page transition only. It deliberately does not intercept navigation:
 * the current site has no client router yet.
 */
export function PageTransition({ children }: PropsWithChildren) {
  return <div className="jiuin-temporary-page-transition" data-entered="true">{children}</div>
}
