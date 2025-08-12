import React, { useEffect, useMemo, useState } from 'react'
import { useLocation, useParams } from 'react-router-dom'
import { Search } from 'lucide-react'

const API_BASE = ''
const KINDS = 'postMessageSend,postMessageRecv,innerHTML,eval,newFunction,setTimeoutStr,setIntervalStr,documentWrite,fetch,XMLHttpRequest,syncXHR,inlineEventHandler,directDOM'

async function getJSON(p){ const r=await fetch(API_BASE+p); if(!r.ok) throw new Error(r.status); return r.json() }
function useQuery(){ const {search}=useLocation(); return useMemo(()=>Object.fromEntries(new URLSearchParams(search)),[search]) }

export default function SiteDetail(){
    const { siteId } = useParams();
    const q = useQuery();
    const [tab,setTab]=useState('pages')
    const [pages,setPages]=useState([])
    const [eps,setEps]=useState([])
    const [sinks,setSinks]=useState([])
    const [loading,setLoading]=useState(true)
    const [filter,setFilter]=useState('')

    useEffect(()=>{ (async()=>{
        setLoading(true)
        try{
            const [p,e,s] = await Promise.all([
                getJSON(`/api/pages?site_id=${encodeURIComponent(siteId)}&limit=200&sort=scanned_at&order=desc`),
                getJSON(`/api/endpoints?site_id=${encodeURIComponent(siteId)}&limit=200&sort=last_seen&order=desc`),
                getJSON(`/api/sinks?site_id=${encodeURIComponent(siteId)}&limit=200&sort=last_detected_at&order=desc&kind=${encodeURIComponent(KINDS)}`)
            ])
            setPages(p.items||[]); setEps(e.items||[]); setSinks(s.items||[])
        } finally { setLoading(false) }
    })() },[siteId])

    const pageUrlFromQuery = q.url || ''

    const filteredPages = useMemo(()=> filter? pages.filter(x=> (x.url_norm||'').toLowerCase().includes(filter.toLowerCase())): pages, [pages,filter])
    const filteredEps   = useMemo(()=> filter? eps.filter(x=> (x.endpoint||'').toLowerCase().includes(filter.toLowerCase())): eps, [eps,filter])
    const filteredSinks = useMemo(()=> filter? sinks.filter(x=> (x.source_url||'').toLowerCase().includes(filter.toLowerCase()) || (x.kind||'').toLowerCase().includes(filter.toLowerCase())): sinks, [sinks,filter])

    return (
        <div className="mx-auto max-w-7xl w-full px-4 py-6">
            <h1 className="text-xl font-semibold text-zinc-800 mb-3 break-all">{siteId}</h1>

            <div className="flex items-center gap-2 rounded-2xl border border-zinc-300 bg-white px-3 py-2 shadow-sm mb-3">
                <Search className="h-4 w-4 text-zinc-500"/>
                <input value={filter} onChange={e=>setFilter(e.target.value)} placeholder="فیلتر…" className="w-full bg-transparent outline-none text-sm" />
            </div>

            <div className="flex gap-2 mb-3">
                {['pages','endpoints','sinks'].map(t => (
                    <button key={t} onClick={()=>setTab(t)} className={`px-3 py-1.5 rounded-full text-sm border ${tab===t?'bg-zinc-900 text-white border-zinc-900':'bg-white border-zinc-300 text-zinc-700'}`}>{t}</button>
                ))}
            </div>

            {loading ? <div className="text-sm text-zinc-500">Loading…</div> : (
                tab==='pages' ? <PagesTab items={filteredPages} highlightUrl={pageUrlFromQuery}/> :
                    tab==='endpoints' ? <EndpointsTab items={filteredEps}/> :
                        <SinksTab items={filteredSinks}/>
            )}
        </div>
    )
}

function PagesTab({ items, highlightUrl }){
    return (
        <div className="rounded-2xl overflow-hidden border border-zinc-200 bg-white">
            <table className="w-full text-sm">
                <thead className="bg-zinc-50 text-zinc-600">
                <tr><th className="text-left p-2">URL</th><th className="text-left p-2">Host</th><th className="text-left p-2">Path</th><th className="text-left p-2">Scanned</th></tr>
                </thead>
                <tbody>
                {items.map((x,i)=> (
                    <tr key={i} className={x.url_norm===highlightUrl? 'bg-amber-50' : 'hover:bg-zinc-50'}>
                        <td className="p-2 break-all">{x.url_norm || x.url}</td>
                        <td className="p-2">{x.host}</td>
                        <td className="p-2">{x.path}</td>
                        <td className="p-2 text-zinc-500">{x.scanned_at}</td>
                    </tr>
                ))}
                </tbody>
            </table>
        </div>
    )
}

function EndpointsTab({ items }){
    return (
        <div className="rounded-2xl overflow-hidden border border-zinc-200 bg-white">
            <table className="w-full text-sm">
                <thead className="bg-zinc-50 text-zinc-600">
                <tr><th className="text-left p-2">Endpoint</th><th className="text-left p-2">Category</th><th className="text-left p-2">Seen</th><th className="text-left p-2">Last</th></tr>
                </thead>
                <tbody>
                {items.map((x,i)=> (
                    <tr key={i} className="hover:bg-zinc-50">
                        <td className="p-2 break-all">{x.endpoint}</td>
                        <td className="p-2">{x.category}</td>
                        <td className="p-2">{x.seen_count}</td>
                        <td className="p-2 text-zinc-500">{x.last_seen}</td>
                    </tr>
                ))}
                </tbody>
            </table>
        </div>
    )
}

function SinksTab({ items }){
    return (
        <div className="rounded-2xl overflow-hidden border border-zinc-200 bg-white">
            <table className="w-full text-sm">
                <thead className="bg-zinc-50 text-zinc-600">
                <tr><th className="text-left p-2">Kind</th><th className="text-left p-2">Source</th><th className="text-left p-2">Func</th><th className="text-left p-2">Loc</th><th className="text-left p-2">Last</th></tr>
                </thead>
                <tbody>
                {items.map((x,i)=> (
                    <tr key={i} className="hover:bg-zinc-50">
                        <td className="p-2">{x.kind}</td>
                        <td className="p-2 break-all"><a href={x.source_url} target="_blank" rel="noreferrer" className="underline decoration-dotted">{x.source_url}</a></td>
                        <td className="p-2">{x.func}</td>
                        <td className="p-2">{x.line}:{x.col}</td>
                        <td className="p-2 text-zinc-500">{x.last_detected_at}</td>
                    </tr>
                ))}
                </tbody>
            </table>
        </div>
    )
}