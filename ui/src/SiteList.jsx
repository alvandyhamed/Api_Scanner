import { useEffect, useState } from "react";
import { api } from "./api";
import WatchManager from "./WatchManager";

export default function SiteList({ reloadKey = 0 }) {
    const [sites, setSites] = useState([]);
    const [open, setOpen] = useState({}); // site_id -> bool
    const [err, setErr] = useState("");

    const load = async () => {
        setErr("");
        try {
            const d = await api.sites(200);
            setSites(d.items || []);
        } catch (e) {
            setErr(e.message || "load error");
        }
    };

    useEffect(() => { load(); }, [reloadKey]);

    return (
        <div className="space-y-3">
            {err && <div className="text-red-600 text-sm">{err}</div>}
            {sites.map(s => (
                <div key={s.site_id} className="card">
                    <div className="flex items-center justify-between cursor-pointer"
                         onClick={()=>setOpen(o=>({...o, [s.site_id]: !o[s.site_id]}))}>
                        <div>
                            <div className="font-semibold">{s.site_id}</div>
                            <div className="text-xs text-gray-500">
                                hosts: {(s.hosts||[]).length} • last scan: {s.last_scan_at ? new Date(s.last_scan_at).toLocaleString() : "-"}
                            </div>
                        </div>
                        <div>{open[s.site_id] ? "▾" : "▸"}</div>
                    </div>
                    {open[s.site_id] && (
                        <div className="mt-3">
                            <WatchManager siteId={s.site_id} />
                        </div>
                    )}
                </div>
            ))}
            {!sites.length && <div className="text-sm text-gray-500">No sites yet</div>}
        </div>
    );
}
