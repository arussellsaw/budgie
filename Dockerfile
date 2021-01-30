FROM golang:1.13-buster AS build
COPY . /src/github.com/arussellsaw/banksheets
RUN cd /src/github.com/arussellsaw/banksheets && CGO_ENABLED=0 go build -o banksheets -mod=vendor

FROM alpine:latest AS final

WORKDIR /app

COPY --from=build /src/github.com/arussellsaw/banksheets/banksheets /app/
COPY --from=build /src/github.com/arussellsaw/banksheets/static /app/static
COPY --from=build /src/github.com/arussellsaw/banksheets/tmpl /app/tmpl

EXPOSE 8080

ENTRYPOINT /app/banksheets