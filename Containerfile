# Stage 1: build the SvelteKit static output
FROM node:22-alpine AS web
WORKDIR /src/web
COPY web/package.json web/pnpm-lock.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm build

# Stage 2: build the Go binary with embedded frontend
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# populate embedded static dir from the web build
RUN rm -rf internal/web/static && mkdir -p internal/web/static && cp -r web/build/. internal/web/static/ && touch internal/web/static/.gitkeep
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/budget-agent ./cmd/budget-agent

# Stage 3: minimal runtime image
FROM gcr.io/distroless/static:nonroot
COPY --from=build /out/budget-agent /budget-agent
USER 65532:65532
ENTRYPOINT ["/budget-agent"]
EXPOSE 8080
