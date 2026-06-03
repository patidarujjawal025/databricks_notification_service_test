FROM 157529275398.dkr.ecr.ap-southeast-1.amazonaws.com/ci-libraries/docker/library/golang:1.21-alpine AS build-env
ARG BITBUCKET_SSH_PRIVATE_KEY
ARG TARGETOS
ARG TARGETARCH

ENV GOPRIVATE="github.com/swiggy-private/*"
ENV GO111MODULE=on
ENV GOPROXY=https://proxy.golang.org,direct
ENV GOSUMDB=off
ENV GIT_SSL_NO_VERIFY=1

RUN apk update && apk add --update --no-cache git openssh ca-certificates make

RUN mkdir -p ~/.ssh && umask 0077 && echo "${BITBUCKET_SSH_PRIVATE_KEY}" > ~/.ssh/id_rsa \
    && git config --global url."git@github.com:".insteadOf https://github.com/ \
    && git config --global url."git@github.com:swiggy-private/".insteadOf https://github.com/swiggy-private/ \
    && ssh-keyscan github.com >> ~/.ssh/known_hosts

RUN echo "StrictHostKeyChecking no" > ~/.ssh/config

WORKDIR /go/src
COPY Databricks_Notification_Service .

RUN go mod download
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o main cmd/main.go

# Minimal Lambda runtime image
FROM 157529275398.dkr.ecr.ap-southeast-1.amazonaws.com/ci-libraries/amazon/aws-lambda-provided:al2
COPY --from=build-env /go/src/main /opt/main
COPY --from=build-env /go/src/config.yaml /opt/config.yaml

WORKDIR /opt
ENTRYPOINT ["/opt/main"]