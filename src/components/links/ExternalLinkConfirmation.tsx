import { useEffect, useId, useRef, useState } from 'react'
import type { ExternalLink } from '../../services/api/types'

type ExternalLinkConfirmationProps = {
  link: ExternalLink
}

/** Ensures configured third-party destinations are never opened directly from the UI. */
export function ExternalLinkConfirmation({ link }: ExternalLinkConfirmationProps) {
  const dialogRef = useRef<HTMLDialogElement>(null)
  const headingId = useId()
  const [isOpen, setIsOpen] = useState(false)

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
          <code>{link.url}</code>
          <div className="external-link-confirmation__actions">
            <button type="button" className="ui-button ui-button--ghost ui-button--sm" onClick={() => setIsOpen(false)}>取消</button>
            <a className="ui-button ui-button--primary ui-button--sm" href={link.url} target="_blank" rel="noopener noreferrer" onClick={() => setIsOpen(false)}>继续前往</a>
          </div>
        </div>
      </dialog>
    </>
  )
}
