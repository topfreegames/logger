FROM alpine:3.6

RUN apk add --no-cache librdkafka-dev && \
    mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2

COPY . /

CMD ["/opt/logger/sbin/logger"]
EXPOSE 1514 8088
