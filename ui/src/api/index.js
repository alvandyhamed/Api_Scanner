const API_BASE = import.meta.env.VITE_API_BASE || "";

async function req(path, opts = {}) {
    const res = await fetch(API_BASE + path, opts);
    if (!res.ok) throw new Error((await res.text()) || `HTTP ${res.status}`);
    try { return await res.json(); } catch { return {}; }
}

export const api = {
    // basics
    health: () => req("/api/health"),
    sites:  (limit = 200) => req(`/api/sites?limit=${limit}`),

    // scan now (one-off)
    scan: (payload) => req("/api/scan", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
    }),

    // watches
    watchesList: (siteId) => req(`/api/watches?site_id=${encodeURIComponent(siteId)}`),
    watchCreate: ({ url, freq_min = 1440, enabled = true }) =>
        req("/api/watches/create", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ url, freq_min, enabled }),
        }),
    watchDelete: (url_norm) =>
        req("/api/watches/delete", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ url_norm }),
        }),
    watchScanNow: (url_norm) =>
        req("/api/watches/scan-now", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ url_norm }),
        }),

    // discord settings
    discordGet: () => req("/api/settings/discord"),
    discordSet: (webhook_url, enabled) =>
        req("/api/settings/discord/set", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ webhook_url, enabled }),
        }),
    discordTest: () => req("/api/settings/discord/test", { method: "POST" }),
};
