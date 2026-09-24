# Chateauneuf Portaria Backend

API local em Go para controle de acesso de prestadores, com persistencia SQLite offline-first e sincronizacao posterior com Google Sheets.

## Rodar

```sh
go mod tidy
go run ./cmd/api
```

Copie `.env.example` para `.env` ou exporte as variaveis no ambiente antes de iniciar.

## Docker

Este backend possui um `Dockerfile` multi-stage. No container, use:

```env
HTTP_ADDR=:8080
DATABASE_PATH=/app/data/portaria.db
MIGRATIONS_PATH=/app/migrations
GOOGLE_CREDENTIALS_FILE=/run/secrets/google-service-account.json
GOOGLE_DRIVE_FOLDER_ID=ID_DA_PASTA_DO_DRIVE
PHOTO_STORAGE_DIR=/app/photos
```

Build isolado:

```sh
docker build -t chateauneuf-portaria-backend:local .
```

O modo recomendado para a estacao e subir pelo `docker-compose.yml` do frontend, que monta:

- volume persistente em `/app/data`
- credencial Google em `/run/secrets/google-service-account.json`

Para conta Google gratuita, use `PHOTO_STORAGE_DIR` apontando para uma pasta local sincronizada pelo Google Drive para computador. A API guarda no banco uma URL local `/api/photos/...` para fotos de entradas, compras, diaristas e servicos agendados, a aplicacao exibe por essa URL e o Google Drive para computador sincroniza os arquivos com a sua conta. Fotos de moradores permanecem embutidas no cadastro para preservar a melhor qualidade possivel dentro do registro.

`GOOGLE_DRIVE_FOLDER_ID` continua disponivel apenas para Drive compartilhado/Workspace. Evite pasta em "Meu Drive" via service account: service accounts nao possuem cota propria de armazenamento e o Google pode retornar `storageQuotaExceeded`.

## Endpoints

### Estoque de produtos de limpeza

- `GET /api/inventory`: produtos, saldos, compras e movimentacoes.
- `POST /api/inventory/products`: produto, unidade, estoque minimo e saldo inicial contado.
- `POST /api/inventory/products/{productID}`: altera nome e estoque minimo; a unidade e preservada.
- `POST /api/inventory/purchases`: compra com data, fornecedor, documento opcional e itens com quantidade e preco unitario.
- `POST /api/inventory/withdrawals`: saida com produto, quantidade, data e responsavel obrigatorio.

Quantidades usam milesimos da unidade (`quantityMilli`, `stockMilli`) e valores usam
centavos (`unitCostCents`, `totalCents`). Cada POST inclui `requestId` para permitir
reenvio sem duplicar estoque. Compras com varios itens sao atomicas e retiradas nao
podem deixar saldo negativo. Os dados ficam no SQLite e nos backups da portaria;
este modulo ainda nao sincroniza com Google Sheets.

### Demais registros

- `POST /api/access-logs`
- `GET /api/access-logs`
- `GET /api/access-logs/open`
- `PATCH /api/access-logs/{id}/checkout`
- `GET /api/residents`
- `POST /api/residents`
- `GET /api/keys`
- `POST /api/keys`
- `POST /api/keys/return`
- `POST /api/keys/delete`
- `GET /api/sync/status`
- `POST /api/sync/run`

## Arquitetura

- `internal/domain`: entidades e regras centrais.
- `internal/usecase`: casos de uso e portas.
- `internal/repository`: adapter SQLite.
- `internal/handler`: adapter HTTP REST.
- `internal/sync`: worker de sincronizacao.
- `internal/google`: adapter Google Sheets.
- `internal/config`: configuracao por ambiente.
- `internal/database`: abertura do SQLite e migrations.
