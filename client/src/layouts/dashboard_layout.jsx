import React from "react";
import { Bell, ChevronDown, LogOut, Menu, X } from "lucide-react";
import Brand from "../components/brand.jsx";

const navItems = [
  { id: "dashboard", label: "Overview", icon: "⌂" },
  { id: "intake", label: "AI intake", icon: "✦" },
  { id: "rfqs", label: "RFQ register", icon: "▤" },
  { id: "costing", label: "Costing", icon: "◫" },
  { id: "rates", label: "Rate masters", icon: "↗" },
  { id: "quotes", label: "Quotations", icon: "▧" }
];

export default function DashboardLayout({ page, setPage, profile, onLogout, children }) {
  const [mobileOpen, setMobileOpen] = React.useState(false);
  const name = profile?.name || profile?.email || "Estimator";
  const role = (profile?.role || "estimator").replaceAll("_", " ");
  const isSalesHead = profile?.role === "sales_head";
  const visibleItems = navItems.filter(item => !isSalesHead || item.id !== "intake");

  return (
    <div className="app-shell">
      <aside className={`sidebar ${mobileOpen ? "sidebar-open" : ""}`}>
        <div className="sidebar-brand"><Brand /></div>
        <div className="workspace-label">WORKSPACE</div>
        <nav className="side-nav">
          {visibleItems.map(item => (
            <button key={item.id} className={`nav-link ${page === item.id ? "active" : ""}`}
              onClick={() => { setPage(item.id); setMobileOpen(false); }}>
              <span className="nav-icon">{item.icon}</span><span>{item.label}</span>
              {item.id === "approvals" && <span className="nav-dot" />}
            </button>
          ))}
        </nav>
        <div className="sidebar-bottom">
          <div className="help-card">
            <span className="help-kicker">ISOCODE LABS</span>
            <strong>Make work flow.</strong>
            <span>RFQ Costing Workspace</span>
          </div>
          <button className="profile-mini" onClick={onLogout}>
            <span className="avatar">{String(name).slice(0, 1).toUpperCase()}</span>
            <span className="profile-copy"><strong>{name}</strong><small>{role}</small></span>
            <LogOut size={15} />
          </button>
        </div>
      </aside>
      {mobileOpen && <button className="mobile-scrim" onClick={() => setMobileOpen(false)} aria-label="Close menu" />}
      <main className="main-area">
        <header className="topbar">
          <button className="icon-button mobile-menu" onClick={() => setMobileOpen(v => !v)} aria-label="Toggle menu">
            {mobileOpen ? <X size={19} /> : <Menu size={19} />}
          </button>
          <div className="breadcrumb">Workspace <span>/</span> <strong>{navItems.find(x => x.id === page)?.label || "Overview"}</strong></div>
          <div className="topbar-right">
            <span className="today-label">{new Intl.DateTimeFormat("en-IN", { dateStyle: "medium" }).format(new Date())}</span>
            <button className="icon-button" title="Notifications"><Bell size={17} /></button>
            <div className="top-avatar">{String(name).slice(0, 1).toUpperCase()}</div>
            <span className="top-name">{name}</span>
            <ChevronDown size={14} className="muted-icon" />
          </div>
        </header>
        <div className="page-wrap">{children}</div>
        <footer className="app-footer"><span>ISOCODE LABS</span><span>RFQ COSTING WORKSPACE <i>·</i> INTERNAL</span></footer>
      </main>
    </div>
  );
}
