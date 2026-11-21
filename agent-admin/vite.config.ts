import tailwindcss from '@tailwindcss/vite'
import {fileURLToPath} from 'node:url'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
export default defineConfig({base:process.env.VITE_ADMIN_BASE_PATH?process.env.VITE_ADMIN_BASE_PATH.replace(/\/$/,'')+'/':'/',plugins:[react(),tailwindcss()],server:{host:'127.0.0.1',port:5174,fs:{allow:['..']},proxy:{'/api':{target:process.env.CREATA_API_ORIGIN||'http://localhost:5555',changeOrigin:false}}},resolve:{alias:{'@':fileURLToPath(new URL('../events-admin/src',import.meta.url))},dedupe:['react','react-dom']},build:{outDir:'dist',rollupOptions:{output:{manualChunks:{'react-vendor':['react','react-dom'],'query-router':['@tanstack/react-query','@tanstack/react-router'],'motion':['motion/react']}}}}})
