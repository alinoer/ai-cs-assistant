-- +goose Up
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE businesses (
  id         BIGSERIAL PRIMARY KEY,
  name       TEXT NOT NULL,
  site_key   TEXT NOT NULL UNIQUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE documents (
  id          BIGSERIAL PRIMARY KEY,
  business_id BIGINT NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
  title       TEXT NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE chunks (
  id          BIGSERIAL PRIMARY KEY,
  document_id BIGINT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  business_id BIGINT NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
  chunk_index INT NOT NULL,
  content     TEXT NOT NULL,
  embedding   vector(768) NOT NULL
);

CREATE INDEX chunks_embedding_hnsw ON chunks USING hnsw (embedding vector_cosine_ops);
CREATE INDEX chunks_business_id ON chunks (business_id);

INSERT INTO businesses (name, site_key) VALUES ('Demo Kedai Kopi', 'demo-kopi');

-- +goose Down
DROP TABLE chunks; DROP TABLE documents; DROP TABLE businesses;
