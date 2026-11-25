import { lazy, Suspense, useMemo } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider, createRootRoute, createRoute, createRouter, Navigate } from '@tanstack/react-router'
import { AdminProvider } from './context'
import { Shell } from './shell'
import { Loading } from './ui'
const EditorialPage = lazy(() => import('./editorial-pages').then(m => ({ default: m.EditorialPage })))
const HermesPage = lazy(() => import('./hermes-pages').then(m => ({ default: m.HermesPage })))
const ArticleEditor = lazy(() => import('./article-editor').then(m => ({ default: m.ArticleEditor })))
const GraphicEditor = lazy(() => import('./graphics-page').then(m => ({ default: m.GraphicEditor })))
const NewspaperEditor = lazy(() => import('./newspaper-editor').then(m => ({ default: m.NewspaperEditor })))
import type { Mode } from './types'
import './styles.css'
import './cats-theme.css'
import {adminBase,adminPath} from './demo'
export function AdminApp({ mode }: { mode: Mode }) {
  const queryClient = useMemo(() => new QueryClient({ defaultOptions: { queries: { staleTime: 15_000, retry: 1, refetchOnWindowFocus: false }, mutations: { retry: false } } }), [])
  const router = useMemo(() => {
    const root = createRootRoute({ component: () => <AdminProvider mode={mode}><Shell/></AdminProvider> })
    const index = createRoute({ getParentRoute: () => root, path: '/', component: () => <Navigate to={mode === 'hermes' ? '/sources' : '/today'}/> })
    const section = createRoute({ getParentRoute: () => root, path: '/$section', component: () => <Suspense fallback={<Loading/>}>{mode === 'hermes' ? <HermesPage/> : <EditorialPage/>}</Suspense> })
    const editor = createRoute({ getParentRoute: () => root, path: '/articles/$articleId', component: () => <Suspense fallback={<Loading/>}><ArticleEditor/></Suspense> })
    const newspaper = createRoute({ getParentRoute: () => root, path: '/newspapers/$issueId', component: () => <Suspense fallback={<Loading/>}><NewspaperEditor/></Suspense> })
    const graphic = createRoute({ getParentRoute: () => root, path: '/charts/$articleId', component: () => <Suspense fallback={<Loading/>}><GraphicEditor/></Suspense> })
    return createRouter({ basepath:adminBase||'/',routeTree: root.addChildren(mode === 'editorial' ? [index, section, editor, newspaper, graphic] : [index, section]), defaultNotFoundComponent: () => <div className='empty'><h1>404</h1><a href={adminPath('/')}>Creata News</a></div> })
  }, [mode])
  return <QueryClientProvider client={queryClient}><RouterProvider router={router}/></QueryClientProvider>
}
