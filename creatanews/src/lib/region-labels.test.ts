import { expect,test } from 'bun:test'
import { regionLabel } from './topics'
test('country labels stay localized without browser ICU region names and preserve unknown codes',()=>{
  const original=Intl.DisplayNames
  Object.defineProperty(Intl,'DisplayNames',{value:undefined,configurable:true,writable:true})
  try{
    expect(regionLabel('KZ','kk')).toBe('Қазақстан')
    expect(regionLabel('KZ','ru')).toBe('Казахстан')
    expect(regionLabel('KZ','en')).toBe('Kazakhstan')
    expect(regionLabel('JP','kk')).toBe('Жапония')
    expect(regionLabel('ZZ','kk')).toBe('ZZ')
  }finally{Object.defineProperty(Intl,'DisplayNames',{value:original,configurable:true,writable:true})}
})
