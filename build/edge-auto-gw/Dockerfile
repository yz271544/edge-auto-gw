FROM golang:1.16-alpine3.13 AS builder

ARG GO_LDFLAGS
ARG TARGETARCH

# set goproxy
# using the go module
RUN go env -w GO111MODULE=on
RUN set -eux && sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories
# source
#RUN apk update && apk add gcc && \
#    apk --no-cache add build-base linux-headers binutils-gold

WORKDIR /code
COPY . .

RUN GOOS=linux GOARCH=$TARGETARCH go build -v -o /usr/local/bin/edge-auto-gw -ldflags "$GO_LDFLAGS -w -s" \
./server/cmd/edge-auto-gw/server.go


FROM alpine:3.13

#RUN apk update

COPY --from=builder /usr/local/bin/edge-auto-gw /usr/local/bin/edge-auto-gw

ENTRYPOINT ["edge-auto-gw"]
