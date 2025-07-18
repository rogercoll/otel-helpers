FROM golang:1.24-bookworm AS builder

WORKDIR /usr/src/app/

COPY go.mod go.mod
COPY go.sum go.sum
COPY main.go main.go
COPY agent.go agent.go

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -o opamp_client .

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /usr/src/app/

COPY --from=builder /usr/src/app/opamp_client/ ./

ENTRYPOINT [ "./opamp_client" ]
