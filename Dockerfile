FROM ubuntu:latest
LABEL authors="wuyus"

ENTRYPOINT ["top", "-b"]