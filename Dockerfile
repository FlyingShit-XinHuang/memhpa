FROM 192.168.1.113/tenxcloud/go:1.7-dev

WORKDIR $GOPATH/src/memhpa

ADD . $GOPATH/src/memhpa

RUN CGO_ENABLED=0 go build \
  && ls | egrep -v 'memhpa' | xargs rm -rf

ENTRYPOINT ["./memhpa"]

CMD ["--logtostderr=true"]