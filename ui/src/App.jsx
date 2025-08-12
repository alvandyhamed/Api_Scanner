import React, { useEffect, useMemo, useRef, useState } from "react";
import { Search, Globe, Link2, ChevronRight, ChevronDown, AlertTriangle, ShieldAlert, PlugZap, ExternalLink, Coffee } from "lucide-react";

/**
 * Landing page for SiteChecker UI
 * - Header with avatar + search box
 * - Left: collapsible tree (site -> host -> paths)
 * - Status bullet per site: red (danger sinks), yellow (new assets), orange (both)
 * - Right: Alerts panel (latest sinks & new assets)
 * - Footer with Buy me a coffee
 *
 * API endpoints used (same-origin via nginx proxy /api):
 *   GET /api/sites
 *   GET /api/pages?site_id=...&limit=1000&sort=scanned_at&order=desc
 *   GET /api/sinks?site_id=...&limit=1&sort=last_detected_at&order=desc
 *   GET /api/endpoints?site_id=...&limit=1&sort=last_seen&order=desc
 */

// If your UI is same-origin with API, leave empty string
const API_BASE = "";
const AVATAR_SRC = "/avatar.jpeg"; // put your avatar in ui/public/avatar.jpg

async function getJSON(path, signal) {
    const res = await fetch(API_BASE + path, { signal });
    if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
    return res.json();
}

const LS_SITES = "sitechecker:lastSeen:sites";
const LS_PAGES = "sitechecker:lastSeen:pages";

function useSeenStore() {
    const [sitesSeen, setSitesSeen] = useState(() => {
        try { return JSON.parse(localStorage.getItem(LS_SITES) || "{}"); } catch { return {}; }
    });
    const [pagesSeen, setPagesSeen] = useState(() => {
        try { return JSON.parse(localStorage.getItem(LS_PAGES) || "{}"); } catch { return {}; }
    });
    const markSiteSeen = (siteId) => setSitesSeen(prev => {
        const n = { ...prev, [siteId]: Date.now() };
        localStorage.setItem(LS_SITES, JSON.stringify(n));
        return n;
    });
    const markPageSeen = (urlNorm) => setPagesSeen(prev => {
        const n = { ...prev, [urlNorm]: Date.now() };
        localStorage.setItem(LS_PAGES, JSON.stringify(n));
        return n;
    });
    return { sitesSeen, pagesSeen, markSiteSeen, markPageSeen };
}

function timeAgo(iso) {
    try {
        const d = new Date(iso);
        const s = Math.floor((Date.now() - d.getTime()) / 1000);
        if (s < 60) return `${s}s ago`;
        const m = Math.floor(s / 60); if (m < 60) return `${m}m ago`;
        const h = Math.floor(m / 60); if (h < 24) return `${h}h ago`;
        const d2 = Math.floor(h / 24); return `${d2}d ago`;
    } catch { return ""; }
}

function cls(...v){ return v.filter(Boolean).join(" "); }

function StatusDot({ state }){
    // state: "none" | "danger" | "new" | "both"
    const color = state === "danger" ? "bg-red-500" : state === "new" ? "bg-amber-400" : state === "both" ? "bg-orange-500" : "bg-zinc-400";
    return <span className={cls("inline-block h-2.5 w-2.5 rounded-full", color)} />;
}

function Row({ depth=0, title, subtitle, leftIcon, chevron, right, onClick, active }){
    return (
        <button
            onClick={onClick}
            className={cls(
                "w-full text-left px-3 py-2 rounded-xl border transition bg-white hover:border-zinc-200",
                active ? "ring-1 ring-zinc-200" : "border-transparent"
            )}
            style={{ paddingLeft: depth*16 + 12 }}
        >
            <div className="flex items-center gap-2">
                {chevron}
                {leftIcon}
                <div className="min-w-0">
                    <div className="truncate text-sm font-medium text-zinc-800">{title}</div>
                    {subtitle && <div className="truncate text-xs text-zinc-500">{subtitle}</div>}
                </div>
                <div className="ml-auto flex items-center gap-2">{right}</div>
            </div>
        </button>
    );
}

