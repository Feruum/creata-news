import {creataWordmarkPath,creataWordmarkViewBox} from './wordmark'

export function CreataBrand({className=''}:{className?:string}) {
  return <span className={`creata-wordmark ${className}`} role='img' aria-label='Creata News'>
    <svg className='creata-logo-full size-auto' xmlns='http://www.w3.org/2000/svg' viewBox={creataWordmarkViewBox} aria-hidden='true' focusable='false'>
      <path className='creata-logo-name' d={creataWordmarkPath} fill='currentColor'/>
      <text className='creata-logo-news' x='900' y='120' fontFamily='Creata Wordmark' fontSize='148.84' fontWeight='500' textLength='570' lengthAdjust='spacingAndGlyphs'>NEWS</text>
    </svg>
  </span>
}

export function CreataMark({className=''}:{className?:string}) {
  return <span className={`creata-symbol ${className}`} role='img' aria-label='Creata News'><svg className='size-auto' xmlns='http://www.w3.org/2000/svg' viewBox='0 0 128 128' aria-hidden='true' focusable='false'><path d='M0 0h128v25.6H0zM0 51.2h128v25.6H0zM0 102.4h128V128H0z' fill='currentColor'/></svg></span>
}
