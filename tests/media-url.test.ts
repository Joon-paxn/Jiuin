import assert from 'node:assert/strict'
import test from 'node:test'
import { resolveMusicMediaUrl } from '../src/services/api/mediaUrl.ts'

test('keeps production media URLs origin-relative', () => {
  assert.equal(resolveMusicMediaUrl('/media/music/track/full', '', 'https:'), '/media/music/track/full')
})

test('allows configured API-origin media but rejects cross-origin paths', () => {
  assert.equal(
    resolveMusicMediaUrl('https://jiuin.cn/media/music/track/lite', 'https://jiuin.cn', 'https:'),
    'https://jiuin.cn/media/music/track/lite',
  )
  assert.equal(resolveMusicMediaUrl('https://bkgapi.jiuin.cn/media/music/track/lite', '', 'https:'), undefined)
  assert.equal(resolveMusicMediaUrl('/media/other/track', '', 'https:'), undefined)
})
