# Nexus Backend

The backend engine for Nexus, a personal knowledge base assistant that retrieves, index, and answers queries about repositories and notes.

## Key Features

* **ReAct Agent**: Powered by OpenAI chat models with dynamic tool calling for searching and browsing.
* **Hybrid Retrieval**: Concurrent execution of PostgreSQL full-text (lexical)search and vector search (pgvector). 
* **Reranking**: Advanced cross-encoder relevance scoring via Cohere Rerank API.
* **Background Sync**: Continuous Github ingestion and embedding generation for new documents.

## Tech Stack

* Go (Golang)
* PostgreSQL + pgvector
* OpenAI (Embeddings & Chat API)
* Cohere (Rerank API)

## Setup and Running

1. **Environment Config**: Copy `.env.example` to `.env` and fill in the required API keys (OpenAI, Cohere, GitHub) and database connection details.
2. **Start Database**: Use the provided `docker-compose.yml` to launch Postgres with pgvector support.
3. **Run API Server**: Start the API server on port 8080.
   ```bash
   go run cmd/api/main.go
   ```
4. **Run CLI Client**: Start the interactive command-line assistant.
   ```bash
   go run cmd/query/main.go
   ```
5. **Run Ingester**: Start the background Github sync and embedding enrichment worker.
   ```bash
   go run cmd/ingester/main.go
   ```
