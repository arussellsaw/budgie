FROM golang:1.13-buster AS build
COPY . /src/github.com/arussellsaw/youneedaspreadsheet
RUN cd /src/github.com/arussellsaw/youneedaspreadsheet && CGO_ENABLED=0 go build -o youneedaspreadsheet -mod=vendor

FROM alpine:latest AS final

WORKDIR /app

COPY --from=build /src/github.com/arussellsaw/youneedaspreadsheet/youneedaspreadsheet /app/
COPY --from=build /src/github.com/arussellsaw/youneedaspreadsheet/tmpl /app/tmpl
COPY --from=build /src/github.com/arussellsaw/youneedaspreadsheet/static /app/static

EXPOSE 8080

ENTRYPOINT /app/youneedaspreadsheet