/** Jiuin 主站 CDN 背景池。 */
export const backgrounds = [
  'https://image.cn-zj1.rains3.com/pc/img1.jpg',
  'https://image.cn-zj1.rains3.com/pc/img2.jpg',
  'https://image.cn-zj1.rains3.com/pc/img3.jpg',
  'https://image.cn-zj1.rains3.com/pc/img4.jpg',
  'https://image.cn-zj1.rains3.com/pc/img5.jpg',
  'https://image.cn-zj1.rains3.com/pc/img6.jpg',
  'https://image.cn-zj1.rains3.com/pc/img7.jpg',
  'https://image.cn-zj1.rains3.com/pc/img8.jpg',
  'https://image.cn-zj1.rains3.com/pc/img9.jpg',
  'https://image.cn-zj1.rains3.com/pc/img10.jpg',
] as const

/** 可由 BackgroundSystem props 或 BackgroundConfig 覆盖的默认视觉参数。 */
export const backgroundSystemDefaults = {
  backgroundBlur: 10,
  backgroundOverlayOpacity: 0.58,
  backgroundImageBrightness: 0.86,
  backgroundImageOpacity: 1,
  backgroundTransitionDuration: 720,
} as const
