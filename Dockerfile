FROM golang:1.17-alpine3.14 as base

WORKDIR /app

COPY go.mod ./

COPY go.sum ./

RUN go mod download

COPY main.go ./

COPY bully/ ./bully/

COPY cluster/ ./cluster/

COPY utils ./utils/

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /knh

FROM gcr.io/distroless/static

COPY --from=base /knh /knh

ENTRYPOINT [ "/knh" ]
