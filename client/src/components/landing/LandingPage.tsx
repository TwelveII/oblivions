import { useNavigate } from "react-router-dom";
import "../../styles/landing.css";

function LandingPage() {
  const navigate = useNavigate();

  return (
    <main className="landing-page">
      <div className="lp-content">
        <section className="lp-card">
          <img
            className="lp-logo"
            src="/oblivions-icon.svg"
            alt="Oblivions"
          />

          <h1>Oblivions</h1>
          <p className="lp-subtitle">Messaging Service</p>

          <div className="lp-actions">
            <button
              className="lp-btn lp-btn-primary"
              onClick={() => navigate("/login")}
            >
              Login
            </button>
            <button
              className="lp-btn lp-btn-secondary"
              onClick={() => navigate("/register")}
            >
              Register
            </button>
          </div>
        </section>

        <footer className="lp-footer">
          <a href="/privacy">Privacy</a>
          <span>·</span>
          <a href="/terms">Terms</a>
        </footer>
      </div>
    </main>
  );
}

export default LandingPage;
