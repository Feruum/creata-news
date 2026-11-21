import type { NextConfig } from 'next'
const config: NextConfig = {
  poweredByHeader: false, devIndicators: false, output: 'standalone',
  distDir: process.env.CREATA_BUILD_DIR || '.next',
  async rewrites(){return {beforeFiles:[{source:'/news/photos/:path*',destination:'/api/legacy-media/news/photos/:path*'},{source:'/newspaper/issues/:path*',destination:'/api/legacy-media/newspaper/issues/:path*'}],afterFiles:process.env.NEXT_PUBLIC_CREATA_DEMO_MODE==='1'?[{source:'/editorial/:path*',destination:'/demo-apps/editorial.html'},{source:'/hermes/:path*',destination:'/demo-apps/hermes.html'}]:[],fallback:[]}},
  async headers() { return [{source: '/api/creata/:path*', headers: [{key: 'Cache-Control', value: 'private, no-store'}]}] }
}
export default config
