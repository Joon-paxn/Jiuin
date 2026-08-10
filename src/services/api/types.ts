export type ApiResponse<T> = {
  code: number
  message: string
  data?: T
}

export type SiteInfo = {
  name: string
  project: string
  domain: string
}

export type CopyrightInfo = {
  year: number
  text: string
}
