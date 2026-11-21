'use client'
import {useEffect,useId,useState} from 'react'
import {datasetCsv,graphicCopy,numberLabel,axisLabel,donutSegments,text,type GraphicData,type Locale} from './model'
const colors=['var(--creata-accent,#4B44F0)','#22a58a','#c574e8','#e99544','#4da9d9','#db6d8b']
export function GraphicFigure({data:g,locale,title,compact=false}:{data:GraphicData;locale:Locale;title:string;compact?:boolean}){
 const id=useId().replaceAll(':',''),[point,setPoint]=useState<number|null>(null),t=graphicCopy[locale]
 const [ready,setReady]=useState(false)
 useEffect(()=>setReady(true),[])
 const W=720,H=360,L=68,R=24,T=25,B=58,pw=W-L-R,ph=H-T-B
 const values=g.series.flatMap(s=>s.values),low=Math.min(0,...values),high=Math.max(0,...values),span=(high-low)||1
 const y=(n:number)=>T+(high-n)/span*ph,x=(i:number)=>L+(i+.5)*pw/g.labels.length,zero=y(0)
 const inspected=point===null?null:g.labels[point]
 const download=()=>{const url=URL.createObjectURL(new Blob(['\uFEFF'+datasetCsv(g,locale)],{type:'text/csv;charset=utf-8'}));const a=document.createElement('a');a.href=url;a.download='creata-chart.csv';a.click();URL.revokeObjectURL(url)}
 return <figure className={`cg-figure${compact?' cg-compact':''}`} data-chart-ready={ready}>
  <div className='cg-plot' tabIndex={compact?-1:0} role='group' aria-label={`${title}. ${t.inspect}`} onKeyDown={e=>{if(['ArrowLeft','ArrowRight','Home','End'].includes(e.key)){e.preventDefault();setPoint(e.key==='Home'?0:e.key==='End'?g.labels.length-1:Math.max(0,Math.min(g.labels.length-1,(point??0)+(e.key==='ArrowLeft'?-1:1))))}}}>
   <div className='cg-unit'>{g.unit} · {g.period}</div>
   <svg viewBox={`0 0 ${W} ${H}`} role='img' aria-labelledby={`${id}-title ${id}-desc`}>
    <title id={`${id}-title`}>{title}</title><desc id={`${id}-desc`}>{g.series.map(s=>text(s.name,locale)).join(', ')+'. '+t.source+': '+g.sourceName+'. '+t.period+': '+g.period+'. '+text(g.methodology,locale)}</desc>
    {g.type==='donut'?donutSegments(g.series[0].values).map((segment,i)=><circle key={i} cx='360' cy='174' r='101.5' fill='none' stroke={colors[i%colors.length]} strokeWidth='47' strokeDasharray={[segment.length,segment.circumference].join(' ')} strokeDashoffset={segment.offset} transform='rotate(-90 360 174)' onPointerEnter={()=>setPoint(i)}><title>{[text(g.labels[i],locale)+':',numberLabel(g.series[0].values[i],locale),g.unit].join(' ')}</title></circle>) : <>
     {Array.from({length:5},(_,i)=>{const n=low+span*i/4;return <g key={i}><line x1={L} x2={W-R} y1={y(n)} y2={y(n)} className='cg-gridline'/><text x={L-12} y={y(n)+4} textAnchor='end' className='cg-tick'>{axisLabel(n,locale)}</text></g>})}
     <line x1={L} x2={W-R} y1={zero} y2={zero} className='cg-baseline'/>
     {g.series.map((s,si)=>{const points=s.values.map((v,i)=>`${x(i)},${y(v)}`).join(' '),barWidth=Math.min(64,pw/g.labels.length*.72/g.series.length);return <g key={si}>
      {g.type==='area'&&<polygon points={`${x(0)},${zero} ${points} ${x(g.labels.length-1)},${zero}`} fill={colors[si]} opacity='.12'/>}
      {g.type==='bar'?s.values.map((v,i)=><rect key={i} x={x(i)+(si-g.series.length/2)*barWidth} y={Math.min(y(v),zero)} width={Math.max(.4,barWidth-2)} height={Math.max(1,Math.abs(zero-y(v)))} rx='2' fill={colors[si]} onPointerEnter={()=>setPoint(i)}><title>{[text(g.labels[i],locale),'·',text(s.name,locale)+':',numberLabel(v,locale),g.unit].join(' ')}</title></rect>):<><polyline points={points} fill='none' stroke={colors[si]} strokeWidth='3' strokeLinejoin='round'/>{s.values.map((v,i)=><circle key={i} cx={x(i)} cy={y(v)} r={point===i?6:3} fill={colors[si]} onPointerEnter={()=>setPoint(i)}><title>{[text(g.labels[i],locale),'·',text(s.name,locale)+':',numberLabel(v,locale),g.unit].join(' ')}</title></circle>)}</>}</g>})}
     {g.labels.map((label,i)=>i%Math.max(1,Math.ceil(g.labels.length/7))===0?<text key={i} x={x(i)} y={H-26} textAnchor='middle' className='cg-tick'>{text(label,locale).slice(0,20)}</text>:null)}
    </>}
   </svg>
  </div>
  <div className='cg-legend'>{(g.type==='donut'?g.labels:g.series.map(s=>s.name)).map((name,i)=><span key={i}><i style={{background:colors[i%colors.length]}}/>{text(name,locale)}</span>)}</div>
  {!compact&&<><output className='cg-inspect' aria-live='polite'>{inspected?<><strong>{text(inspected,locale)}</strong>{g.series.map((s,i)=><span key={i}>{text(s.name,locale)}: {numberLabel(s.values[point!],locale)} {g.unit}</span>)}</>:t.inspect}</output><figcaption className='cg-provenance'><span>{t.source}: <a href={g.sourceUrl} target='_blank' rel='noopener noreferrer'>{g.sourceName} ↗</a></span><span>{t.period}: {g.period}</span></figcaption><details className='cg-data'><summary>{t.table}</summary><div className='cg-table-wrap'><table><caption>{title} · {g.unit}</caption><thead><tr><th>{t.period}</th>{g.series.map((s,i)=><th key={i}>{text(s.name,locale)}</th>)}</tr></thead><tbody>{g.labels.map((label,i)=><tr key={i}><th scope='row'>{text(label,locale)}</th>{g.series.map((s,si)=><td key={si}>{numberLabel(s.values[i],locale)}</td>)}</tr>)}</tbody></table></div><button type='button' onClick={download}>{t.download} ↓</button></details><div className='cg-methodology'><strong>{t.methodology}</strong><p>{text(g.methodology,locale)}</p></div></>}
 </figure>
}
