import {test,expect} from 'bun:test'
import {splitMessage,formatDigest,localeFrom,deliveryFailure} from './digest'
test('localized digest contains only supplied published content and links',()=>{
  const d:any={articles:[{slug:'real',title:{ru:'Факты',kk:'Деректер',en:'Facts'}}],opportunities:[],locale:'kk'}
  const text=formatDigest(d,'https://creata.example')
  expect(text).toContain('Деректер');expect(text).toContain('/kk/articles/real');expect(text).not.toContain('undefined')
})
test('message splitting preserves emoji and Telegram limits',()=>{
  const source='Привет 🌍 '.repeat(1000)
  const parts=splitMessage(source)
  expect(parts.join('')).toBe(source)
  expect(parts.every(p=>p.length<=3800&&!/[\uD800-\uDBFF]$/.test(p))).toBe(true)
})
test('locale choices are limited to supported languages',()=>{
  expect(localeFrom('kk-KZ')).toBe('kk');expect(localeFrom('es')).toBe('en');expect(localeFrom('ru')).toBe('ru')
})
test('uncertain and partial sends are not blindly retried',()=>{
  expect(deliveryFailure(new Error('timeout'),0).unknown).toBe(true)
  expect(deliveryFailure({error_code:429,parameters:{retry_after:45}},0)).toEqual({error:'Telegram rate limit',retryAfterSeconds:45,unknown:false})
  expect(deliveryFailure({error_code:429,parameters:{retry_after:45}},1).unknown).toBe(true)
})
test('an outside-interest article is content even without other matches',()=>{
  const d:any={locale:'en',articles:[],opportunities:[],outsideInterest:{slug:'outside',title:{ru:'Другое',kk:'Басқа',en:'Outside'}}}
  const text=formatDigest(d,'https://creata.example')
  expect(text).toContain('Outside');expect(text).not.toContain('No new stories yet.')
})
