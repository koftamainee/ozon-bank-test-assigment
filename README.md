## Development Quickstart

Requirements: Go 1.26+, Docker + Docker Compose

If you using tmux, you can just launch project in tmux-sessionizer, or just do this
```sh
cp .env.example .env      # fill in passwords / JWT secret
make dev                  # starts Postgres, applies migrations + app with hot reload (air)
```

`make dev` brings up three tmux windows: shell, database in docker and app with hot reload.
The app listens on `http://localhost:8080`.

Run `make db-reset` to wipe the dev volume and re-run migrations.

### Without tmux

```sh
docker compose -f docker-compose.dev.yaml up -d
export $(grep -v '^#' .env | xargs)
go run ./cmd/api # or just air
```

## Prod env

```sh
cp .env.example .env
docker compose up --build -d
```
The compose file wires `ENV`, `JWT_SECRET`, `JWT_TTL`, `JWT_COOKIE_SECURE` through.
In `ENV=prod` the app refuses to start without a strong `JWT_SECRET` (>= 32 bytes) and
`JWT_COOKIE_SECURE=true`.

## Health endpoints

kubernetes-style

- `GET /healthz` - Liveness - process is up
- `GET /readyz` - Readiness - process is up && deps are up

## API

- `POST /auth/login` `{"username": "..."}` - sets session cookie.
- `POST /auth/logout` - clears the session cookie.
- `POST /graphql` - GraphQL API. Dev also allows GET with
  `?query=...`. `ENV=prod` disables introspection and the GET transport.
- WebSocket `graphql-ws` subscription protocol over `/graphql` for `commentAdded`.

The schema lives in `internal/graphql/schema.graphql` and is regenerated with `make gqlgen`.

### Testing flow (curl)

Start the app (`make dev`) and save the session cookie to a file:

```sh
curl -s -c /tmp/cookies.txt -X POST http://localhost:8080/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"alice"}'
# {"ok":true}
```

Client doesnt send password. This is for demo purposes to not use bcrypt or similar stuff

REST errors use the envelope `{"code":"...","error":"..."}`:

```sh
curl -s -X POST http://localhost:8080/auth/login -H 'Content-Type: application/json' \
  -d '{"username":"bad user!"}'          # 422 {"code":"VALIDATION_ERROR","error":"invalid username"}
curl -s -X POST http://localhost:8080/auth/login -H 'Content-Type: application/json' \
  -H 'Origin: http://evil.example' -d '{"username":"alice"}'    # 403 cross-site request
```

#### GraphQL

Mutations need the auth session cookie from login

```sh
curl -s -b /tmp/cookies.txt -X POST http://localhost:8080/graphql \
  -H 'Content-Type: application/json' \
  -d '{"query":"mutation { createPost(input:{title:\"hello\",body:\"world\"}) { id title body } }"}'
# {"data":{"createPost":{"id":"1","title":"hello","body":"world"}}}

curl -s -X POST http://localhost:8080/graphql -H 'Content-Type: application/json' \
  -d '{"query":"mutation { createPost(input:{title:\"x\",body:\"y\"}) { id } }"}'
# {"errors":[{"message":"unauthorized",...,"extensions":{"code":"UNAUTHORIZED"}}],"data":null}
```

Queries are public:

```sh
curl -s -X POST http://localhost:8080/graphql -H 'Content-Type: application/json' \
  -d '{"query":"{ posts(first: 10) { items { id title body } next } }"}'

curl -s -X POST http://localhost:8080/graphql -H 'Content-Type: application/json' \
  -d '{"query":"{ post(id: 1) { id title body } }"}'
```

Comments - nested replies via `parentID`, listing is depth-first:

```sh
curl -s -b /tmp/cookies.txt -X POST http://localhost:8080/graphql \
  -H 'Content-Type: application/json' \
  -d '{"query":"mutation { createComment(input:{postID: 1, body:\"first comment\"}) { id depth } }"}'
# {"data":{"createComment":{"id":"1","depth":0}}}

curl -s -b /tmp/cookies.txt -X POST http://localhost:8080/graphql \
  -H 'Content-Type: application/json' \
  -d '{"query":"mutation { createComment(input:{postID: 1, parentID: 1, body:\"reply\"}) { id depth } }"}'
# {"data":{"createComment":{"id":"2","depth":1}}}

curl -s -X POST http://localhost:8080/graphql -H 'Content-Type: application/json' \
  -d '{"query":"{ comments(postID: 1, first: 10) { items { id body depth } next } }"}'
```

Pagination: `first` defaults to 20 (clamped to [1..100]); use the opaque `next`
cursor with `after` for the next page:

```sh
curl -s -X POST http://localhost:8080/graphql -H 'Content-Type: application/json' \
  -d '{"query":"{ posts(first: 2) { items { id title } next } }"}'
# {"data":{"posts":{"items":[{"id":"4","title":"four"},{"id":"3","title":"three"}],"next":"MjAyNi0wOC0wN..."}}}

curl -s -X POST http://localhost:8080/graphql -H 'Content-Type: application/json' \
  -d '{"query":"{ posts(first: 2, after: \"MjAyNi0wOC0wN...\") { items { id title } next } }"}'
# last page has "next": null
```

Toggle comments on a post:

```sh
curl -s -b /tmp/cookies.txt -X POST http://localhost:8080/graphql \
  -H 'Content-Type: application/json' \
  -d '{"query":"mutation { setCommentsAllowed(postID: 1, allowed: false) { id commentsAllowed } }"}'
# {"data":{"setCommentsAllowed":{"id":"1","commentsAllowed":false}}}

curl -s -b /tmp/cookies.txt -X POST http://localhost:8080/graphql \
  -H 'Content-Type: application/json' \
  -d '{"query":"mutation { createComment(input:{postID: 1, body:\"nope\"}) { id } }"}'
# {"errors":[...,"extensions":{"code":"COMMENTS_DISABLED"}}],"data":null}
```

#### Logout

```sh
curl -s -b /tmp/cookies.txt -X POST http://localhost:8080/auth/logout -w '%{http_code}'
# 204 - cookie cleared

curl -s -c /tmp/cookies.txt -X POST http://localhost:8080/auth/login \
  -H 'Content-Type: application/json' -d '{"username":"alice"}'
# {"ok":true} - new session cookie
```

#### Subscriptions

`commentAdded` runs over the `graphql-ws` subprotocol on `/graphql`. Subscribing
is public; publishing a comment needs the session cookie.

Terminal 1 - subscribe to post 1:

```sh
wscat -s graphql-ws -c ws://localhost:8080/graphql
# {"type":"connection_init"}
# < {"type":"connection_ack"}
# < {"type":"ka"}
# {"type":"start","id":"1","payload":{"query":"subscription { commentAdded(postID: 1) { id body } }"}}
```

Terminal 2 - publish a comment on that post:

```sh
curl -s -b /tmp/cookies.txt -X POST http://localhost:8080/graphql \
  -H 'Content-Type: application/json' \
  -d '{"query":"mutation { createComment(input:{postID: 1, body:\"sub test\"}) { id body } }"}'
```

Back in terminal 1 the event arrives as a `data` frame:

```json
{"payload":{"data":{"commentAdded":{"id":"1","body":"sub test"}}},"id":"1","type":"data"}
```
