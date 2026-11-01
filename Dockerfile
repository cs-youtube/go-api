FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./

FROM build AS test
RUN CGO_ENABLED=0 go test ./...

FROM build AS binary
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /api .

FROM scratch
COPY --from=binary /api /api
USER 65534
ENTRYPOINT ["/api"]