export default function App(){
    const [query, setQuery] = useState("");
    const [sites, setSites] = useState({ loading: true, items: [] });
    const [expanded, setExpanded] = useState({}); // {siteId: true}
    const [pagesBySite, setPagesBySite] = useState({}); // {siteId: {loading, items: []}}
    const [statusBySite, setStatusBySite] = useState({}); // {siteId: {state, latestSinkAt, latestAssetAt}}
    const [selectedSite, setSelectedSite] = useState(null);
    const [alerts, setAlerts] = useState({ sinks: [], assets: [], loading: false });
    const abortRef = useRef();
    const { sitesSeen, pagesSeen, markSiteSeen, markPageSeen } = useSeenStore();

    // Load sites
    useEffect(() => {
        const ac = new AbortController(); abortRef.current = ac;
        (async () => {
            try {
                const data = await getJSON(`/api/sites?limit=200`, ac.signal);
                setSites({ loading: false, items: data.items || [] });
            } catch (e){ if (e.name !== "AbortError") setSites({ loading:false, items: [] }); }
        })();
        return () => ac.abort();
    }, []);

    // When a site expands for first time: load its pages + compute status
    async function toggleSite(siteId){
        const isOpen = !!expanded[siteId];
        setExpanded(s => ({ ...s, [siteId]: !isOpen }));
        if (!isOpen && !pagesBySite[siteId]){
            setPagesBySite(p => ({ ...p, [siteId]: { loading: true, items: [] } }));
            try {
                const data = await getJSON(`/api/pages?site_id=${encodeURIComponent(siteId)}&limit=1000&sort=scanned_at&order=desc`);
                setPagesBySite(p => ({ ...p, [siteId]: { loading: false, items: data.items || [] } }));
            } catch(e){
                setPagesBySite(p => ({ ...p, [siteId]: { loading: false, items: [] } }));
            }
            // compute status
            await computeSiteStatus(siteId);
            setSelectedSite(siteId);
            markSiteSeen(siteId);
            loadAlerts(siteId);
        }
    }

    async function computeSiteStatus(siteId){
        try {
            const [sinkRes, assetRes] = await Promise.all([
                getJSON(`/api/sinks?site_id=${encodeURIComponent(siteId)}&limit=1&sort=last_detected_at&order=desc`),
                getJSON(`/api/endpoints?site_id=${encodeURIComponent(siteId)}&limit=1&sort=last_seen&order=desc`),
            ]);
            const latestSinkAt = sinkRes?.items?.[0]?.last_detected_at || null;
            const latestAssetAt = assetRes?.items?.[0]?.last_seen || null;
            const seenAt = sitesSeen[siteId] || 0;
            const hasDanger = latestSinkAt && new Date(latestSinkAt).getTime() > seenAt;
            const hasNewAsset = latestAssetAt && new Date(latestAssetAt).getTime() > seenAt;
            const state = hasDanger && hasNewAsset ? "both" : hasDanger ? "danger" : hasNewAsset ? "new" : "none";
            setStatusBySite(s => ({ ...s, [siteId]: { state, latestSinkAt, latestAssetAt } }));
        } catch(e){
            setStatusBySite(s => ({ ...s, [siteId]: { state: "none" } }));
        }
    }

    async function loadAlerts(siteId){
        setAlerts(a => ({ ...a, loading: true }));
        try {
            const [sinksRes, epsRes] = await Promise.all([
                getJSON(`/api/sinks?site_id=${encodeURIComponent(siteId)}&limit=10&sort=last_detected_at&order=desc`),
                getJSON(`/api/endpoints?site_id=${encodeURIComponent(siteId)}&limit=10&sort=last_seen&order=desc`),
            ]);
            setAlerts({ loading: false, sinks: sinksRes.items || [], assets: epsRes.items || [] });
        } catch(e){ setAlerts({ loading:false, sinks: [], assets: [] }); }
    }

    const filteredSites = useMemo(() => {
        const q = query.trim().toLowerCase();
        if (!q) return sites.items;
        return (sites.items||[]).filter(s => {
            const sid = (s._id||"").toLowerCase();
            const hosts = (s.hosts||[]).join(" ").toLowerCase();
            return sid.includes(q) || hosts.includes(q);
        })
    }, [query, sites.items]);

    function groupPagesByHost(pages){
        const out = {};
        for (const p of (pages||[])){
            const host = p.host || safeHost(p.url_norm || p.url);
            const path = p.path || safePath(p.url_norm || p.url);
            (out[host] ||= []).push({ path, url_norm: p.url_norm || p.url, scanned_at: p.scanned_at });
        }
        return Object.entries(out).sort(([a],[b]) => a.localeCompare(b)).map(([host, arr]) => ({ host, pages: arr.sort((a,b)=> a.path.localeCompare(b.path)) }));
    }

    function onOpenDetails(payload){
        // Placeholder for router navigation
        console.log("navigate →", payload);
    }

    return (
        <div className="min-h-screen bg-zinc-50 flex flex-col">
            {/* Header */}
            <header className="sticky top-0 z-10 bg-white/80 backdrop-blur border-b border-zinc-200">
                <div className="mx-auto max-w-7xl px-4 py-3 flex items-center gap-4">
                    <img src={AVATAR_SRC} alt="avatar" className="h-10 w-10 rounded-full object-cover ring-2 ring-zinc-200" />
                    <div className="grow">
                        <div className="flex items-center gap-2 rounded-2xl border border-zinc-300 bg-white px-3 py-2 shadow-sm">
                            <Search className="h-4 w-4 text-zinc-500" />
                            <input
                                value={query}
                                onChange={(e)=>setQuery(e.target.value)}
                                placeholder="جستجو بین سایت‌ها، ساب‌دامین‌ها و مسیرها…"
                                className="w-full bg-transparent outline-none text-sm"
                            />
                        </div>
                    </div>
                </div>
            </header>

            {/* Body grid */}
            <main className="mx-auto max-w-7xl w-full grow px-4 py-6 grid grid-cols-1 lg:grid-cols-12 gap-4">
                {/* Left: tree */}
                <section className="lg:col-span-8 space-y-3">
                    <h2 className="text-sm font-medium text-zinc-600">Latest scanned sites</h2>
                    {sites.loading ? (
                        <div className="text-zinc-500 text-sm">Loading…</div>
                    ) : filteredSites.length === 0 ? (
                        <div className="text-zinc-500 text-sm">No sites yet. Run your first scan.</div>
                    ) : (
                        filteredSites.map((s) => {
                            const siteId = s._id;
                            const open = !!expanded[siteId];
                            const stat = statusBySite[siteId]?.state || "none";
                            const pagesState = pagesBySite[siteId];
                            const grouped = open ? groupPagesByHost(pagesState?.items||[]) : [];
                            return (
                                <div key={siteId} className="rounded-2xl bg-white border border-zinc-200 shadow-sm overflow-hidden">
                                    <Row
                                        title={siteId}
                                        subtitle={s.hosts?.length ? `${s.hosts.length} host • last ${timeAgo(s.last_scan_at)}` : `last ${timeAgo(s.last_scan_at)}`}
                                        chevron={open ? <ChevronDown className="h-4 w-4 text-zinc-600"/> : <ChevronRight className="h-4 w-4 text-zinc-600"/>}
                                        leftIcon={<Globe className="h-4 w-4 text-zinc-700"/>}
                                        right={<StatusDot state={stat} />}
                                        onClick={()=> toggleSite(siteId)}
                                    />

                                    {open && (
                                        <div className="px-2 pb-3">
                                            {pagesState?.loading ? (
                                                <div className="px-4 py-3 text-sm text-zinc-500">Loading pages…</div>
                                            ) : grouped.length === 0 ? (
                                                <div className="px-4 py-3 text-sm text-zinc-500">No pages stored.</div>
                                            ) : (
                                                <div className="space-y-1">
                                                    {grouped.map(({ host, pages }) => (
                                                        <div key={host}>
                                                            <Row
                                                                depth={1}
                                                                title={host}
                                                                leftIcon={<Link2 className="h-4 w-4 text-zinc-600"/>}
                                                                onClick={()=>{ setSelectedSite(siteId); loadAlerts(siteId); }}
                                                            />
                                                            <div className="mt-1 space-y-1">
                                                                {pages.map(p => {
                                                                    const isNew = p.scanned_at && (!pagesSeen[p.url_norm] || new Date(p.scanned_at).getTime() > pagesSeen[p.url_norm]);
                                                                    return (
                                                                        <Row
                                                                            key={p.url_norm}
                                                                            depth={2}
                                                                            title={p.path}
                                                                            subtitle={p.url_norm}
                                                                            leftIcon={<span className="h-2.5 w-2.5 rounded-full bg-zinc-300"/>}
                                                                            right={<StatusDot state={isNew?"new":"none"} />}
                                                                            onClick={()=>{ markPageSeen(p.url_norm); onOpenDetails({ siteId, url: p.url_norm }); }}
                                                                        />
                                                                    );
                                                                })}
                                                            </div>
                                                        </div>
                                                    ))}
                                                </div>
                                            )}
                                        </div>
                                    )}
                                </div>
                            );
                        })
                    )}
                </section>

                {/* Right: alerts panel */}
                <aside className="lg:col-span-4">
                    <div className="rounded-2xl bg-white border border-zinc-200 shadow-sm p-4">
                        <div className="flex items-center gap-2 mb-2">
                            <AlertTriangle className="h-4 w-4 text-orange-500"/>
                            <h3 className="font-medium text-zinc-800">Latest alerts {selectedSite ? `for ${selectedSite}` : "(select a site)"}</h3>
                        </div>
                        {alerts.loading ? (
                            <div className="text-sm text-zinc-500">Loading…</div>
                        ) : selectedSite ? (
                            <div className="space-y-4">
                                <div>
                                    <div className="text-xs font-semibold text-zinc-600 mb-1 flex items-center gap-1"><ShieldAlert className="h-4 w-4"/>Dangerous sinks</div>
                                    {alerts.sinks.length === 0 ? (
                                        <div className="text-xs text-zinc-500">No recent sinks.</div>
                                    ) : (
                                        <ul className="space-y-1 text-sm">
                                            {alerts.sinks.map((s,i) => (
                                                <li key={i} className="flex items-center justify-between gap-2">
                                                    <span className="truncate" title={`${s.kind} @ ${s.source_url}`}>{s.kind} · <a className="underline decoration-dotted" href={s.source_url} target="_blank" rel="noreferrer">{trimMid(s.source_url, 36)}</a></span>
                                                    <span className="text-xs text-zinc-500">{timeAgo(s.last_detected_at)}</span>
                                                </li>
                                            ))}
                                        </ul>
                                    )}
                                </div>
                                <div>
                                    <div className="text-xs font-semibold text-zinc-600 mb-1 flex items-center gap-1"><PlugZap className="h-4 w-4"/>New assets</div>
                                    {alerts.assets.length === 0 ? (
                                        <div className="text-xs text-zinc-500">No recent assets.</div>
                                    ) : (
                                        <ul className="space-y-1 text-sm">
                                            {alerts.assets.map((e,i) => (
                                                <li key={i} className="flex items-center justify-between gap-2">
                                                    <span className="truncate" title={e.endpoint}>{e.category || "endpoint"} · {trimMid(e.endpoint, 40)}</span>
                                                    <span className="text-xs text-zinc-500">{timeAgo(e.last_seen)}</span>
                                                </li>
                                            ))}
                                        </ul>
                                    )}
                                </div>
                            </div>
                        ) : (
                            <div className="text-sm text-zinc-500">Choose a site from the list to see alerts.</div>
                        )}
                    </div>
                </aside>
            </main>

            {/* Footer */}
            <footer className="w-full border-t border-zinc-200 bg-white">
                <div className="mx-auto max-w-7xl px-4 py-3 flex items-center">
                    <div className="flex items-center gap-2">
                        <span className="h-3 w-3 rounded-full bg-zinc-300"></span>
                        <span className="h-3 w-3 rounded-full bg-zinc-300"></span>
                        <span className="h-3 w-3 rounded-full bg-zinc-300"></span>
                    </div>
                    <div className="mx-auto text-xs text-zinc-500">power by Hamed0x</div>
                    <a
                        href="https://www.buymeacoffee.com/hamed0x"
                        target="_blank" rel="noreferrer"
                        className="inline-flex items-center gap-2 rounded-full border border-amber-400 bg-amber-50 px-3 py-1 text-sm text-amber-700 hover:bg-amber-100"
                    >
                        <Coffee className="h-4 w-4"/> Buy me a coffee
                    </a>
                </div>
            </footer>
        </div>
    );
}

function safeHost(u){ try { return new URL(u).hostname; } catch { return ""; } }
function safePath(u){ try { return new URL(u).pathname || "/"; } catch { return "/"; } }
function trimMid(s, n){ if(!s) return ""; if(s.length<=n) return s; const k=Math.floor(n/2)-2; return s.slice(0,k)+"…"+s.slice(-k); }
