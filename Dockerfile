FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod ./
COPY cmd/ ./cmd/
COPY internal/ ./internal/
RUN CGO_ENABLED=0 go build -o /out/gofi-mcp ./cmd/gofi-mcp

FROM scratch
COPY --from=builder /out/gofi-mcp /gofi-mcp
ENTRYPOINT ["/gofi-mcp"]
