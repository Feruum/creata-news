import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { AdminApp } from '../../events-admin/src/creata/app'
createRoot(document.getElementById('root')!).render(<StrictMode><AdminApp mode='hermes'/></StrictMode>)
