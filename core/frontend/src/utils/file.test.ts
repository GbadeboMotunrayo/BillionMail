import { describe, it, expect } from 'vitest'
import { decodeTextBuffer } from './file'

describe('decodeTextBuffer', () => {
  it('reads UTF-8 and strips a BOM', () => {
    const bytes = new TextEncoder().encode('\uFEFFemail,name\na@b.com,张三')
    expect(decodeTextBuffer(bytes.buffer)).toBe('email,name\na@b.com,张三')
  })

  it('falls back to GBK for files Excel saved on Chinese Windows', () => {
    // "张三" in GBK
    const gbk = new Uint8Array([0x61, 0x2c, 0xd5, 0xc5, 0xc8, 0xfd])
    expect(decodeTextBuffer(gbk.buffer)).toBe('a,张三')
  })
})
