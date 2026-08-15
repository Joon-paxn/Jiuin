function publicAsset(path: string) {
  const base = import.meta.env.BASE_URL.endsWith('/')
    ? import.meta.env.BASE_URL
    : `${import.meta.env.BASE_URL}/`

  return `${base}${path}`
}

/** Jiuin 主站本地背景池。 */
export const backgrounds = [
  publicAsset('backgrounds/img1.jpg'),
  publicAsset('backgrounds/img2.jpg'),
  publicAsset('backgrounds/img3.jpg'),
  publicAsset('backgrounds/img4.jpg'),
  publicAsset('backgrounds/img5.jpg'),
  publicAsset('backgrounds/img6.jpg'),
  publicAsset('backgrounds/img7.jpg'),
  publicAsset('backgrounds/img8.jpg'),
  publicAsset('backgrounds/img9.jpg'),
  publicAsset('backgrounds/img10.jpg'),
] as const

/** 可由 BackgroundSystem props 或 BackgroundConfig 覆盖的默认视觉参数。 */
export const backgroundSystemDefaults = {
  backgroundBlur: 10,
  backgroundOverlayOpacity: 0.58,
  backgroundImageBrightness: 0.86,
  backgroundImageOpacity: 1,
  backgroundTransitionDuration: 720,
} as const
