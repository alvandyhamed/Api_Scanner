import React, { useEffect, useMemo, useState } from 'react'
import { Search, Globe, Link2, ChevronRight, ChevronDown, AlertTriangle, ShieldAlert, PlugZap, Eye, EyeOff, Timer } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import DomainScanner from "../componnets/DomainScanner.jsx";

const API_BASE = import.meta.env.VITE_API_BASE || "";
const AVATAR_SRC = '/avatar.jpg'

async function getJSON(path, signal){
    const res = await fetch(API_BASE + path, { signal })
    if(!res.ok) throw new Error(`${res.status}`)
    return res.json()
}
async function postJSON(path, body){
    const res = await fetch(API_BASE + path, {
        method: "POST",
        headers: { "Content-Type":"application/json" },
        body: JSON.stringify(body || {})
    })
    if(!res.ok) throw new Error(await res.text())
    try { return await res.json() } catch { return {} }
}

// ---- utils
function timeAgo(iso){
    try{
        const d=new Date(iso); const s=((Date.now()-d)/1000|0);
        if(s<60) return s+"s ago"; const m=s/60|0; if(m<60) return m+"m ago";
        const h=m/60|0; if(h<24) return h+"h ago"; return (h/24|0)+"d ago"
    }catch{return''}
}
function msToDHm(ms){
    if (ms <= 0) return "now"
    const s = Math.floor(ms/1000)
    const d = Math.floor(s/86400)
    const h = Math.floor((s%86400)/3600)
    const m = Math.floor((s%3600)/60)
    if (d > 0) return `${d}d ${h}h`
    if (h > 0) return `${h}h ${m}m`
    return `${m}m`
}
function timeUntil(iso){
    if(!iso) return ""
    const t = new Date(iso).getTime()
    return msToDHm(t - Date.now())
}
function StatusDot({ state }){
    const c= state==='danger'?'bg-red-500'
        : state==='new'   ?'bg-amber-400'
            : state==='both'  ?'bg-orange-500'
                :'bg-zinc-400';
    return <span className={`inline-block h-2.5 w-2.5 rounded-full ${c}`}/>
}
function safeHost(u){ try{return new URL(u).hostname}catch{return''} }
function safePath(u){ try{return new URL(u).pathname||'/'}catch{return'/'} }
function trimMid(s,n){ if(!s) return ''; if(s.length<=n) return s; const k=(n/2|0)-2; return s.slice(0,k)+'…'+s.slice(-k) }
function normalizeUrl(u){
    try{
        const x = new URL(u)
        // نرمال‌سازی اسلش آخر
        if(!x.pathname) x.pathname = '/'
        if(!x.pathname.endsWith('/')) {
            // فقط برای root اسلش بذاریم
            if (x.pathname === '') x.pathname = '/'
        }
        return x.toString()
    }catch{return u}
}

// local seen
const LS_SITES='sitechecker:lastSeen:sites';
function useSeen(){
    const [sitesSeen,setSitesSeen]=useState(()=>{try{return JSON.parse(localStorage.getItem(LS_SITES)||"{}")}catch{return{}}})
    return {
        sitesSeen,
        markSite:(id)=>setSitesSeen(p=>{const n={...p,[id]:Date.now()};localStorage.setItem(LS_SITES,JSON.stringify(n));return n}),
    }
}

