import React from "react";
import { ArrowUpRight, ArrowRight, Clock3, FilePlus2, RefreshCw, Search, TrendingUp,History } from "lucide-react";
import StatusBadge from "../components/status_badge.jsx";

const money = n => new Intl.NumberFormat("en-IN", { style: "currency", currency: "INR", maximumFractionDigits: 0 }).format(Number(n || 0));

export default function Dashboard({ rfqs, loading, onRefresh, setPage }) {
  const active = rfqs.filter(r => !["won", "lost"].includes(String(r.status).toLowerCase()));
  const sent = rfqs.filter(r => String(r.status).toLowerCase() === "sent");
  const estimating = rfqs.filter(r => String(r.status).toLowerCase() === "estimating");
  const pending = rfqs.filter(r => String(r.status).toLowerCase().includes("approval"));
  const recent = [...rfqs].slice(0, 5);

  return (
    <>
      <div className="page-heading">
        <div><span className="eyebrow"><i /> OVERVIEW</span><h1>Good morning.</h1><p>Here’s what’s happening across your RFQ workspace.</p></div>
        <button className="secondary-button" onClick={onRefresh}><RefreshCw size={15} /> Refresh data</button>
      </div>
      <section className="welcome-strip">
        <div><span className="strip-label">YOUR WORKSPACE</span><h2>Keep every quote moving.</h2><p>One place to manage incoming requests, costing and approvals.</p></div>
        <button className="strip-action" onClick={() => setPage("rfqs")}>View RFQ register <ArrowUpRight size={16} /></button>
        <div className="strip-orbit orbit-a" /><div className="strip-orbit orbit-b" />
      </section>
      <div className="section-row"><div><h2>At a glance</h2><p>Live summary from your RFQ register</p></div><span className="updated-label"><span /> {loading ? "Updating..." : "Connected to workspace"}</span></div>
      <section className="stats-grid">
        <article className="stat-card"><div className="stat-top"><span>Total RFQs</span><span className="stat-icon"><FilePlus2 size={17} /></span></div><strong>{rfqs.length.toString().padStart(2, "0")}</strong><small>All registered requests</small></article>
        <article className="stat-card"><div className="stat-top"><span>Active RFQs</span><span className="stat-icon"><TrendingUp size={17} /></span></div><strong>{active.length.toString().padStart(2, "0")}</strong><small>Currently in progress</small></article>
        <article className="stat-card"><div className="stat-top"><span>Estimating</span><span className="stat-icon"><Clock3 size={17} /></span></div><strong>{estimating.length.toString().padStart(2, "0")}</strong><small>Awaiting costing completion</small></article>
        <article className="stat-card"><div className="stat-top"><span>Quotes sent</span><span className="stat-icon"><ArrowUpRight size={17} /></span></div><strong>{sent.length.toString().padStart(2, "0")}</strong><small>Submitted to customers</small></article>
      </section>
      <section className="content-grid">
        <div className="panel recent-panel">
          <div className="panel-heading"><div><h2>Recent RFQs</h2><p>Your latest customer requests</p></div><button className="text-button" onClick={() => setPage("rfqs")}>View all <ArrowRight size={15} /></button></div>
          {loading ? <div className="empty-state">Loading RFQs...</div> : recent.length ? (
            <div className="table-wrap"><table><thead><tr><th>RFQ DETAILS</th><th>CUSTOMER</th><th>DUE DATE</th><th>STATUS</th></tr></thead><tbody>
              {recent.map(r => <tr key={r.id}><td><strong>{r.rfq_number}</strong><small>{r.part_count || 0} part(s)</small></td><td>{r.customer_name || "—"}</td><td>{r.due_date ? new Date(r.due_date).toLocaleDateString("en-IN", { day: "2-digit", month: "short", year: "numeric" }) : "—"}</td><td><StatusBadge status={r.status} /></td></tr>)}
            </tbody></table></div>
          ) : <div className="empty-state"><Search size={22} /><strong>No RFQs yet</strong><span>New customer requests will appear here.</span></div>}
        </div>
        <aside className="panel side-panel">
          <div className="panel-heading"><div><h2>Quick actions</h2><p>Jump into your workflow</p></div></div>
          <button className="quick-action" onClick={() => setPage("rfqs")}><span className="quick-icon"><FilePlus2 size={17} /></span><span><strong>Browse RFQs</strong><small>Review customer requests</small></span><ArrowUpRight size={16} /></button>
          <button className="quick-action" onClick={() => setPage("costing")}><span className="quick-icon"><TrendingUp size={17} /></span><span><strong>Start costing</strong><small>Calculate part-level costs</small></span><ArrowUpRight size={16} /></button>
          <button className="quick-action" onClick={() => setPage("rates")}><span className="quick-icon"><Clock3 size={17} /></span><span><strong>Manage rates</strong><small>Review effective rate masters</small></span><ArrowUpRight size={16} /></button>
          <button
  className="quick-action"
  onClick={() => setPage("historical_quotes")}
>
  <span className="quick-icon"><History size={17} /></span>
  <span>
    <strong>Historical quotes</strong>
    <small>Search previous customer quotations</small>
  </span>
  <ArrowUpRight size={16} />
</button>
          <div className="side-note"><span className="note-mark">i</span><span><strong>Keep it traceable.</strong><br />Every costing should use the correct approved rate version.</span></div>
        </aside>
      </section>
    </>
  );
}
