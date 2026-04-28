FROM node:22-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json* ./
RUN npm install
COPY web/ ./
RUN npm run build

FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN mkdir -p cmd/kubeshell-web/web/dist && cp -R web/dist/* cmd/kubeshell-web/web/dist/
RUN CGO_ENABLED=0 go build -o /out/kubeshell-web ./cmd/kubeshell-web

FROM cgr.dev/chainguard/static:latest
USER 65532:65532
COPY --from=build /out/kubeshell-web /kubeshell-web
EXPOSE 8080
ENTRYPOINT ["/kubeshell-web"]
