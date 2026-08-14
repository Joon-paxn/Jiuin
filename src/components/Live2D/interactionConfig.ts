export type Live2DInteractionPart = 'head' | 'face' | 'body' | 'hand' | 'other'

export type Live2DInteractionEvent = {
  hitAreaName: string
  part: Live2DInteractionPart
  message: string
}

/**
 * Dialogue copy belongs here rather than the React view. New model hit areas
 * are normalized at runtime and reuse the matching pool automatically.
 */
export const interactionDialogues: Readonly<Record<Live2DInteractionPart, readonly string[]>> = {
  head: [
    '欸……不要一直摸我的头啦。',
    '头发会乱掉的，不过这次就原谅你。',
    '唔，轻一点。',
  ],
  face: [
    '靠得太近了啦……',
    '不要盯着我看那么久。',
    '脸上有什么吗？',
  ],
  body: [
    '你在看哪里呀？',
    '这里会有点痒……',
    '别闹啦。',
  ],
  hand: [
    '要牵手吗？',
    '手在这里哦。',
    '轻轻碰就好了。',
  ],
  other: [
    '嗯？怎么了？',
    '我有在听哦。',
  ],
}

export function resolveInteractionPart(hitAreaName: string): Live2DInteractionPart {
  const name = hitAreaName.toLowerCase()

  if (/(face|cheek|mouth|eye|脸|面|頬|口|眼)/.test(name)) return 'face'
  if (/(head|hair|ear|头|頭|发|髪|耳)/.test(name)) return 'head'
  if (/(hand|arm|finger|手|腕|臂)/.test(name)) return 'hand'
  if (/(body|bust|torso|chest|躯干|身体|身|胸)/.test(name)) return 'body'
  return 'other'
}

export function selectInteractionDialogue(
  part: Live2DInteractionPart,
  previousMessage?: string,
) {
  const candidates = interactionDialogues[part]
  if (candidates.length < 2 || !previousMessage) {
    return candidates[Math.floor(Math.random() * candidates.length)]
  }

  const alternatives = candidates.filter((message) => message !== previousMessage)
  return alternatives[Math.floor(Math.random() * alternatives.length)]
}
