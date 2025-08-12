import React from 'react'
import { Routes, Route, Link } from 'react-router-dom'
import Landing from './pages/Landing.jsx'
import SiteDetail from './pages/SiteDetail.jsx'

export default function App(){
    return (
        <div className="min-h-screen bg-zinc-50 flex flex-col">
            <header className="sticky top-0 z-10 bg-white/80 backdrop-blur border-b border-zinc-200">
                <div className="mx-auto max-w-7xl px-4 py-3 flex items-center gap-4">
                    <Link to="/" className="text-lg font-semibold text-zinc-800">SiteChecker</Link>
                </div>
            </header>
            <Routes>
                <Route path="/" element={<Landing/>} />
                <Route path="/site/:siteId" element={<SiteDetail/>} />
            </Routes>
            <footer className="w-full border-t border-zinc-200 bg-white">
                <div className="mx-auto max-w-7xl px-4 py-3 flex items-center">
                    <div className="mx-auto text-xs text-zinc-500">power by Hamed0x</div>
                    <a href="https://www.buymeacoffee.com/hamed0x" target="_blank" rel="noreferrer" className="inline-flex items-center gap-2 rounded-full border border-amber-400 bg-amber-50 px-3 py-1 text-sm text-amber-700 hover:bg-amber-100">Buy me a coffee</a>
                </div>
            </footer>
        </div>
    )
}