FROM golang:1.12-alpine AS build

ENV CGO_ENABLED 0

WORKDIR /app
COPY . .

RUN apk add --no-cache git
RUN go install -v .

FROM scratch
COPY --from=build /go/bin/backend-challenge /bin/backend-challenge
ENTRYPOINT ["/bin/backend-challenge"]
CMD ["-listen-addr", ":8080"]
