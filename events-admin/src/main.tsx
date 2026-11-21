import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'

import { AdminApp } from './creata/app'
createRoot(document.getElementById('root')!).render(<StrictMode><AdminApp mode='editorial'/></StrictMode>)
