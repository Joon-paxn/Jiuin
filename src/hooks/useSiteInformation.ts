import { useEffect, useState } from 'react'
import { site } from '../config/site'
import { siteApi } from '../services/api/site'

type SiteInformation = {
  name: string
  project: string
  domain: string
  copyrightYear?: number
  copyrightText?: string
}

const fallbackSiteInformation: SiteInformation = {
  name: site.chineseName,
  project: site.name,
  domain: site.domain,
}

/** 站点公共数据的前端接入点；未配置 API 时继续使用静态品牌回退。 */
export function useSiteInformation() {
  const [information, setInformation] = useState<SiteInformation>(fallbackSiteInformation)

  useEffect(() => {
    let ignored = false

    async function loadSiteInformation() {
      try {
        const [siteInfo, copyright] = await Promise.all([
          siteApi.getInfo(),
          siteApi.getCopyright(),
        ])

        if (!ignored) {
          setInformation({
            name: siteInfo.name,
            project: siteInfo.project,
            domain: siteInfo.domain,
            copyrightYear: copyright.year,
            copyrightText: copyright.text,
          })
        }
      } catch {
        // API 未配置或暂不可用时使用静态品牌信息，避免影响首屏展示。
      }
    }

    void loadSiteInformation()

    return () => {
      ignored = true
    }
  }, [])

  return information
}
