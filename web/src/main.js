import axios from "axios";
import "./style.css";

const app = document.querySelector("#app");

app.innerHTML = `
  <main class="card">
    <h1>Claims Processor</h1>
    <p class="subtitle">A JFrog Xray + AppTrust compliant-promotion demo</p>

    <button id="ping" type="button">Call the back-end</button>

    <section id="result" class="result" aria-live="polite">
      <p class="muted">Click the button to call <code>/api/hello</code>.</p>
    </section>

    <footer class="footer">
      <span id="status" class="status status--idle">idle</span>
    </footer>
  </main>
`;

const resultEl = document.querySelector("#result");
const statusEl = document.querySelector("#status");
const buttonEl = document.querySelector("#ping");

function setStatus(state, label) {
  statusEl.textContent = label;
  statusEl.className = `status status--${state}`;
}

async function callBackend() {
  setStatus("loading", "calling…");
  buttonEl.disabled = true;
  try {
    const { data } = await axios.get("/api/hello");
    resultEl.innerHTML = `
      <p class="message">${data.message}</p>
      <dl class="meta">
        <dt>service</dt><dd>${data.service}</dd>
        <dt>version</dt><dd>${data.version}</dd>
        <dt>timestamp</dt><dd>${data.timestamp}</dd>
      </dl>
    `;
    setStatus("ok", "200 OK");
  } catch (err) {
    resultEl.innerHTML = `<p class="error">Request failed: ${err.message}</p>`;
    setStatus("error", "error");
  } finally {
    buttonEl.disabled = false;
  }
}

buttonEl.addEventListener("click", callBackend);
