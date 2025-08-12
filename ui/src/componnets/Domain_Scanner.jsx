import React, { useState, useRef } from "react";

// Drop this component anywhere in your app (plain JS version)
// Adjust SCAN_API to your backend endpoint (e.g., "/scan" for Go service)
const API_BASE = import.meta.env.VITE_API_BASE || "";
const SCAN_API = `${API_BASE}/api/scan`;

export default function DomainScanner() {
    const [domain, setDomain] = useState("");
    const [status, setStatus] = useState("idle"); // idle | loading | success | error
    const [message, setMessage] = useState("");
    const inputRef = useRef(null);

    const isValidDomain = (v) => {
        // Accepts domain or URL. Quick sanity check; tweak to your needs
        try {
            // If user typed bare domain, URL() will fail — prepend scheme
            const url = new URL(/^https?:\/\//i.test(v) ? v : `http://${v}`);
            return !!url.hostname && /[.]/.test(url.hostname);
        } catch {
            return false;
        }
    };

    const handleAdd = async () => {
        if (!domain.trim()) {
            setStatus("error");
            setMessage("لطفاً دامنه را وارد کنید.");
            inputRef.current?.focus();
            return;
        }
        if (!isValidDomain(domain.trim())) {
            setStatus("error");
            setMessage("فرمت دامنه/آدرس معتبر نیست.");
            inputRef.current?.focus();
            return;
        }

        setStatus("loading");
        setMessage("");

        try {
            const body = { url: domain.trim() };
            console.log(SCAN_API)
            const res = await fetch(SCAN_API, {

                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(body),
            });

            if (!res.ok) {
                const txt = await res.text();
                console.log(res)
                throw new Error(txt || `HTTP ${res.status}`);
            }

            // server response (adjust to your API)
            const data = await res.json().catch(() => ({}));
            setStatus("success");
            setMessage(data?.message || "اسکن با موفقیت شروع شد.");
            setDomain("");
        } catch (err) {
            setStatus("error");
            setMessage(err?.message || "خطا در شروع اسکن");
        }
    };

    const onKey = (e) => {
        if (e.key === "Enter") handleAdd();
    };

    return (
        <div style={styles.wrap}>
            <div style={styles.row}>
                <div style={styles.inputWrap}>
                    <span style={styles.searchIcon} aria-hidden>🔎</span>
                    <input
                        ref={inputRef}
                        type="text"
                        placeholder="دامنه یا URL را وارد کنید"
                        value={domain}
                        onChange={(e) => setDomain(e.target.value)}
                        onKeyDown={onKey}
                        style={styles.input}
                        disabled={status === "loading"}
                    />
                </div>
                <button
                    onClick={handleAdd}
                    disabled={status === "loading"}
                    style={{
                        ...styles.button,
                        ...(status === "loading" ? styles.buttonDisabled : {}),
                    }}
                >
                    {status === "loading" ? "در حال افزودن…" : "افزودن"}
                </button>
            </div>

            {status === "loading" && (
                <div style={styles.progressBar}>
                    <div style={styles.progressIndeterminate} />
                </div>
            )}

            {status === "success" && (
                <div style={{ ...styles.note, color: "#0a7" }}>{message}</div>
            )}
            {status === "error" && (
                <div style={{ ...styles.note, color: "#c33" }}>{message}</div>
            )}
        </div>
    );
}

const styles = {
    wrap: { display: "grid", gap: 10, maxWidth: 720, margin: "12px auto" },
    row: { display: "grid", gridTemplateColumns: "1fr 120px", gap: 8 },
    inputWrap: {
        position: "relative",
        display: "flex",
        alignItems: "center",
        border: "1px solid #e1e5ea",
        borderRadius: 12,
        padding: "10px 12px 10px 36px",
        boxShadow: "inset 0 1px 2px rgba(0,0,0,03)",
        background: "#fff",
    },
    searchIcon: {
        position: "absolute",
        left: 10,
        fontSize: 14,
        opacity: 0.7,
        pointerEvents: "none",
    },
    input: {
        width: "100%",
        border: "none",
        outline: "none",
        fontSize: 14,
        background: "transparent",
    },
    button: {
        height: 44,
        borderRadius: 12,
        border: "1px solid #d9e0e7",
        background: "#f7f9fc",
        cursor: "pointer",
        fontWeight: 600,
    },
    buttonDisabled: {
        opacity: 0.6,
        cursor: "not-allowed",
    },
    progressBar: {
        position: "relative",
        height: 6,
        overflow: "hidden",
        background: "#eef1f4",
        borderRadius: 999,
    },
    progressIndeterminate: {
        position: "absolute",
        top: 0,
        left: 0,
        bottom: 0,
        width: "30%",
        transform: "translateX(-100%)",
        animation: "indet 1.2s linear infinite",
        background: "#2b6ff7",
        borderRadius: 999,
    },
    note: { fontSize: 13 },
};

// Simple keyframes via style tag (guarded for SSR)
if (typeof document !== "undefined" && !document.getElementById("indet-kf")) {
    const styleTag = document.createElement("style");
    styleTag.id = "indet-kf";
    styleTag.innerHTML = `@keyframes indet { 0% { transform: translateX(-100%); } 50% { transform: translateX(60%); } 100% { transform: translateX(120%); } }`;
    document.head.appendChild(styleTag);
}
