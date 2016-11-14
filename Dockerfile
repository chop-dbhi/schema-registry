FROM alpine

COPY ./dist/linux-amd64/schema-registry /schema-registry

RUN mkdir /data

ENTRYPOINT ["/schema-registry", "-http", "0.0.0.0:8080", "-db", "/data/registry.db"]

VOLUME /data

EXPOSE 8080
