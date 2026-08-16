import type { BackgroundThemeOverrides } from './background.types'

type HslColor = {
  hue: number
  saturation: number
  lightness: number
}

const analysisSize = 48

function clamp(value: number, minimum: number, maximum: number) {
  return Math.min(maximum, Math.max(minimum, value))
}

function normalizeHue(hue: number) {
  return ((hue % 360) + 360) % 360
}

function toCssHsl(hue: number, saturation: number, lightness: number, opacity?: number) {
  const color = `${Math.round(normalizeHue(hue))} ${Math.round(clamp(saturation, 0, 100))}% ${Math.round(clamp(lightness, 0, 100))}%`

  return opacity === undefined ? `hsl(${color})` : `hsl(${color} / ${opacity})`
}

function rgbToHsl(red: number, green: number, blue: number): HslColor {
  const r = red / 255
  const g = green / 255
  const b = blue / 255
  const maximum = Math.max(r, g, b)
  const minimum = Math.min(r, g, b)
  const delta = maximum - minimum
  const lightness = (maximum + minimum) / 2

  if (delta === 0) {
    return { hue: 0, saturation: 0, lightness: lightness * 100 }
  }

  const saturation = delta / (1 - Math.abs((2 * lightness) - 1))
  let hue: number

  if (maximum === r) {
    hue = ((g - b) / delta) % 6
  } else if (maximum === g) {
    hue = ((b - r) / delta) + 2
  } else {
    hue = ((r - g) / delta) + 4
  }

  return {
    hue: normalizeHue(hue * 60),
    saturation: saturation * 100,
    lightness: lightness * 100,
  }
}

function getHueFromUrl(url: string) {
  let hash = 2_166_136_261

  for (let index = 0; index < url.length; index += 1) {
    hash ^= url.charCodeAt(index)
    hash = Math.imul(hash, 16_777_619)
  }

  return hash >>> 0
}

function createThemeOverrides({ hue, saturation, lightness }: HslColor): BackgroundThemeOverrides {
  const sourceSaturation = clamp(saturation, 42, 82)
  const sourceLightness = clamp(lightness, 20, 80)
  const primaryLightness = clamp(80 + ((sourceLightness - 50) * 0.08), 76, 84)
  const secondaryHue = normalizeHue(hue + 30)
  const accentHue = normalizeHue(hue - 30)
  const darkSaturation = clamp(sourceSaturation * 0.58, 24, 48)

  return {
    primary: toCssHsl(hue, sourceSaturation, primaryLightness),
    primaryStrong: toCssHsl(hue, clamp(sourceSaturation * 0.92, 40, 76), 66),
    secondary: toCssHsl(secondaryHue, clamp(sourceSaturation * 0.78, 38, 70), 76),
    accent: toCssHsl(accentHue, clamp(sourceSaturation * 0.92, 42, 78), 76),
    background: toCssHsl(hue, darkSaturation, 10),
    backgroundMid: toCssHsl(hue + 10, darkSaturation, 15),
    backgroundEnd: toCssHsl(hue - 12, darkSaturation, 18),
    glass: toCssHsl(hue, clamp(sourceSaturation * 0.4, 20, 42), 88, 0.1),
    glassStrong: toCssHsl(hue, clamp(sourceSaturation * 0.42, 20, 44), 92, 0.17),
    progress: toCssHsl(hue, sourceSaturation, primaryLightness),
    highlight: toCssHsl(hue, sourceSaturation, 76, 0.22),
    overlay: toCssHsl(hue, darkSaturation, 8, 0.46),
    overlayStrong: toCssHsl(hue, darkSaturation, 7, 0.72),
    primaryWash: toCssHsl(hue, sourceSaturation, primaryLightness, 0.2),
    accentWash: toCssHsl(accentHue, sourceSaturation, 76, 0.2),
    shadow: toCssHsl(hue, darkSaturation, 5, 0.42),
    shadowSoft: toCssHsl(hue, darkSaturation, 5, 0.28),
    focusRing: toCssHsl(hue, sourceSaturation, primaryLightness, 0.7),
  }
}

function extractDominantColor(pixels: Uint8ClampedArray): HslColor | null {
  const hueBucketCount = 24
  const buckets = Array.from({ length: hueBucketCount }, () => ({ weight: 0, saturation: 0, lightness: 0 }))

  for (let index = 0; index < pixels.length; index += 16) {
    const alpha = pixels[index + 3] / 255

    if (alpha < 0.45) {
      continue
    }

    const color = rgbToHsl(pixels[index], pixels[index + 1], pixels[index + 2])

    if (color.saturation < 9) {
      continue
    }

    const exposureWeight = color.lightness < 4 || color.lightness > 96 ? 0.35 : 1
    const weight = alpha * exposureWeight * (0.2 + (color.saturation / 100))
    const bucketIndex = Math.min(hueBucketCount - 1, Math.floor(color.hue / (360 / hueBucketCount)))
    const bucket = buckets[bucketIndex]

    bucket.weight += weight
    bucket.saturation += color.saturation * weight
    bucket.lightness += color.lightness * weight
  }

  const dominantBucket = buckets.reduce((best, bucket) => bucket.weight > best.weight ? bucket : best)
  const dominantIndex = buckets.indexOf(dominantBucket)

  if (dominantBucket.weight === 0 || dominantIndex < 0) {
    return null
  }

  return {
    hue: (dominantIndex + 0.5) * (360 / hueBucketCount),
    saturation: dominantBucket.saturation / dominantBucket.weight,
    lightness: dominantBucket.lightness / dominantBucket.weight,
  }
}

function analyzeImage(image: HTMLImageElement): HslColor | null {
  try {
    const canvas = document.createElement('canvas')
    canvas.width = analysisSize
    canvas.height = analysisSize
    const context = canvas.getContext('2d')

    if (!context) {
      return null
    }

    context.drawImage(image, 0, 0, analysisSize, analysisSize)
    return extractDominantColor(context.getImageData(0, 0, analysisSize, analysisSize).data)
  } catch {
    // Cross-origin images without pixel-reading CORS taint the canvas. The
    // caller will retain the deterministic URL fallback without another load.
    return null
  }
}

/**
 * CORS 不可用时仍提供稳定且会随 URL 变化的回退主题；图片本身不会因此被拦截显示。
 */
export function createFallbackBackgroundTheme(url: string) {
  const hash = getHueFromUrl(url)

  return createThemeOverrides({
    hue: hash % 360,
    saturation: 50 + ((hash >>> 9) % 24),
    lightness: 38 + ((hash >>> 17) % 25),
  })
}

/** 读取已允许 CORS 的图片像素，以当前背景的主色生成主题变量。 */
export function analyzeBackgroundTheme(image: HTMLImageElement, url: string) {
  const color = analyzeImage(image)

  return color ? createThemeOverrides(color) : createFallbackBackgroundTheme(url)
}
