FROM golang:1.16-buster AS build
COPY . /src/github.com/arussellsaw/budgie
RUN cd /src/github.com/arussellsaw/budgie && CGO_ENABLED=0 go build -o budgie -mod=vendor

FROM alpine:latest AS final

WORKDIR /app

COPY --from=build /src/github.com/arussellsaw/budgie/budgie /app/
COPY --from=build /src/github.com/arussellsaw/budgie/tmpl /app/tmpl
COPY --from=build /src/github.com/arussellsaw/budgie/static /app/static
COPY --from=build /src/github.com/arussellsaw/budgie/build /app/build

EXPOSE 8080

ENTRYPOINT /app/budgie