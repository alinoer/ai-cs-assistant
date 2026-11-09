// AI CS Assistant — embeddable chat widget.
// Kontrak atribut pada tag <script>:
//   data-site-key   (wajib)  id tenant
//   data-title      (opsi)   judul di header
//   data-api        (opsi)   base URL API, default: origin dari src script
//   data-accent     (opsi)   warna aksen, default: #6b4423
//   data-position   (opsi)   "br" (default) atau "bl"
// Aman XSS: semua teks dari server dirender via textContent.
(function () {
  const script = document.currentScript;
  const siteKey = script.dataset.siteKey;
  const apiBase = script.dataset.api || new URL(script.src).origin;
  const title = script.dataset.title || "Tanya kami";
  const accent = script.dataset.accent || "#6b4423";
  const position = script.dataset.position === "bl" ? "bl" : "br";

  if (!siteKey) {
    console.error("[cs-widget] atribut data-site-key wajib diisi");
    return;
  }

  const host = document.createElement("div");
  document.body.appendChild(host);
  const root = host.attachShadow({ mode: "open" });

  root.innerHTML = `
    <style>
      :host {
        all: initial;
        --accent: ${cssColor(accent)};
        --accent-fg: #ffffff;
        --bg-panel: #ffffff;
        --bg-surface: #f6f3ef;
        --bg-bot: #ffffff;
        --bg-not-found: #fff8ec;
        --fg: #1f1a16;
        --fg-muted: #6b6560;
        --border: #e6dfd6;
        --border-strong: #d7cec1;
        --error: #9b2c2c;
        --error-bg: #fdecec;
        --radius-panel: 14px;
        --radius-bubble: 14px;
        --radius-sm: 8px;
        --shadow: 0 10px 30px rgba(31, 26, 22, .18), 0 2px 6px rgba(31, 26, 22, .08);
        --font: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
      }
      * { box-sizing: border-box; font-family: var(--font); }

      .launcher {
        position: fixed; z-index: 2147483647;
        ${position === "br" ? "right: 20px;" : "left: 20px;"} bottom: 20px;
        width: 56px; height: 56px;
        border-radius: 18px;
        border: none;
        background: var(--accent); color: var(--accent-fg);
        cursor: pointer;
        display: grid; place-items: center;
        box-shadow: var(--shadow);
        transition: transform .15s ease, box-shadow .15s ease;
      }
      .launcher:hover { transform: translateY(-1px); }
      .launcher:focus-visible {
        outline: 3px solid color-mix(in srgb, var(--accent) 40%, white);
        outline-offset: 2px;
      }
      .launcher svg { width: 26px; height: 26px; display: block; }
      .launcher .close-icon { display: none; }
      .launcher[aria-expanded="true"] .open-icon { display: none; }
      .launcher[aria-expanded="true"] .close-icon { display: block; }

      .panel {
        position: fixed; z-index: 2147483647;
        ${position === "br" ? "right: 20px;" : "left: 20px;"} bottom: 88px;
        width: min(380px, calc(100vw - 32px));
        height: min(560px, calc(100vh - 120px));
        background: var(--bg-panel);
        color: var(--fg);
        border-radius: var(--radius-panel);
        overflow: hidden;
        box-shadow: var(--shadow);
        display: flex; flex-direction: column;
        border: 1px solid var(--border);
      }
      .panel[hidden] { display: none; }

      @media (max-width: 480px) {
        .panel {
          right: 8px; left: 8px; bottom: 84px;
          width: auto;
          height: calc(100vh - 100px);
          max-height: 640px;
        }
      }

      .header {
        display: flex; align-items: center; gap: 10px;
        padding: 12px 14px;
        border-bottom: 1px solid var(--border);
        background: var(--bg-panel);
      }
      .header .dot {
        width: 8px; height: 8px; border-radius: 50%;
        background: #2d6a4f;
        box-shadow: 0 0 0 3px color-mix(in srgb, #2d6a4f 20%, transparent);
        flex: 0 0 auto;
      }
      .header .title { font-size: 14px; font-weight: 600; letter-spacing: -0.01em; }
      .header .sub { font-size: 11px; color: var(--fg-muted); margin-top: 1px; }
      .header .meta { flex: 1; min-width: 0; }
      .header .meta > div { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
      .icon-btn {
        background: none; border: none; color: var(--fg-muted);
        width: 32px; height: 32px; border-radius: 8px;
        display: grid; place-items: center; cursor: pointer;
      }
      .icon-btn:hover { background: var(--bg-surface); color: var(--fg); }
      .icon-btn:focus-visible {
        outline: 2px solid var(--accent); outline-offset: 2px;
      }

      .messages {
        flex: 1; overflow-y: auto;
        padding: 14px;
        background: var(--bg-surface);
        display: flex; flex-direction: column; gap: 8px;
        scroll-behavior: smooth;
      }

      .msg {
        max-width: 85%;
        padding: 9px 12px;
        border-radius: var(--radius-bubble);
        font-size: 14px; line-height: 1.5;
        word-wrap: break-word;
      }
      .msg p { margin: 0; white-space: pre-wrap; }

      .user {
        align-self: flex-end;
        background: var(--accent); color: var(--accent-fg);
        border-bottom-right-radius: 4px;
      }
      .bot {
        align-self: flex-start;
        background: var(--bg-bot); color: var(--fg);
        border: 1px solid var(--border);
        border-bottom-left-radius: 4px;
      }
      .bot.not-found {
        background: var(--bg-not-found);
        border-color: #f0dfb7;
      }
      .bot.error {
        background: var(--error-bg);
        border-color: #f1c7c7;
        color: var(--error);
      }

      .sources {
        display: flex; flex-wrap: wrap; gap: 6px;
        margin-top: 8px;
      }
      .source-chip {
        font-size: 11px; line-height: 1;
        padding: 4px 8px;
        border-radius: 6px;
        background: color-mix(in srgb, var(--accent) 10%, white);
        color: color-mix(in srgb, var(--accent) 85%, black);
        border: 1px solid color-mix(in srgb, var(--accent) 25%, white);
        font-variant-numeric: tabular-nums;
        cursor: default;
      }

      .typing {
        align-self: flex-start;
        background: var(--bg-bot); border: 1px solid var(--border);
        border-radius: var(--radius-bubble); border-bottom-left-radius: 4px;
        padding: 10px 14px;
        display: flex; gap: 4px;
      }
      .typing span {
        width: 6px; height: 6px; border-radius: 50%;
        background: var(--fg-muted);
        animation: blink 1.2s infinite ease-in-out;
      }
      .typing span:nth-child(2) { animation-delay: .15s; }
      .typing span:nth-child(3) { animation-delay: .3s; }
      @keyframes blink {
        0%, 60%, 100% { opacity: .25; transform: translateY(0); }
        30% { opacity: 1; transform: translateY(-2px); }
      }

      .suggestions {
        display: flex; flex-wrap: wrap; gap: 6px;
        margin-top: 4px;
      }
      .suggest {
        font: inherit; font-size: 12px;
        padding: 6px 10px;
        border-radius: 999px;
        border: 1px solid var(--border-strong);
        background: var(--bg-panel);
        color: var(--fg);
        cursor: pointer;
        text-align: left;
      }
      .suggest:hover { border-color: var(--accent); color: var(--accent); }
      .suggest:focus-visible { outline: 2px solid var(--accent); outline-offset: 2px; }

      .composer {
        display: flex; gap: 8px; align-items: flex-end;
        padding: 10px;
        border-top: 1px solid var(--border);
        background: var(--bg-panel);
      }
      .composer textarea {
        flex: 1;
        resize: none;
        min-height: 40px; max-height: 120px;
        padding: 10px 12px;
        border: 1px solid var(--border-strong);
        border-radius: 10px;
        font-size: 14px; line-height: 1.4;
        color: var(--fg); background: var(--bg-panel);
        font-family: var(--font);
      }
      .composer textarea:focus {
        outline: none;
        border-color: var(--accent);
        box-shadow: 0 0 0 3px color-mix(in srgb, var(--accent) 20%, transparent);
      }
      .composer textarea:disabled { opacity: .6; }
      .send {
        width: 40px; height: 40px;
        border: none; border-radius: 10px;
        background: var(--accent); color: var(--accent-fg);
        cursor: pointer;
        display: grid; place-items: center;
        flex: 0 0 auto;
      }
      .send:disabled { opacity: .4; cursor: default; }
      .send:focus-visible { outline: 2px solid var(--accent); outline-offset: 2px; }
      .send svg { width: 18px; height: 18px; }

      .foot {
        font-size: 10px; color: var(--fg-muted);
        text-align: center; padding: 6px 10px 10px;
        background: var(--bg-panel);
      }

      @media (prefers-reduced-motion: reduce) {
        .launcher, .typing span { animation: none; transition: none; }
      }
    </style>

    <button class="launcher" aria-label="Buka chat" aria-expanded="false">
      <svg class="open-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
        <path d="M4 5.5A2.5 2.5 0 0 1 6.5 3h11A2.5 2.5 0 0 1 20 5.5v8A2.5 2.5 0 0 1 17.5 16H10l-4.3 3.6a.6.6 0 0 1-1-.46V16A2.5 2.5 0 0 1 4 13.5v-8Z"
              stroke="currentColor" stroke-width="1.6" stroke-linejoin="round"/>
      </svg>
      <svg class="close-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
        <path d="M6 6l12 12M18 6L6 18" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
      </svg>
    </button>

    <div class="panel" hidden role="dialog" aria-label="Jendela chat">
      <div class="header">
        <span class="dot" aria-hidden="true"></span>
        <div class="meta">
          <div class="title"></div>
          <div class="sub">Biasanya balas dalam beberapa detik</div>
        </div>
        <button class="icon-btn close" aria-label="Tutup chat">
          <svg viewBox="0 0 24 24" width="16" height="16" fill="none" aria-hidden="true">
            <path d="M6 6l12 12M18 6L6 18" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
          </svg>
        </button>
      </div>

      <div class="messages" aria-live="polite" aria-atomic="false"></div>

      <form class="composer">
        <textarea rows="1" maxlength="500" placeholder="Tulis pertanyaan…" autocomplete="off" required aria-label="Pertanyaan"></textarea>
        <button class="send" type="submit" aria-label="Kirim">
          <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
            <path d="M4 12l16-8-6 16-2.5-6.5L4 12Z" stroke="currentColor" stroke-width="1.6" stroke-linejoin="round" fill="none"/>
          </svg>
        </button>
      </form>
      <div class="foot">Jawaban berdasar dokumen resmi — jika tidak ada, akan diberitahu.</div>
    </div>
  `;

  const launcher = root.querySelector(".launcher");
  const panel = root.querySelector(".panel");
  const messages = root.querySelector(".messages");
  const form = root.querySelector(".composer");
  const input = form.querySelector("textarea");
  const sendBtn = form.querySelector(".send");
  root.querySelector(".title").textContent = title;

  const SUGGESTIONS = ["Jam buka berapa?", "Harga Americano?", "Cara order lewat GoFood?"];

  let greeted = false;
  function toggle(open) {
    panel.hidden = !open;
    launcher.setAttribute("aria-expanded", open ? "true" : "false");
    launcher.setAttribute("aria-label", open ? "Tutup chat" : "Buka chat");
    if (open && !greeted) {
      addMessage("bot", "Halo! Silakan tanya apa saja tentang " + title + ". Beberapa contoh:");
      addSuggestions();
      greeted = true;
    }
    if (open) setTimeout(() => input.focus(), 50);
  }
  launcher.addEventListener("click", () => toggle(panel.hidden));
  root.querySelector(".close").addEventListener("click", () => toggle(false));

  // Textarea auto-grow + Enter kirim / Shift+Enter baris baru.
  input.addEventListener("input", () => {
    input.style.height = "auto";
    input.style.height = Math.min(input.scrollHeight, 120) + "px";
  });
  input.addEventListener("keydown", (e) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      form.requestSubmit();
    }
  });

  function addMessage(kind, text, sources) {
    const el = document.createElement("div");
    el.className = "msg " + kind;
    const p = document.createElement("p");
    p.textContent = text;
    el.appendChild(p);
    if (sources && sources.length) {
      const wrap = document.createElement("div");
      wrap.className = "sources";
      sources.forEach((s, i) => {
        const chip = document.createElement("span");
        chip.className = "source-chip";
        chip.textContent = "[" + (i + 1) + "] " + (s.section || "");
        if (s.section) chip.title = s.section;
        wrap.appendChild(chip);
      });
      el.appendChild(wrap);
    }
    messages.appendChild(el);
    messages.scrollTop = messages.scrollHeight;
    return el;
  }

  function addSuggestions() {
    const wrap = document.createElement("div");
    wrap.className = "suggestions";
    SUGGESTIONS.forEach((q) => {
      const b = document.createElement("button");
      b.type = "button";
      b.className = "suggest";
      b.textContent = q;
      b.addEventListener("click", () => {
        if (input.disabled) return;
        input.value = q;
        form.requestSubmit();
      });
      wrap.appendChild(b);
    });
    messages.appendChild(wrap);
    return wrap;
  }

  function addTyping() {
    const el = document.createElement("div");
    el.className = "typing";
    el.setAttribute("aria-label", "Sedang mengetik");
    el.innerHTML = "<span></span><span></span><span></span>";
    messages.appendChild(el);
    messages.scrollTop = messages.scrollHeight;
    return el;
  }

  function setBusy(busy) {
    input.disabled = busy;
    sendBtn.disabled = busy;
  }

  function clearSuggestions() {
    messages.querySelectorAll(".suggestions").forEach((n) => n.remove());
  }

  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    const question = input.value.trim();
    if (!question) return;

    clearSuggestions();
    addMessage("user", question);
    input.value = "";
    input.style.height = "auto";
    setBusy(true);
    const typing = addTyping();

    try {
      const res = await fetch(apiBase + "/chat", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ site_key: siteKey, question }),
      });
      const data = await res.json().catch(() => ({}));
      typing.remove();
      if (!res.ok) {
        const msg = res.status === 429
          ? "Terlalu banyak permintaan. Coba lagi sebentar ya."
          : (data.error || "Maaf, terjadi kesalahan.");
        addMessage("bot error", msg);
        return;
      }
      // Jawaban tanpa sumber = bot tidak menemukan jawaban di dokumen.
      // Tampilkan dengan nuansa "tenang", bukan error merah.
      const hasSources = Array.isArray(data.sources) && data.sources.length > 0;
      addMessage(hasSources ? "bot" : "bot not-found", data.answer || "", data.sources);
    } catch (err) {
      typing.remove();
      addMessage("bot error", "Tidak bisa terhubung ke server. Coba lagi sebentar.");
    } finally {
      setBusy(false);
      input.focus();
    }
  });

  // Sanitasi nilai warna dari data-accent agar tidak bisa menyuntikkan CSS lain.
  function cssColor(v) {
    if (/^#[0-9a-fA-F]{3,8}$/.test(v)) return v;
    if (/^(rgb|hsl)a?\([0-9.,%\s/]+\)$/.test(v)) return v;
    return "#6b4423";
  }
})();
