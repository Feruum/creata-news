'use client'
import { useState,type ReactNode } from 'react'
import {CreataBrand} from '@creata/brand/logo'
export function MediaImage({src,alt='',priority=false,className='',fallback}: {src?:string;alt?:string;priority?:boolean;className?:string;fallback?:ReactNode}) {const[failed,setFailed]=useState(false);return <div className={`media-frame ${className}`}>{src&&!failed?<img src={src} alt={alt} loading={priority?'eager':'lazy'} fetchPriority={priority?'high':'auto'} onError={()=>setFailed(true)}/>:fallback||<div className='media-fallback' role='img' aria-label={alt}><span className='fallback-orbit'/><CreataBrand className='media-fallback-brand'/></div>}</div>}
