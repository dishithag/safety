# syntax=docker/dockerfile:1

# CMD selects which ./cmd/<CMD> package to build (e.g. summarizer, analyticsapi).
FROM golang:1.25 AS build
ARG CMD=summarizer
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/app ./cmd/${CMD}

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/app /app
ENTRYPOINT ["/app"]


Now it does:
COPY go.mod go.sum ./
RUN go mod download
Why this makes sense:
go.mod declares dependencies
go.sum pins the dependency checksums
copying both before go mod download makes the container build more consistent with what you tested locally
