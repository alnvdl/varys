FROM golang:1.24-bookworm AS build

COPY . /varys
WORKDIR /varys
RUN go build -o ./varys ./cmd/varys/main.go

FROM ubuntu:jammy AS final

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update
RUN apt-get upgrade -qq -y
RUN apt-get install -qq -y ca-certificates

COPY --from=build /varys/varys /varys
CMD ["/varys"]
