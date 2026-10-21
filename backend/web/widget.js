(function () {
  const script = document.currentScript;
  const siteKey = script.dataset.siteKey;
  const apiBase = script.dataset.api || new URL(script.src).origin;
  const title = script.dataset.title || "Tanya kami";

  if (!siteKey) {
    console.error("[cs-widget] atribut data-site-key wajib diisi");
    return;
  }

  // Wadah + Shadow DOM: CSS website klien tidak bisa menembus ke sini
  const host = document.createElement("div");
  document.body.appendChild(host);
  const root = host.attachShadow({ mode: "open" });

  root.innerHTML = `
    <style>
      :host { all: initial; }
      * { box-sizing: border-box; font-family: system-ui, -apple-system, sans-serif; }
      .launcher {
        position: fixed; right: 20px; bottom: 20px; z-index: 2147483647;
        width: 56px; height: 56px; border-radius: 50%; border: none;
        background: #6b4423; color: #fff; font-size: 24px; cursor: pointer;
        box-shadow: 0 4px 12px rgba(0,0,0,.25);
      }
      .panel {
        position: fixed; right: 20px; bottom: 88px; z-index: 2147483647;
        width: min(360px, calc(100vw - 40px)); height: min(520px, calc(100vh - 120px));
        background: #fff; border-radius: 12px; overflow: hidden;
        box-shadow: 0 8px 24px rgba(0,0,0,.2);
        display: flex; flex-direction: column;
      }
      .panel[hidden] { display: none; }
      .header {
        background: #6b4423; color: #fff; padding: 12px 16px;
        display: flex; justify-content: space-between; align-items: center; font-weight: 600;
      }
      .close { background: none; border: none; color: #fff; font-size: 22px; cursor: pointer; }
      .messages { flex: 1; overflow-y: auto; padding: 12px; background: #faf7f4; }
      .msg { max-width: 85%; margin: 6px 0; padding: 8px 12px; border-radius: 12px; font-size: 14px; line-height: 1.4; }
      .msg p { margin: 0; white-space: pre-wrap; }
      .msg small { display: block; margin-top: 6px; color: #8a7a6d; font-size: 11px; }
      .user { margin-left: auto; background: #6b4423; color: #fff; }
      .bot { background: #fff; border: 1px solid #eadfd5; color: #2b2b2b; }
      .error { border-color: #e5a0a0; color: #a33; }
      .composer { display: flex; gap: 8px; padding: 10px; border-top: 1px solid #eadfd5; }
      .composer input { flex: 1; padding: 10px; border: 1px solid #d9cbbd; border-radius: 8px; font-size: 14px; }
      .composer button {
        padding: 0 14px; border: none; border-radius: 8px;
        background: #6b4423; color: #fff; cursor: pointer; font-size: 14px;
      }
      .composer button:disabled { opacity: .5; cursor: default; }
    </style>
    <button class="launcher" aria-label="Buka chat">💬</button>
    <div class="panel" hidden>
      <div class="header"><span class="title"></span><button class="close" aria-label="Tutup chat">×</button></div>
      <div class="messages" aria-live="polite"></div>
      <form class="composer">
        <input maxlength="500" placeholder="Tulis pertanyaan..." autocomplete="off" required>
        <button type="submit">Kirim</button>
      </form>
    </div>
  `;

  const launcher = root.querySelector(".launcher");
  const panel = root.querySelector(".panel");
  const messages = root.querySelector(".messages");
  const form = root.querySelector(".composer");
  const input = form.querySelector("input");
  const sendBtn = form.querySelector("button");
  root.querySelector(".title").textContent = title;

  let greeted = false;
  function toggle(open) {
    panel.hidden = !open;
    if (open && !greeted) {
      addMessage("bot", "Halo! Ada yang bisa kami bantu?");
      greeted = true;
    }
    if (open) input.focus();
  }
  launcher.addEventListener("click", () => toggle(panel.hidden));
  root.querySelector(".close").addEventListener("click", () => toggle(false));

  // Semua teks dimasukkan lewat textContent, BUKAN innerHTML → aman dari XSS
  function addMessage(kind, text, sources) {
    const el = document.createElement("div");
    el.className = "msg " + kind;
    const p = document.createElement("p");
    p.textContent = text;
    el.appendChild(p);
    if (sources && sources.length) {
      const s = document.createElement("small");
      s.textContent = "Sumber: " + sources.map((x) => x.section).join(", ");
      el.appendChild(s);
    }
    messages.appendChild(el);
    messages.scrollTop = messages.scrollHeight;
    return el;
  }

  function setBusy(busy) {
    input.disabled = busy;
    sendBtn.disabled = busy;
  }

  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    const question = input.value.trim();
    if (!question) return;

    addMessage("user", question);
    input.value = "";
    setBusy(true);
    const typing = addMessage("bot", "Sedang mengetik…");

    try {
      const res = await fetch(apiBase + "/chat", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ site_key: siteKey, question }),
      });
      const data = await res.json().catch(() => ({}));
      typing.remove();
      if (!res.ok) {
        addMessage("bot error", data.error || "Maaf, terjadi kesalahan.");
        return;
      }
      addMessage("bot", data.answer, data.sources);
    } catch (err) {
      typing.remove();
      addMessage("bot error", "Tidak bisa terhubung ke server. Coba lagi sebentar.");
    } finally {
      setBusy(false);
      input.focus();
    }
  });
})();
