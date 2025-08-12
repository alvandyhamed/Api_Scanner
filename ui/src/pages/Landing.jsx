import React, { useEffect, useMemo, useRef, useState } from 'react'
import { Search, Globe, Link2, ChevronRight, ChevronDown, AlertTriangle, ShieldAlert, PlugZap, Coffee } from 'lucide-react'
import { useNavigate } from 'react-router-dom'

const API_BASE = ''
const AVATAR_SRC = '/avatar.jpg'

async function getJSON(path, signal){
    const res = await fetch(API_BASE + path, { signal }); if(!res.ok) throw new Error(`${res.status}`); return res.json()
}
function timeAgo(iso){ try{const d=new Date(iso);const s=((Date.now()-d)/1000|0); if(s<60)return s+"s ago"; const m=s/60|0; if(m<60)return m+"m ago"; const h=m/60|0; if(h<24)return h+"h ago"; return (h/24|0)+"d ago"}catch{return''} }
function cls(...v){return v.filter(Boolean).join(' ')}
function StatusDot({ state }){ const c= state==='danger'?'bg-red-500': state==='new'?'bg-amber-400': state==='both'?'bg-orange-500':'bg-zinc-400'; return <span className={`inline-block h-2.5 w-2.5 rounded-full ${c}`}/> }

const LS_SITES='sitechecker:lastSeen:sites', LS_PAGES='sitechecker:lastSeen:pages'
function useSeen(){
    const [sitesSeen,setSitesSeen]=useState(()=>{try{return JSON.parse(localStorage.getItem(LS_SITES)||"{}")}catch{return{}}})
    const [pagesSeen,setPagesSeen]=useState(()=>{try{return JSON.parse(localStorage.getItem(LS_PAGES)||"{}")}catch{return{}}})
    return {
        sitesSeen, pagesSeen,
        markSite:(id)=>setSitesSeen(p=>{const n={...p,[id]:Date.now()};localStorage.setItem(LS_SITES,JSON.stringify(n));return n}),
        markPage:(u)=>setPagesSeen(p=>{const n={...p,[u]:Date.now()};localStorage.setItem(LS_PAGES,JSON.stringify(n));return n})
    }
}

