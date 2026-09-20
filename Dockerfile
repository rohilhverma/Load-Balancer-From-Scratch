FROM golang:1.27-alpine AS build

WORKDIR /src
COPY go.mod ./
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/hrl .

FROM scratch
COPY --from=build /out/hrl /hrl
USER 65532:65532
ENTRYPOINT ["/hrl"]
