import React from "react";

export default function Brand({ compact = false }) {
  return (
    <div className={`brand ${compact ? "brand-compact" : ""}`}>
      <span className="brand-mark" aria-hidden="true"><i /><i /><i /></span>
      {!compact && <span className="brand-name">ISOCODE <b>WORKSPACE</b></span>}
    </div>
  );
}
