import { useEffect } from 'react'

const SITE_NAME = 'PasarKita'

/** Sets the browser tab title while the calling page is mounted. */
export function usePageTitle(title?: string) {
  useEffect(() => {
    document.title = title ? `${title} | ${SITE_NAME}` : SITE_NAME
  }, [title])
}
