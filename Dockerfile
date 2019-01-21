FROM golang:latest

COPY ./default.tmpl /templates/default.tmpl
# build directories


RUN mkdir -p /go/src/github.com/vu-long/alertmanager-bot
ADD . /go/src/github.com/vu-long/alertmanager-bot

# Go dep!
# RUN go get -u github.com/golang/dep/...
# RUN dep init -v
# RUN dep ensure -v -vendor-only

# for production
WORKDIR /go/src/github.com/vu-long/alertmanager-bot
RUN make build
EXPOSE 8080
ENTRYPOINT ["/go/src/github.com/vu-long/alertmanager-bot/alertmanager-bot"]


# for developing
# RUN go get -u github.com/canthefason/go-watcher
# RUN go install github.com/canthefason/go-watcher/cmd/watcher

# EXPOSE 8080:8080
# ENTRYPOINT ["watcher -run github.com/vu-long/alertmanager-bot/cmd/alertmanager-bot -watch github.com/vu-long/alertmanager-bot"]