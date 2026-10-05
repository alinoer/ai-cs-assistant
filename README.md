# AI Interview Coach

Practice job interviews in English, tailored to a real job description, and get instant feedback on both **what you say** and **how you say it**.

## Why
Many developers in non-English-speaking countries have the technical skills for remote roles, but struggle in English interviews. Generic question banks don't match the actual job, and practicing alone gives no feedback. This app turns any job description into a focused mock interview and reviews every answer for content quality and English clarity.

## Features (MVP)
- Paste a job description → get 5–8 tailored interview questions (technical + behavioral)
- Answer in English (text) → receive structured feedback:
  - Content score and what's missing
  - Grammar and clarity corrections
  - A stronger example answer
- Session history to track progress over time

## Tech Stack
- **Backend:** Go (net/http), PostgreSQL, pgx + sqlc, goose migrations
- **Frontend:** React + TypeScript (Vite), Tailwind CSS, shadcn/ui
- **LLM:** Google Gemini (default) and Anthropic Claude, swappable via a provider interface
- **Deploy:** Vercel (frontend), Fly.io (backend), Neon (Postgres)

## Architecture
React SPA → Go REST API → LLM provider (Gemini / Claude)
                        → PostgreSQL

Diagram coming soon.

## LLM Design
- **Provider interface:** one Go interface, multiple implementations; the model is selected by environment variable.
- **Structured output:** responses are constrained by JSON schemas generated from Go structs, then validated, with one retry on invalid output.
- **Concurrency:** content feedback and language feedback run in parallel goroutines (errgroup).
- **Reliability:** per-request timeouts via context, graceful error messages.

## Cost & Abuse Control
- Per-IP rate limiting
- Input length limits for job descriptions and answers
- Provider-level spending caps
- Estimated cost: ~$0.05 per full practice session

## Getting Started
Instructions coming soon.

## Roadmap
- [ ] Voice answers (Web Speech API)
- [ ] Streaming feedback (SSE)
- [ ] Upload CV → RAG with pgvector to suggest relevant experiences for each answer
- [ ] CV ↔ job description matching
- [ ] Model evaluation: Gemini vs Claude on feedback quality

## License
MIT
