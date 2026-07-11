FROM golang:1.26-alpine AS builder

ARG GIT_COMMIT
ARG GIT_TAG

ENV APP_GIT_COMMIT=${GIT_COMMIT}
ENV APP_GIT_TAG=${GIT_TAG}

WORKDIR /

RUN apk add -U --no-cache ca-certificates

COPY services/api/dist/api .

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /api .

USER nonroot:nonroot
EXPOSE 9000

CMD [ "./api" ]
