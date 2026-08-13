import assert from 'node:assert/strict'
import test from 'node:test'

import { resolveMusicMediaUrl } from '../src/services/api/mediaUrl.ts'

test('accepts only root-relative public music paths in same-origin production', () => {
  assert.equal(resolveMusicMediaUrl('/media/music/full/id.mp3', '', 'https:'), '/media/music/full/id.mp3')
  assert.equal(resolveMusicMediaUrl('/media/music/covers/id.jpg', '', 'https:'), '/media/music/covers/id.jpg')
  assert.equal(resolveMusicMediaUrl('media/music/full/id.mp3', '', 'https:'), undefined)
  assert.equal(resolveMusicMediaUrl('/images/cover.jpg', '', 'https:'), undefined)
  assert.equal(resolveMusicMediaUrl('https://jiuin.cn/media/music/full/id.mp3', '', 'https:'), undefined)
})

test('resolves development media only against the configured API origin', () => {
  const apiBase = 'http://127.0.0.1:8080'
  assert.equal(resolveMusicMediaUrl('/media/music/lite/id.mp3', apiBase, 'http:'), 'http://127.0.0.1:8080/media/music/lite/id.mp3')
  assert.equal(resolveMusicMediaUrl('http://127.0.0.1:8080/media/music/full/id.mp3', apiBase, 'http:'), 'http://127.0.0.1:8080/media/music/full/id.mp3')
  assert.equal(resolveMusicMediaUrl('https://example.com/media/music/full/id.mp3', apiBase, 'http:'), undefined)
})

test('rejects unsafe URL forms and HTTPS downgrade', () => {
  assert.equal(resolveMusicMediaUrl('//example.com/media/music/full/id.mp3', '', 'https:'), undefined)
  assert.equal(resolveMusicMediaUrl('/media/music/full/id.mp3?token=secret', '', 'https:'), undefined)
  assert.equal(resolveMusicMediaUrl('/media/music/full/id.mp3#fragment', '', 'https:'), undefined)
  assert.equal(resolveMusicMediaUrl('http://user:pass@api.example/media/music/full/id.mp3', 'http://api.example', 'http:'), undefined)
  assert.equal(resolveMusicMediaUrl('/media/music/full/id.mp3', 'http://api.example', 'https:'), undefined)
})
