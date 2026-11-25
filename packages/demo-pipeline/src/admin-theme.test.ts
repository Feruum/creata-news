import { expect, test } from 'bun:test'
import { createPipelineDemo, DEMO_PIPELINE_KEY } from './index'

const seed = { articles: [], categories: [], opportunities: [] }
const patch = (adminTheme: unknown): RequestInit => ({ method: 'PATCH', body: JSON.stringify({ adminTheme }) })

test('appearance is validated and persisted for the signed-in demo account only', async () => {
  const storage=new Map<string,string>()
  const demo=createPipelineDemo(seed,storage)
  expect(await demo.request('/auth/me')).toMatchObject({user:{adminTheme:'light'}})
  expect(await demo.request('/account',patch('cats'))).toMatchObject({adminTheme:'cats'})
  await expect(demo.request('/account',patch('wrong'))).rejects.toMatchObject({status:400})
  await expect(demo.request('/account',patch(null))).rejects.toMatchObject({status:400})
  const reopened=createPipelineDemo(seed,storage)
  expect(await reopened.request('/auth/me')).toMatchObject({user:{adminTheme:'cats'}})
  const state=JSON.parse(storage.get(DEMO_PIPELINE_KEY)!)
  const current=state.session.user
  state.session.user=state.users.find((user:{id:string})=>user.id==='demo-editor')
  storage.set(DEMO_PIPELINE_KEY,JSON.stringify(state))
  expect(await reopened.request('/auth/me')).toMatchObject({user:{adminTheme:'light'}})
  await reopened.request('/account',patch('dark'))
  const switched=JSON.parse(storage.get(DEMO_PIPELINE_KEY)!)
  switched.session.user=current
  storage.set(DEMO_PIPELINE_KEY,JSON.stringify(switched))
  expect(await reopened.request('/auth/me')).toMatchObject({user:{adminTheme:'cats'}})
  await reopened.request('/auth/logout',{method:'POST'})
  await reopened.request('/auth/dev-login',{method:'POST',body:JSON.stringify({role:'chief_editor'})})
  expect(await reopened.request('/auth/me')).toMatchObject({user:{adminTheme:'cats'}})
})
