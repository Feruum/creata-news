import type {Locale} from '@/lib/models'
import {demoCopy} from './config'
const names={ru:{editorial:'Редакция',hermes:'Пульт Hermes'},kk:{editorial:'Редакция',hermes:'Hermes басқаруы'},en:{editorial:'Newsroom',hermes:'Hermes control'}}
export function DemoNavigation({locale}:{locale:Locale}){
 const t=names[locale]
 return <div className='preview-banner' data-testid='demo-notice'>{demoCopy[locale].notice}<span> · <a href='/editorial/today'>{t.editorial} ↗</a> · <a href='/hermes/sources'>{t.hermes} ↗</a></span></div>
}
