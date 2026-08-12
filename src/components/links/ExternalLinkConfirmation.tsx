import { useEffect, useId, useMemo, useRef, useState } from 'react'
import type { ExternalLink } from '../../services/api/types'
import { toSafeExternalUrl } from '../../utils/safeUrl'

type ExternalLinkConfirmationProps = {
  link: ExternalLink
}

/** Ensures configured third-party destinations are never opened directly from the UI. */
export function ExternalLinkConfirmation({ link }: ExternalLinkConfirmationProps) {
  const dialogRef = useRef<HTMLDialogElement>(null)
  const headingId = useId()
  const [isOpen, setIsOpen] = useState(false)
  const safeUrl = useMemo(() => toSafeExternalUrl(link.url), [link.url])

  useEffect(() => {
    const dialog = dialogRef.current
    if (!dialog) {
      return
    }

    if (isOpen && !dialog.open) {
      dialog.showModal()
    }
    if (!isOpen && dialog.open) {
      dialog.close()
    }
  }, [isOpen])

  // Do not turn an unexpected API value into an href. This is deliberately a
  // second boundary after the Go service's external-link validation.
  if (!safeUrl) {
    return null
  }

  return (
    <>
      <button className="site-footer__external-link" type="button" onClick={() => setIsOpen(true)}>
        {link.name}
      </button>
      <dialog
        ref={dialogRef}
        className="external-link-confirmation"
        aria-labelledby={headingId}
        onCancel={() => setIsOpen(false)}
        onClose={() => setIsOpen(false)}
      >
        <div className="external-link-confirmation__content">
          <span className="external-link-confirmation__eyebrow">EXTERNAL DESTINATION</span>
          <h2 id={headingId}>即将离开霁雪居</h2>
          <p>{link.description || `你将前往 ${link.name}。`}</p>
          <code>{safeUrl}</code>
          <div className="external-link-confirmation__actions">
            <button type="button" className="ui-button ui-button--ghost ui-button--sm" onClick={() => setIsOpen(false)}>取消</button>
            <a className="ui-button ui-button--primary ui-button--sm" href={safeUrl} target="_blank" rel="noopener noreferrer" onClick={() => setIsOpen(false)}>继续前往</a>
          </div>
        </div>
      </dialog>
    </>
  )
}
