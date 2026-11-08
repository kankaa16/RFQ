
import React from "react";
import { api, getToken, setToken, clearToken } from "./services/api.js";
import DashboardLayout from "./layouts/dashboard_layout.jsx";
import AnimatedBackground from "./components/animated_background.jsx";
import Login from "./pages/login.jsx";
import Dashboard from "./pages/dashboard.jsx";
import SalesHeadDashboard from "./pages/sales_head_dashboard.jsx";
import RFQs from "./pages/rfqs.jsx";
import Intake from "./pages/intake.jsx";
import Costing from "./pages/costing.jsx";
import Rates from "./pages/rates.jsx";
import Quotes from "./pages/quotes.jsx";
import HistoricalQuotes from "./pages/historical_quotes.jsx";

export default function App() {
  const [token, setAuth] = React.useState(getToken());
  const [profile, setProfile] = React.useState(null);
  const [page, setPage] = React.useState("dashboard");
  const [rfqs, setRfqs] = React.useState([]);
  const [rates, setRates] = React.useState([]);
  const [rfqLoading, setRfqLoading] = React.useState(false);
  const [rateLoading, setRateLoading] = React.useState(false);
  const [loginLoading, setLoginLoading] = React.useState(false);
  const [loginError, setLoginError] = React.useState("");
  const [globalError, setGlobalError] = React.useState("");

  const loadProfile = React.useCallback(async () => {
    try {
      const res = await api.profile();
      setProfile(res.data || res);
    } catch {
      clearToken();
      setAuth("");
    }
  }, []);

  const loadRFQs = React.useCallback(async () => {
    setRfqLoading(true);
    setGlobalError("");

    try {
      const res = await api.rfqs();
      setRfqs(res.data || []);
    } catch (e) {
      setGlobalError(e.message);
    } finally {
      setRfqLoading(false);
    }
  }, []);

  const loadRates = React.useCallback(async () => {
    setRateLoading(true);

    try {
      const res = await api.rates();
      setRates(res.data || []);
    } catch (e) {
      setGlobalError(e.message);
    } finally {
      setRateLoading(false);
    }
  }, []);

  React.useEffect(() => {
    if (token) {
      loadProfile();
      loadRFQs();
      loadRates();
    }
  }, [token, loadProfile, loadRFQs, loadRates]);

  async function handleLogin(email, password) {
    setLoginLoading(true);
    setLoginError("");

    try {
      const res = await api.login(email, password);
      const data = res.data || res;
      const jwt = data.token || data.access_token || data.jwt;

      if (!jwt) throw new Error("Login response did not contain a token.");

      setToken(jwt);
      setAuth(jwt);
      setProfile(data.user || data.profile || null);
    } catch (e) {
      setLoginError(e.message);
    } finally {
      setLoginLoading(false);
    }
  }

  function logout() {
    clearToken();
    setAuth("");
    setProfile(null);
    setRfqs([]);
    setRates([]);
    setPage("dashboard");
  }

  async function loadParts(rfqId) {
    const res = await api.parts(rfqId);
    return res.data || [];
  }

  async function calculateCosting(data) {
    return api.calculateCosting(data);
  }

  if (!token) {
    return <Login onLogin={handleLogin} loading={loginLoading} error={loginError} />;
  }

  const userRole = String(profile?.role || "").toLowerCase();
  const isSalesHead = userRole === "sales_head";

  return (
    <>
      <AnimatedBackground />

      <DashboardLayout
        page={page}
        setPage={setPage}
        profile={profile}
        onLogout={logout}
      >
        {globalError && (
          <div className="global-error">
            {globalError}
            <button onClick={() => setGlobalError("")}>Dismiss</button>
          </div>
        )}

        {page === "dashboard" && (
          isSalesHead ? (
            <SalesHeadDashboard
              rfqs={rfqs}
              loading={rfqLoading}
              onRefresh={loadRFQs}
              setPage={setPage}
            />
          ) : (
            <Dashboard
              rfqs={rfqs}
              loading={rfqLoading}
              onRefresh={loadRFQs}
              setPage={setPage}
            />
          )
        )}

        {page === "intake" && <Intake onConverted={loadRFQs} />}

        {page === "rfqs" && (
          <RFQs
            rfqs={rfqs}
            loading={rfqLoading}
            onRefresh={loadRFQs}
            onLoadParts={loadParts}
          />
        )}

        {page === "costing" && (
          <Costing
            rfqs={rfqs}
            onLoadParts={loadParts}
            onCalculate={calculateCosting}
          />
        )}

        {page === "rates" && (
          <Rates rates={rates} loading={rateLoading} onRefresh={loadRates} />
        )}

        {/* {page === "approvals" && <Approval />} */}
        {page === "quotes" && <Quotes />}
        {page === "historical_quotes" && <HistoricalQuotes />}
      </DashboardLayout>
    </>
  );
}