export default function Landing(){
    const [query,setQuery]=useState('')
    const [sites,setSites]=useState({loading:true,items:[]})
    const [expanded,setExpanded]=useState({})
    const [pagesBySite,setPagesBySite]=useState({})
    const [statusBySite,setStatusBySite]=useState({})
    const [selectedSite,setSelectedSite]=useState(null)
    const [alerts,setAlerts]=useState({sinks:[],assets:[],loading:false})
    const abortRef=useRef()
    const nav=useNavigate();
    const {sitesSeen,pagesSeen,markSite,markPage}=useSeen()

    useEffect(()=>{ const ac=new AbortController(); abortRef.current=ac; (async()=>{ try{ const d=await getJSON('/api/sites?limit=200',ac.signal); setSites({loading:false,items:d.items||[]}) }catch(e){ if(e.name!=='AbortError') setSites({loading:false,items:[]}) } })(); return ()=>ac.abort() },[])

    async function toggleSite(siteId){
        const open=!!expanded[siteId]; setExpanded(s=>({...s,[siteId]:!open}))
        if(!open && !pagesBySite[siteId]){
            setPagesBySite(p=>({...p,[siteId]:{loading:true,items:[]}}))
            try{ const d=await getJSON(`/api/pages?site_id=${encodeURIComponent(siteId)}&limit=1000&sort=scanned_at&order=desc`); setPagesBySite(p=>({...p,[siteId]:{loading:false,items:d.items||[]}})) }catch{ setPagesBySite(p=>({...p,[siteId]:{loading:false,items:[]}})) }
            await computeStatus(siteId); setSelectedSite(siteId); markSite(siteId); loadAlerts(siteId)
        }
    }
    async function computeStatus(siteId){ try{ const [sr,er]=await Promise.all([ getJSON(`/api/sinks?site_id=${encodeURIComponent(siteId)}&limit=1&sort=last_detected_at&order=desc`), getJSON(`/api/endpoints?site_id=${encodeURIComponent(siteId)}&limit=1&sort=last_seen&order=desc`) ]); const sAt=sr?.items?.[0]?.last_detected_at; const aAt=er?.items?.[0]?.last_seen; const seen=sitesSeen[siteId]||0; const hasDanger=sAt && new Date(sAt).getTime()>seen; const hasNew=aAt && new Date(aAt).getTime()>seen; const state=hasDanger&&hasNew?'both': hasDanger?'danger': hasNew?'new':'none'; setStatusBySite(s=>({...s,[siteId]:{state, sAt,aAt}})) }catch{ setStatusBySite(s=>({...s,[siteId]:{state:'none'}})) } }
    async function loadAlerts(siteId){ setAlerts(a=>({...a,loading:true})); try{ const [sk,ep]=await Promise.all([ getJSON(`/api/sinks?site_id=${encodeURIComponent(siteId)}&limit=10&sort=last_detected_at&order=desc`), getJSON(`/api/endpoints?site_id=${encodeURIComponent(siteId)}&limit=10&sort=last_seen&order=desc`) ]); setAlerts({loading:false,sinks:sk.items||[],assets:ep.items||[]}) }catch{ setAlerts({loading:false,sinks:[],assets:[]}) } }

    const filteredSites=useMemo(()=>{ const q=query.trim().toLowerCase(); if(!q) return sites.items; return (sites.items||[]).filter(s=> (s._id||'').toLowerCase().includes(q) || (s.hosts||[]).join(' ').toLowerCase().includes(q)) },[query,sites.items])
    function groupByHost(pages){ const out={}; for(const p of (pages||[])){ const host=p.host||safeHost(p.url_norm||p.url); const path=p.path||safePath(p.url_norm||p.url); (out[host] ||= []).push({ path, url_norm:p.url_norm||p.url, scanned_at:p.scanned_at }); } return Object.entries(out).sort(([a],[b])=>a.localeCompare(b)).map(([host,arr])=>({host,pages:arr.sort((a,b)=>a.path.localeCompare(b.path))})) }

    return (
        <div className="mx-auto max-w-7xl w-full grow px-4 py-6 grid grid-cols-1 lg:grid-cols-12 gap-4">
            <section className="lg:col-span-8 space-y-3">
                <div className="flex items-center gap-3 mb-2">
                    <img src={AVATAR_SRC} alt="avatar" className="h-10 w-10 rounded-full object-cover ring-2 ring-zinc-200" />
                    <div className="grow">
                        <div className="flex items-center gap-2 rounded-2xl border border-zinc-300 bg-white px-3 py-2 shadow-sm">
                            <Search className="h-4 w-4 text-zinc-500" />
                            <input value={query} onChange={e=>setQuery(e.target.value)} placeholder="جستجو بین سایت‌ها، ساب‌دامین‌ها و مسیرها…" className="w-full bg-transparent outline-none text-sm" />
                        </div>
                    </div>
                </div>

                {sites.loading ? <div className="text-zinc-500 text-sm">Loading…</div> : (
                    filteredSites.map(s=>{
                        const siteId=s._id; const open=!!expanded[siteId]; const stat=statusBySite[siteId]?.state||'none'; const ps=pagesBySite[siteId]; const grouped=open?groupByHost(ps?.items||[]):[];
                        return (
                            <div key={siteId} className="rounded-2xl bg-white border border-zinc-200 shadow-sm overflow-hidden">
                                <button className="w-full text-left px-3 py-2 flex items-center gap-2" onClick={()=>toggleSite(siteId)}>
                                    {open? <ChevronDown className="h-4 w-4 text-zinc-600"/> : <ChevronRight className="h-4 w-4 text-zinc-600"/>}
                                    <Globe className="h-4 w-4 text-zinc-700"/>
                                    <div className="min-w-0">
                                        <div className="truncate font-medium">{siteId}</div>
                                        <div className="truncate text-xs text-zinc-500">{s.hosts?.length? `${s.hosts.length} host • `:''}last {timeAgo(s.last_scan_at)}</div>
                                    </div>
                                    <span className="ml-auto"><StatusDot state={stat}/></span>
                                </button>
                                {open && (
                                    <div className="px-2 pb-3">
                                        {ps?.loading? <div className="px-4 py-3 text-sm text-zinc-500">Loading pages…</div> : (
                                            <div className="space-y-1">
                                                {grouped.map(({host,pages})=> (
                                                    <div key={host}>
                                                        <div className="px-3 py-2 text-sm text-zinc-700 flex items-center gap-2"><Link2 className="h-4 w-4"/>{host}</div>
                                                        <div className="mt-1 space-y-1">
                                                            {pages.map(p=> (
                                                                <button key={p.url_norm} onClick={()=> nav(`/site/${encodeURIComponent(siteId)}?url=${encodeURIComponent(p.url_norm)}`)} className="w-full text-left px-3 py-2 rounded-xl bg-white hover:border-zinc-200 border border-transparent">
                                                                    <div className="truncate text-sm">{p.path}</div>
                                                                    <div className="truncate text-xs text-zinc-500">{p.url_norm}</div>
                                                                </button>
                                                            ))}
                                                        </div>
                                                    </div>
                                                ))}
                                            </div>
                                        )}
                                    </div>
                                )}
                            </div>
                        )
                    })
                )}
            </section>

            <aside className="lg:col-span-4">
                <div className="rounded-2xl bg-white border border-zinc-200 shadow-sm p-4">
                    <div className="flex items-center gap-2 mb-2"><AlertTriangle className="h-4 w-4 text-orange-500"/><h3 className="font-medium">Latest alerts {selectedSite?`for ${selectedSite}`:''}</h3></div>
                    {alerts.loading? <div className="text-sm text-zinc-500">Loading…</div> : (
                        <div className="space-y-4">
                            <div>
                                <div className="text-xs font-semibold text-zinc-600 mb-1 flex items-center gap-1"><ShieldAlert className="h-4 w-4"/>Dangerous sinks</div>
                                <ul className="space-y-1 text-sm">
                                    {(alerts.sinks||[]).map((s,i)=>(<li key={i} className="flex items-center justify-between gap-2"><span className="truncate">{s.kind} · {trimMid(s.source_url,36)}</span><span className="text-xs text-zinc-500">{timeAgo(s.last_detected_at)}</span></li>))}
                                </ul>
                            </div>
                            <div>
                                <div className="text-xs font-semibold text-zinc-600 mb-1 flex items-center gap-1"><PlugZap className="h-4 w-4"/>New assets</div>
                                <ul className="space-y-1 text-sm">
                                    {(alerts.assets||[]).map((e,i)=>(<li key={i} className="flex items-center justify-between gap-2"><span className="truncate">{e.category||'endpoint'} · {trimMid(e.endpoint,40)}</span><span className="text-xs text-zinc-500">{timeAgo(e.last_seen)}</span></li>))}
                                </ul>
                            </div>
                        </div>
                    )}
                </div>
            </aside>
        </div>
    )
}
function safeHost(u){ try{return new URL(u).hostname}catch{return''} }
function safePath(u){ try{return new URL(u).pathname||'/'}catch{return'/'} }
function trimMid(s,n){ if(!s) return ''; if(s.length<=n) return s; const k=(n/2|0)-2; return s.slice(0,k)+'…'+s.slice(-k) }