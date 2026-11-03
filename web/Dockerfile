FROM golang:1.27.3-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /nav .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /nav /nav
USER 65532:65532
ENTRYPOINT ["/nav"]
CMD ["-config", "/etc/nav/config.json"]
