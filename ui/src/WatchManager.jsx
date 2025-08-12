// src/componnets/WatchManager.jsx
import React, { useEffect, useState } from "react";

const API_BASE = import.meta.env.VITE_API_BASE || "";
async function req(path, opts={}) {
    const r = await fetch(API_BASE + path, opts);
    if (!r.ok) throw new Error((await r.text()) || `HTTP ${r.status}`);
    try { return await r.json(); } catch { return {}; }
}

const FREQS = [
    { label: "1h", val: 60 },
    { label: "6h", val: 360 },
    { label: "24h", val: 1440 },
    { label: "1w", val: 10080 },
];
const seenKey = (u) => `seen::${u}`;

export default function WatchManager({ siteId }) {
    const [list, setList] = useState([]);
    const [newUrl, setNewUrl] = useState("");
    const [freq, setFreq] = useState(1440);
    const [busy, setBusy] = useState(false);
    const [err, setErr] = useState("");

    const load = async () => {
        setErr("");
        try {
            const d = await req(`/api/watches?site_id=${encodeURIComponent(siteId)}`);
            setList(d.items || []);
        } catch(e) { setErr(e.message || "load error"); }
    };

    useEffect(()=>{ load(); }, [siteId]);

    const add = async (e) => {
        e.preventDefault();
        if (!newUrl.trim()) return;
        setBusy(true);
        try {
            await req("/api/watches/create", {
                method:"POST",
                headers:{ "Content-Type":"application/json"},
                body: JSON.stringify({ url: newUrl.trim(), freq_min: freq, enabled: true })
            });
            setNewUrl("");
            await load();
        } catch(e){ setErr(e.message || "add error"); }
        finally { setBusy(false); }
    };

    const del = async (url_norm) => {
        await req("/api/watches/delete", {
            method:"POST",
            headers:{ "Content-Type":"application/json"},
            body: JSON.stringify({ url_norm })
        });
        await load();
    };

    const scanNow = async (url_norm) => {
        await req("/api/watches/scan-now", {
            method:"POST",
            headers:{ "Content-Type":"application/json"},
            body: JSON.stringify({ url_norm })
        });
        await load();
    };

    const updateFreq = async (w, newFreq) => {
        // /api/watches/create Upsert می‌کند (site_id+url_norm یکتا)
        await req("/api/watches/create", {
            method:"POST",
            headers:{ "Content-Type":"application/json"},
            body: JSON.stringify({ url: w.url || w.url_norm, freq_min: newFreq, enabled: true })
        });
        await load();
    };

    const markSeen = (u) => {
        localStorage.setItem(seenKey(u), new Date().toISOString());
        setList([...list]);
    };

    const isRed = (w) => {
        if (!w?.last_change_at) return false;
        const lastSeen = localStorage.getItem(seenKey(w.url_norm));
        if (!lastSeen) return true;
        return new Date(w.last_change_at) > new Date(lastSeen);
    };

    return (
        <div className="space-y-3">
            <form onSubmit={add} className="flex gap-2">
                <input
                    className="flex-1 border border-zinc-300 rounded-xl px-3 py-2 text-sm"
                    placeholder={`https://${siteId}/path`}
                    value={newUrl}
                    onChange={e=>setNewUrl(e.target.value)}
                />
                <select
                    className="border border-zinc-300 rounded-xl px-2 py-2 text-sm"
                    value={freq}
                    onChange={e=>setFreq(+e.target.value)}
                >
                    {FREQS.map(f => <option key={f.val} value={f.val}>{f.label}</option>)}
                </select>
                <button className="px-3 py-2 rounded-lg bg-zinc-900 text-white text-sm" disabled={busy}>Add</button>
            </form>

            {err && <div className="text-red-600 text-sm">{err}</div>}

            <ul className="divide-y">
                {list.map(w => (
                    <li key={w.url_norm} className="py-2 flex items-center justify-between gap-2">
                        <div className="min-w-0 cursor-pointer" onClick={()=>markSeen(w.url_norm)}>
                            <div className="truncate text-sm font-medium flex items-center gap-2">
                                {isRed(w) && <span title="new changes" className="text-red-600">●</span>}
                                <span className="truncate">{w.url_norm}</span>
                            </div>
                            <div className="text-xs text-zinc-500">
                                freq: {w.freq_min}m • next: {w.next_run_at ? new Date(w.next_run_at).toLocaleString() : "-"}
                                {w.last_run_at && <> • last: {new Date(w.last_run_at).toLocaleString()}</>}
                            </div>
                        </div>
                        <div className="flex items-center gap-2 shrink-0">
                            <select
                                className="border border-zinc-300 rounded-xl px-2 py-1 text-xs"
                                value={w.freq_min || 1440}
                                onChange={e=>updateFreq(w, +e.target.value)}
                            >
                                {FREQS.map(f => <option key={f.val} value={f.val}>{f.label}</option>)}
                            </select>
                            <button className="px-2 py-1 rounded-md text-xs border border-zinc-300" onClick={()=>scanNow(w.url_norm)}>Scan</button>
                            <button className="px-2 py-1 rounded-md text-xs bg-red-600 text-white" onClick={()=>del(w.url_norm)}>Delete</button>
                        </div>
                    </li>
                ))}
                {!list.length && <li className="py-3 text-sm text-zinc-500">No watches</li>}
            </ul>
        </div>
    );
}