export default function Home(){
    const [query,setQuery]=useState('')
    const [sites,setSites]=useState({loading:true,items:[]})
    const [expanded,setExpanded]=useState({})
    const [pagesBySite,setPagesBySite]=useState({})
    const [statusBySite,setStatusBySite]=useState({})
    const [selectedSite,setSelectedSite]=useState(null)
    const [alerts,setAlerts]=useState({sinks:[],assets:[],loading:false})
    // watches: { [siteId]: { loading, items, mapByUrlNorm } }
    const [watchesBySite, setWatchesBySite] = useState({})

    const nav=useNavigate();
    const {sitesSeen,markSite}=useSeen()

    // ---- load sites
    const fetchSites = async (signal) => {
        const d = await getJSON('/api/sites?limit=200', signal);
        setSites({loading:false, items: d.items || []});
    };
    useEffect(() => {
        const ac = new AbortController();
        fetchSites(ac.signal).catch(()=> setSites({loading:false, items:[]}));
        return () => ac.abort();
    }, []);
    const refreshSitesWithRetry = async () => {
        for (let i=0; i<5; i++) {
            try { await fetchSites(); break; } catch {}
            await new Promise(r => setTimeout(r, 1200));
        }
    };

    // ---- site open
    async function toggleSite(siteId){
        const open=!!expanded[siteId]; setExpanded(s=>({...s,[siteId]:!open}))
        if(!open && !pagesBySite[siteId]){
            setPagesBySite(p=>({...p,[siteId]:{loading:true,items:[]}}))
            try{
                const d=await getJSON(`/api/pages?site_id=${encodeURIComponent(siteId)}&limit=1000&sort=scanned_at&order=desc`);
                setPagesBySite(p=>({...p,[siteId]:{loading:false,items:d.items||[]}}))
            }catch{
                setPagesBySite(p=>({...p,[siteId]:{loading:false,items:[]}}))
            }
            await computeStatus(siteId);
            await loadWatchesMerged(siteId); // ← هردو حالت www و بدون www
            setSelectedSite(siteId);
            markSite(siteId);
            loadAlerts(siteId);
        }
    }

    async function computeStatus(siteId){
        try{
            const [sr,er]=await Promise.all([
                getJSON(`/api/sinks?site_id=${encodeURIComponent(siteId)}&limit=1&sort=last_detected_at&order=desc`),
                getJSON(`/api/endpoints?site_id=${encodeURIComponent(siteId)}&limit=1&sort=last_seen&order=desc`)
            ]);
            const sAt=sr?.items?.[0]?.last_detected_at;
            const aAt=er?.items?.[0]?.last_seen;
            const seen=sitesSeen[siteId]||0;
            const hasDanger=sAt && new Date(sAt).getTime()>seen;
            const hasNew=aAt && new Date(aAt).getTime()>seen;
            const state=hasDanger&&hasNew?'both': hasDanger?'danger': hasNew?'new':'none';
            setStatusBySite(s=>({...s,[siteId]:{state, sAt,aAt}}))
        }catch{
            setStatusBySite(s=>({...s,[siteId]:{state:'none'}}))
        }
    }

    async function loadAlerts(siteId){
        setAlerts(a=>({...a,loading:true}));
        try{
            const [sk,ep]=await Promise.all([
                getJSON(`/api/sinks?site_id=${encodeURIComponent(siteId)}&limit=10&sort=last_detected_at&order=desc`),
                getJSON(`/api/endpoints?site_id=${encodeURIComponent(siteId)}&limit=10&sort=last_seen&order=desc`)
            ]);
            setAlerts({loading:false,sinks:sk.items||[],assets:ep.items||[]})
        }catch{
            setAlerts({loading:false,sinks:[],assets:[]})
        }
    }

    // ---- watches
    function siteVariants(id){
        const set = new Set([id])
        if (id.startsWith('www.')) set.add(id.slice(4))
        else set.add('www.'+id)
        return Array.from(set)
    }
    async function loadWatchesMerged(siteId){
        setWatchesBySite(w=>({...w, [siteId]: {loading:true, items:[], mapByUrlNorm:{}}}))
        const vars = siteVariants(siteId)
        try{
            const all=[]
            for (const sid of vars){
                try{
                    const d = await getJSON(`/api/watches?site_id=${encodeURIComponent(sid)}`)
                    if (d.items) all.push(...d.items)
                }catch{ /* ignore */ }
            }
            // ساخت map بر اساس url_norm نرمال‌شده
            const map = {}
            for (const it of all){
                const key = normalizeUrl(it.url_norm || it.url)
                map[key] = it
            }
            setWatchesBySite(w=>({...w, [siteId]: {loading:false, items:all, mapByUrlNorm:map}}))
        }catch{
            setWatchesBySite(w=>({...w, [siteId]: {loading:false, items:[], mapByUrlNorm:{}}}))
        }
    }

    async function createWatch(url, freq_min=1440){
        await postJSON('/api/watches/create', { url, freq_min, enabled:true })
    }
    async function deleteWatch(url_norm){
        await postJSON('/api/watches/delete', { url_norm })
    }
    async function scanNowWatch(url_norm){
        await postJSON('/api/watches/scan-now', { url_norm })
    }
    async function changeWatchFreq(w, newFreq){
        await postJSON('/api/watches/create', { url: w.url || w.url_norm, freq_min:newFreq, enabled:true })
    }
    async function hasWatch(siteId, urlNorm) {
        const key = normalizeUrl(urlNorm);
        return !!watchesBySite[siteId]?.mapByUrlNorm?.[key];


    }
    async function toggleWatch(url_norm, siteId){
        const key = normalizeUrl(url_norm);

        const exists = hasWatch(siteId, key);
        if (exists){
            await deleteWatch(key)
            await loadWatchesMerged(siteId)
            return true
        }else {
            const input=document.getElementById(`watch-url-${siteId}`)
            if(input){
                input.value=key;
                input.focus();
                input.scrollIntoView({behavior:'smooth',block:'center'})
            }
            return false
        }

    }

    // ---- filter
    const filteredSites=useMemo(()=>{
        const q=query.trim().toLowerCase();
        if(!q) return sites.items;
        return (sites.items||[]).filter(s=>{
            const id=(s._id||'').toLowerCase();
            const hs=(s.hosts||[]).join(' ').toLowerCase();
            return id.includes(q) || hs.includes(q);
        })
    },[query,sites.items])

    function groupByHost(pages){
        const out={};
        for(const p of (pages||[])){
            const host=p.host||safeHost(p.url_norm||p.url);
            const path=p.path||safePath(p.url_norm||p.url);
            (out[host] ||= []).push({ path, url_norm: normalizeUrl(p.url_norm||p.url), scanned_at:p.scanned_at });
        }
        return Object.entries(out)
            .sort(([a],[b])=>a.localeCompare(b))
            .map(([host,arr])=>({host,pages:arr.sort((a,b)=>a.path.localeCompare(b.path))}))
    }

    return (
        <div className="mx-auto max-w-7xl w-full grow px-4 py-6 grid grid-cols-1 lg:grid-cols-12 gap-4">
            {/* ستون اصلی */}
            <section className="lg:col-span-8 space-y-3">
                {/* سرچ + آواتار */}
                <div className="flex items-center gap-3 mb-2">
                    <img src={AVATAR_SRC} alt="avatar" className="h-10 w-10 rounded-full object-cover ring-2 ring-zinc-200" />
                    <div className="grow">
                        <div className="flex items-center gap-2 rounded-2xl border border-zinc-300 bg-white px-3 py-2 shadow-sm">
                            <Search className="h-4 w-4 text-zinc-500" />
                            <input
                                value={query}
                                onChange={e=>setQuery(e.target.value)}
                                placeholder="جستجو بین سایت‌ها، ساب‌دامین‌ها و مسیرها…"
                                className="w-full bg-transparent outline-none text-sm"
                            />
                        </div>
                    </div>
                </div>

                {/* فرم اسکن دامنه */}
                <DomainScanner onScanned={refreshSitesWithRetry}/>

                {/* لیست سایت‌ها */}
                {sites.loading ? (
                    <div className="text-zinc-500 text-sm">Loading…</div>
                ) : (
                    filteredSites.map(s=>{
                        const siteId=s._id;
                        const open=!!expanded[siteId];
                        const stat=statusBySite[siteId]?.state||'none';
                        const ps=pagesBySite[siteId];
                        const grouped=open?groupByHost(ps?.items||[]):[];
                        const ws = watchesBySite[siteId] || {loading:false, items:[], mapByUrlNorm:{}};
                        const watchMap = ws.mapByUrlNorm || {};

                        return (
                            <div key={siteId} className="rounded-2xl bg-white border border-zinc-200 shadow-sm overflow-hidden">
                                <button className="w-full text-left px-3 py-2 flex items-center gap-2" onClick={()=>toggleSite(siteId)}>
                                    {open? <ChevronDown className="h-4 w-4 text-zinc-600"/> : <ChevronRight className="h-4 w-4 text-zinc-600"/>}
                                    <Globe className="h-4 w-4 text-zinc-700"/>
                                    <div className="min-w-0">
                                        <div className="truncate font-medium flex items-center gap-2">
                                            {siteId}
                                            {!!ws.items?.length && (
                                                <span className="text-xs px-2 py-0.5 rounded-full bg-zinc-100 text-zinc-700">
                          {ws.items.length} watched
                        </span>
                                            )}
                                        </div>
                                        <div className="truncate text-xs text-zinc-500">
                                            {s.hosts?.length? `${s.hosts.length} host • `:''}
                                            last {timeAgo(s.last_scan_at)}
                                        </div>
                                    </div>
                                    <span className="ml-auto"><StatusDot state={stat}/></span>
                                </button>

                                {open && (
                                    <div className="px-2 pb-4 space-y-4">
                                        {/* Watch Manager */}
                                        <div className="px-3 pt-3">
                                            <div className="text-sm font-medium mb-2">Watches</div>

                                            {/* فرم افزودن */}
                                            <div className="flex gap-2 items-center">
                                                <input
                                                    id={`watch-url-${siteId}`}
                                                    className="flex-1 border border-zinc-300 rounded-xl px-3 py-2 text-sm"
                                                    placeholder={`https://${siteId}/path`}
                                                />
                                                <select id={`watch-freq-${siteId}`} className="border border-zinc-300 rounded-xl px-2 py-2 text-sm" defaultValue="1440">
                                                    <option value="60">1h</option>
                                                    <option value="360">6h</option>
                                                    <option value="1440">24h</option>
                                                    <option value="10080">1w</option>
                                                </select>
                                                <button
                                                    className="px-3 py-2 rounded-lg bg-zinc-900 text-white text-sm"
                                                    onClick={async()=>{
                                                        const url = document.getElementById(`watch-url-${siteId}`).value.trim();
                                                        const freq = +document.getElementById(`watch-freq-${siteId}`).value;
                                                        if(!url) return;
                                                        await createWatch(url, freq);
                                                        document.getElementById(`watch-url-${siteId}`).value='';
                                                        await loadWatchesMerged(siteId);
                                                    }}
                                                >Add</button>
                                            </div>
                                        </div>

                                        {/* صفحات گروه‌بندی‌شده */}
                                        {ps?.loading? (
                                            <div className="px-4 py-3 text-sm text-zinc-500">Loading pages…</div>
                                        ) : (
                                            <div className="space-y-1">
                                                {grouped.map(({host,pages})=> (
                                                    <div key={host}>
                                                        <div className="px-3 py-2 text-sm text-zinc-700 flex items-center gap-2"><Link2 className="h-4 w-4"/>{host}</div>
                                                        <div className="mt-1 space-y-1">
                                                            {pages.map(p=> {
                                                                const key = normalizeUrl(p.url_norm)
                                                                const w   = watchMap[key]
                                                                const isWatched = !!w
                                                                const nextIn = isWatched ? timeUntil(w.next_run_at) : ""

                                                                return (
                                                                    <div
                                                                        key={p.url_norm}
                                                                        className="w-full px-3 py-2 rounded-xl bg-white hover:border-zinc-200 border border-transparent flex items-center justify-between gap-2"
                                                                        onDoubleClick={async()=>{ await toggleWatch(p.url_norm, siteId) }} // دابل‌کلیک → toggle
                                                                    >
                                                                        <button
                                                                            onClick={()=> nav(`/site/${encodeURIComponent(siteId)}?url=${encodeURIComponent(p.url_norm)}`)}
                                                                            className="text-left min-w-0"
                                                                        >
                                                                            <div className="truncate text-sm">{p.path}</div>
                                                                            <div className="truncate text-xs text-zinc-500">{p.url_norm}</div>
                                                                        </button>

                                                                        <div className="shrink-0 flex items-center gap-2">
                                                                            {isWatched ? (
                                                                                <>
      <span className="inline-flex items-center gap-1 text-xs text-zinc-600">
        <Timer className="h-3.5 w-3.5"/>{nextIn}
      </span>
                                                                                    <button
                                                                                        className="px-2 py-1 rounded-md text-xs border border-zinc-300 inline-flex items-center gap-1"
                                                                                        title="Unwatch"
                                                                                        onClick={() => toggleWatch(p.url_norm, siteId)}   // ← Unwatch واقعی
                                                                                    >
                                                                                        <EyeOff className="h-3.5 w-3.5"/> Unwatch
                                                                                    </button>
                                                                                </>
                                                                            ) : (
                                                                                <button
                                                                                    className="px-2 py-1 rounded-md text-xs border border-zinc-300 inline-flex items-center gap-1"
                                                                                    title="Watch (choose frequency above then Add)"
                                                                                    onClick={() => {
                                                                                        const input = document.getElementById(`watch-url-${siteId}`);
                                                                                        if (input) {
                                                                                            input.value = p.url_norm;                      // فقط پر کردن فرم
                                                                                            input.focus();
                                                                                            input.scrollIntoView({ behavior: 'smooth', block: 'center' });
                                                                                        }
                                                                                    }}
                                                                                >
                                                                                    <Eye className="h-3.5 w-3.5"/> Watch
                                                                                </button>
                                                                            )}
                                                                        </div>

                                                                    </div>
                                                                )
                                                            })}
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

            {/* سایدبار هشدارها */}
            <aside className="lg:col-span-4">
                <div className="rounded-2xl bg-white border border-zinc-200 shadow-sm p-4">
                    <div className="flex items-center gap-2 mb-2">
                        <AlertTriangle className="h-4 w-4 text-orange-500"/>
                        <h3 className="font-medium">Latest alerts {selectedSite?`for ${selectedSite}`:''}</h3>
                    </div>
                    {alerts.loading? (
                        <div className="text-sm text-zinc-500">Loading…</div>
                    ) : (
                        <div className="space-y-4">
                            <div>
                                <div className="text-xs font-semibold text-zinc-600 mb-1 flex items-center gap-1">
                                    <ShieldAlert className="h-4 w-4"/>Dangerous sinks
                                </div>
                                <ul className="space-y-1 text-sm">
                                    {(alerts.sinks||[]).map((s,i)=>(
                                        <li key={i} className="flex items-center justify-between gap-2">
                                            <span className="truncate">{s.kind} · {trimMid(s.source_url,36)}</span>
                                            <span className="text-xs text-zinc-500">{timeAgo(s.last_detected_at)}</span>
                                        </li>
                                    ))}
                                </ul>
                            </div>
                            <div>
                                <div className="text-xs font-semibold text-zinc-600 mb-1 flex items-center gap-1">
                                    <PlugZap className="h-4 w-4"/>New assets
                                </div>
                                <ul className="space-y-1 text-sm">
                                    {(alerts.assets||[]).map((e,i)=>(
                                        <li key={i} className="flex items-center justify-between gap-2">
                                            <span className="truncate">{e.category||'endpoint'} · {trimMid(e.endpoint,40)}</span>
                                            <span className="text-xs text-zinc-500">{timeAgo(e.last_seen)}</span>
                                        </li>
                                    ))}
                                </ul>
                            </div>
                        </div>
                    )}
                </div>
            </aside>
        </div>
    )
}
