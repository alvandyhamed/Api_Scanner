import React, { useEffect, useState } from "react";
const API_BASE = import.meta.env.VITE_API_BASE || "";
async function req(path, opts={}) {
    const r = await fetch(API_BASE + path, opts);
    if(!r.ok) throw new Error((await r.text()) || `HTTP ${r.status}`);
    try { return await r.json(); } catch { return {}; }
}

export default function Settings(){
    const [enabled, setEnabled] = useState(false);
    const [webhook, setWebhook] = useState("");
    const [masked, setMasked] = useState("");
    const [msg, setMsg] = useState("");
    const [err, setErr] = useState("");

    const load = async () => {
        setErr(""); setMsg("");
        try {
            const d = await req("/api/settings/discord");
            setEnabled(!!d.enabled);
            setMasked(d.webhook_masked || "");
        } catch(e){ setErr(e.message || "load error"); }
    };
    useEffect(()=>{ load(); }, []);

    const save = async () => {
        setErr(""); setMsg("");
        try {
            await req("/api/settings/discord/set", {
                method:"POST", headers:{ "Content-Type":"application/json" },
                body: JSON.stringify({ webhook_url: webhook, enabled })
            });
            setWebhook("");
            setMsg("Saved ✅"); await load();
        } catch(e){ setErr(e.message || "save error"); }
    };
    const test = async () => {
        setErr(""); setMsg("");
        try { await req("/api/settings/discord/test", { method:"POST" }); setMsg("Test sent ✅"); }
        catch(e){ setErr(e.message || "test error"); }
    };

    return (
        <div className="mx-auto max-w-3xl px-4 py-6 space-y-4">
            <h1 className="text-xl font-semibold">Discord settings</h1>
            <div className="rounded-2xl bg-white border border-zinc-200 shadow-sm p-4 space-y-3">
                <div className="text-sm text-zinc-600">Current: {masked || "—"}</div>
                <label className="flex items-center gap-2 text-sm">
                    <input type="checkbox" checked={enabled} onChange={e=>setEnabled(e.target.checked)} />
                    <span>Enable Discord notifications</span>
                </label>
                <input className="w-full border border-zinc-300 rounded-xl px-3 py-2 text-sm"
                       placeholder="Discord Webhook URL"
                       value={webhook} onChange={e=>setWebhook(e.target.value)} />
                <div className="flex items-center gap-2">
                    <button className="px-3 py-2 rounded-lg bg-zinc-900 text-white text-sm" onClick={save}>Save</button>
                    <button className="px-3 py-2 rounded-lg border border-zinc-300 text-sm" onClick={test}>Send test</button>
                    {msg && <span className="text-green-700 text-sm">{msg}</span>}
                    {err && <span className="text-red-600 text-sm">{err}</span>}
                </div>
            </div>
        </div>
    );
}
