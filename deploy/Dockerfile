FROM golang:1.24 AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/router ./cmd/router \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/mockvllm ./cmd/mockvllm \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/bench ./cmd/bench

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/ /usr/local/bin/
EXPOSE 9000
ENTRYPOINT ["/usr/local/bin/router"]
