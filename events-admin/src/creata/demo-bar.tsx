import {useQueryClient} from '@tanstack/react-query'
import {useAdmin} from './hooks'
import {applicationUrls,getDemoPipeline} from './demo'
import {Button} from './ui'
const copy={
 ru:{notice:'Демо · Сбор, AI и отправки симулируются. Изменения сохраняются только в этом браузере.',editorial:'Редакция',hermes:'Пульт Hermes',site:'Открыть сайт',reset:'Сбросить демо',confirm:'Сбросить изменения демо в этом браузере?'},
 kk:{notice:'Демо · Жинау, AI және жіберу модельденеді. Өзгерістер тек осы браузерде сақталады.',editorial:'Редакция',hermes:'Hermes басқаруы',site:'Сайтты ашу',reset:'Демоны қалпына келтіру',confirm:'Осы браузердегі демо өзгерістерін өшіру керек пе?'},
 en:{notice:'Demo · Collection, AI and deliveries are simulated. Changes stay in this browser only.',editorial:'Newsroom',hermes:'Hermes control',site:'Open site',reset:'Reset demo',confirm:'Reset demo changes in this browser?'},
}
export function DemoBar(){
 const {locale}=useAdmin(),client=useQueryClient(),t=copy[locale]
 return <section className='callout' data-testid='admin-demo-notice'><p>{t.notice}</p><div className='form-actions'><a className='text-button' href={applicationUrls.editorial+'/today'}>{t.editorial}</a><a className='text-button' href={applicationUrls.hermes+'/sources'}>{t.hermes}</a><a className='text-button' href={applicationUrls.web+'/'+locale}>{t.site}</a><Button className='small' variant='ghost' onClick={async()=>{if(!window.confirm(t.confirm))return;(await getDemoPipeline()).reset();try{localStorage.removeItem('creata:demo:v1')}catch{}await client.invalidateQueries()}}>{t.reset}</Button></div></section>
}
