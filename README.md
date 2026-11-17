# API de Ocorrências (Go)

API REST para registro de ocorrências em rodovias, feita com a biblioteca padrão do Go (`net/http`) e PostgreSQL via `pgx`. É a versão em Go da [ocorrencias-api](https://github.com/jot4-ge/ocorrencias-api) (FastAPI), com o mesmo contrato de endpoints.

## Stack

- Go 1.22+ (roteamento com método e parâmetros na biblioteca padrão)
- PostgreSQL 16 e `pgx/v5` (pool de conexões, SQL com parâmetros)
- Docker Compose (banco de dados)

## Como rodar

Pré-requisitos: Go 1.22+, Docker Desktop e Git.

```bash
# 1. Clonar e entrar na pasta
git clone https://github.com/jot4-ge/ocorrencias-api-go.git
cd ocorrencias-api-go

# 2. Subir o PostgreSQL (porta 5433)
docker compose up -d

# 3. Iniciar a API
go run .
```

A API sobe em http://localhost:8081. A tabela é criada automaticamente na inicialização.

A conexão usa a variável `DATABASE_URL`. Sem ela, o padrão é
`postgres://postgres:postgres@localhost:5433/ocorrencias_go` (apenas para desenvolvimento local).

## Endpoints

| Método | Rota | Descrição |
|---|---|---|
| GET | `/ocorrencias` | Lista com filtros `rodovia` e `tipo` |
| GET | `/ocorrencias/total` | Total de ocorrências |
| GET | `/ocorrencias/{id}` | Busca por id (404 se não existir) |
| POST | `/ocorrencias` | Cria uma ocorrência (201) |
| PUT | `/ocorrencias/{id}` | Substitui a ocorrência inteira |
| PATCH | `/ocorrencias/{id}` | Altera apenas os campos enviados |
| DELETE | `/ocorrencias/{id}` | Remove (204) |

### Regras de validação

- `rodovia`: formato `BR-` seguido de 3 dígitos (ex.: `BR-040`)
- `tipo`: `acidente`, `buraco`, `alagamento` ou `obstrucao`
- `km`: inteiro de 0 a 5000
- `descricao`: opcional, até 200 caracteres

Erros seguem o formato `{"detail": "mensagem"}`, com status 404 (não encontrada), 422 (dados inválidos) e 400 (JSON malformado).

### Exemplo

```bash
curl -X POST http://localhost:8081/ocorrencias \
  -H "Content-Type: application/json" \
  -d '{"rodovia": "BR-040", "tipo": "acidente", "km": 12}'
```

## Estrutura

```
├── main.go            # rotas, handlers e validação
├── repo.go            # acesso ao banco (SQL com pgx)
├── docker-compose.yml # PostgreSQL
├── go.mod
└── go.sum
```

## Decisões de projeto

- Biblioteca padrão em vez de framework: o roteador do Go 1.22 já cobre método e `{id}`.
- Repositório separado dos handlers: o SQL fica em `repo.go`, e os handlers decidem o status HTTP.
- SQL parametrizado (`$1`, `$2`) para evitar SQL injection.

## Próximos passos

- Testes automatizados com `go test`
- Migrações de esquema
- Front-end em React consumindo a API