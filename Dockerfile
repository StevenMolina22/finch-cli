FROM golang:1.25-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/finch .

FROM alpine:3.22

RUN adduser -D -H finch

COPY --from=build /out/finch /usr/local/bin/finch

USER finch
EXPOSE 3333

CMD ["sh", "-c", "finch mcp --transport http --addr :${PORT:-3333}"]
