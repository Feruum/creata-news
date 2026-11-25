import {expect,test} from 'bun:test'
import {createPipelineDemo} from './index'
import type {Opportunity} from '@creata/contracts'
import content from '../../../creatanews/src/demo/content.json'
import reader from '../../../creatanews/src/demo/reader-data.json'
const oldTitle={ru:'Designthon: дизайн-марафон TransforNATION',en:'Designthon: TransforNATION design marathon',kk:'Designthon: TransforNATION дизайн марафоны'}
const newTitle={ru:'Designathon: дизайн-марафон TransforNATION',en:'Designathon: TransforNATION design marathon',kk:'Designathon: TransforNATION дизайн марафоны'}
const initial=content.opportunities.find(item=>item.id==='o-designthon') as Opportunity
const seed={articles:[],categories:[],opportunities:[{...initial,title:oldTitle}]}
test('Designathon is correctly spelt in both generated demo bundles',()=>{
 for(const bundle of [content,reader])expect(bundle.opportunities.find(item=>item.id==='o-designthon')?.title).toEqual(newTitle)
})
test('old saved demo opportunities correct only seeded typo, preserving editor changes',async()=>{
 const storage=new Map<string,string>()
 const demo=createPipelineDemo(seed,storage)
 await demo.request('/admin/opportunities/o-designthon',{method:'PATCH',body:JSON.stringify({title:{...oldTitle,kk:'Редакционная правка'}})})
 const reopened=createPipelineDemo({...seed,opportunities:[{...initial,title:newTitle}]},storage)
 const {items}=await reopened.request<{items:Opportunity[]}>('/admin/opportunities')
 expect(items.find(item=>item.id==='o-designthon')?.title).toEqual({...newTitle,kk:'Редакционная правка'})
})
