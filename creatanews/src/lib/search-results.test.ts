import { expect,test } from 'bun:test'
import { matchSearchGroups,searchAvailability } from './search-results'
test('search matches translated category names and returns actual category identifiers',()=>{
  const groups=matchSearchGroups({authors:[],opportunities:[],categories:[{id:'science',slug:'science-stem',name:{ru:'Наука',kk:'Ғылым',en:'Science'}}]},'ғылым','kk')
  expect(groups.categories.map(category=>category.slug)).toEqual(['science-stem'])
  expect(matchSearchGroups({authors:[],opportunities:[],categories:[{id:'science',slug:'science-stem',name:{ru:'Наука',kk:'Ғылым',en:'Science'}}]},'unmatched','en').categories).toHaveLength(0)
})
test('partial search availability preserves successful results and never implies a complete empty result',()=>{
  expect(searchAvailability([false,true,false,false],0)).toEqual({partial:true,failed:false,empty:false})
  expect(searchAvailability([true,true,true,true],0)).toEqual({partial:false,failed:true,empty:false})
  expect(searchAvailability([false,false,false,false],0)).toEqual({partial:false,failed:false,empty:true})
  expect(searchAvailability([false,true,false,false],3)).toEqual({partial:true,failed:false,empty:false})
})
