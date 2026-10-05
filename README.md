# AI Interview Coach

Latihan wawancara kerja berbasis AI: pengguna menjawab pertanyaan wawancara, model memberi umpan balik.

Monorepo dengan dua bagian:

- `backend/` — HTTP server Go (saat ini hanya `/healthz`).
- `frontend/` — SPA React + TypeScript via Vite, dikelola dengan Bun.

## Tech Stack

- **Backend:** Go 1.24
- **Frontend:** React 19, TypeScript, Vite, Bun
- **LLM:** Gemini / Anthropic (dipilih via env `LLM_PROVIDER`)

## Getting Started

### Backend

```bash
cd backend
cp .env.example .env   # isi API key
go run .
```

Server berjalan di `http://localhost:8080`.

### Frontend

```bash
cd frontend
bun install
bun run dev
```

Dev server berjalan di `http://localhost:5173`.

## Struktur

```
.
├── backend/     # Go API
└── frontend/    # React SPA (Vite + Bun)
```
