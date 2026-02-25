FROM --platform=$BUILDPLATFORM golang:1.26.0-trixie AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
  go build -trimpath -ldflags="-s -w" -o /out/alertreplay ./cmd/alertreplay

FROM gcr.io/distroless/static-debian13:nonroot

COPY --from=builder /out/alertreplay /usr/local/bin/alertreplay

ENTRYPOINT ["/usr/local/bin/alertreplay"]
