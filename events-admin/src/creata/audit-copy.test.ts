import {expect,test} from 'bun:test'
import {tr} from './i18n'

test('Russian follow-ups describe story development without changing other translations',()=>{
 expect(tr('storyFollowUps','ru')).toBe('Развитие сюжета')
 expect(tr('storyFollowUps','kk')).toBe('Қайта тексеру')
 expect(tr('storyFollowUps','en')).toBe('Story follow-ups')
})
