# AI CS Assistant

An embeddable AI customer-support assistant for small businesses. Upload your FAQ, SOPs, product catalog, or policies, and get a chat widget that answers customers from your own documents, with sources, and says "I don't know" instead of making things up.

## Why
Small businesses answer the same questions every day on their website and chat. Generic chatbots hallucinate; rule-based bots are rigid. This project uses retrieval-augmented generation (RAG) so answers stay grounded in the business's own content.

## Features (MVP)
- Upload documents (text/Markdown, PDF) → automatic chunking and embedding
- Grounded answers with source citations
- Refuses to answer when the documents don't cover the question, and logs it as "unanswered"
- Embeddable widget: one `<script>` tag per website (site key)
- Admin dashboard: documents, chat logs, unanswered questions

## Tech Stack
- **Backend:** Go (net/http), PostgreSQL + pgvector, pgx + sqlc, goose migrations
- **Frontend:** React + TypeScript (Vite, Bun), Tailwind CSS, shadcn/ui
- **LLM:** Google Gemini (generation + embeddings), Groq as fallback for generation, swappable via a provider interface
- **Deploy:** Vercel (frontend), Fly.io (backend), Neon (Postgres)

## Architecture
Widget / Admin (React) → Go REST API → LLM provider (Gemini / Groq)
                                     → PostgreSQL + pgvector

Diagram coming soon.

## RAG Design
- **Ingestion:** documents are split into overlapping chunks, embedded, and stored in pgvector with an HNSW index.
- **Retrieval:** top-k cosine similarity search scoped per business; a similarity threshold decides whether the question is answerable.
- **Generation:** structured output (JSON schema from Go structs) with answer, sources, and an `answered` flag; validated, with one retry.
- **Reliability:** provider fallback on rate limits, per-request timeouts via context.

## Cost & Abuse Control
- Rate limiting per site key and per IP
- Input length limits
- CORS allowlist per business

## Getting Started
Instructions coming soon.

## Roadmap
- [ ] Streaming answers (SSE)
- [ ] WhatsApp integration
- [ ] Human handoff
- [ ] Retrieval evaluation set and metrics
- [ ] Multi-language answers (Indonesian / English)

## License
MIT
