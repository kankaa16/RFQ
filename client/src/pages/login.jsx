import React from "react";
import { ArrowUpRight, LockKeyhole } from "lucide-react";
import Brand from "../components/brand.jsx";
import AnimatedBackground from "../components/animated_background.jsx";

export default function Login({ onLogin, loading, error }) {
  const [email, setEmail] = React.useState("");
  const [password, setPassword] = React.useState("");

  function submit(e) {
    e.preventDefault();
    onLogin(email, password);
  }

  return (
    <div className="login-page">
      <AnimatedBackground />
      <header className="login-header"><Brand /><span>INTERNAL WORKSPACE</span></header>
      <main className="login-content">
        <div className="login-intro">
          <span className="eyebrow"><i /> ISOCODE LABS / OPERATIONS</span>
          <h1>Good work<br /><em>starts here.</em></h1>
          <p>A focused workspace for RFQs, costing and customer quotations.</p>
        </div>
        <form className="login-card" onSubmit={submit}>
          <div className="login-card-top"><span className="lock-icon"><LockKeyhole size={17} /></span><span>SECURE SIGN IN</span></div>
          <h2>Welcome back.</h2>
          <p>Sign in to continue to your workspace.</p>
          <label>Email address</label>
          <input type="email" value={email} onChange={e => setEmail(e.target.value)} placeholder="you@company.com" required />
          <label>Password</label>
          <input type="password" value={password} onChange={e => setPassword(e.target.value)} placeholder="Enter your password" required />
          {error && <div className="form-error">{error}</div>}
          <button className="primary-button full-button" disabled={loading}>{loading ? "Signing in..." : <>Sign in <ArrowUpRight size={17} /></>}</button>
          <div className="login-note">Access is restricted to authorised team members.</div>
        </form>
      </main>
      <footer className="login-footer"><span>© 2026 ISOCODE LABS</span><span>BUILT FOR BETTER WORKFLOWS</span></footer>
    </div>
  );
}
