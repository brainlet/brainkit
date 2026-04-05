# vector/ Fixtures

Tests the vector store API across backends: index creation, vector upsert, similarity query, index listing, index description, vector deletion, and index deletion.

Vector fixtures do not require AI (the `vector` category is not in `aiCategories`). They require containers or servers based on the backend path segment.

## Fixtures

### create-upsert-query/

| Fixture | AI | Container | What it tests |
|---------|-----|-----------|---------------|
| libsql | no | libsql-server | LibSQLVector: createIndex, listIndexes, upsert 3 vectors, query topK=2, deleteIndex; full CRUD lifecycle |
| mongodb | no | mongodb | MongoDBVector: createIndex + upsert on Community Edition (no Atlas Search); asserts upsert succeeded |
| pgvector | no | postgres | PgVector: createIndex, upsert 2 vectors, query topK=2, deleteIndex; asserts result count and top match |

### methods/

| Fixture | AI | Container | What it tests |
|---------|-----|-----------|---------------|
| libsql | no | libsql-server | LibSQLVector comprehensive methods: createIndex (x2), listIndexes, describeIndex, upsert, query, deleteVectors, query-after-delete, deleteIndex, listIndexes-after-delete |
| mongodb | no | mongodb | MongoDBVector on Community: connect, createIndex (expected fail), fallback upsert to backing collection, disconnect; asserts 2 vectors upserted |
| pgvector | no | postgres | PgVector: createIndex, upsert 4 vectors with metadata, query by nearest vector, asserts result count=2 and correct top match |
