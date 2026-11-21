import type {createPipelineDemo,DemoSeed} from '@creata/demo-pipeline'
let engine:Promise<ReturnType<typeof createPipelineDemo>>|undefined
export function getDemoPipeline(){
 if(!engine)engine=Promise.all([import('@creata/demo-pipeline'),import('./content.json')]).then(([module,seed])=>module.createPipelineDemo(seed.default as unknown as DemoSeed))
 return engine
}
