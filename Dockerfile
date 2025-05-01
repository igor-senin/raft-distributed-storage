FROM ubuntu:latest

RUN apt-get update && apt-get install -y curl

COPY raft-distributed-storage /opt

EXPOSE 8786

ENTRYPOINT ["/opt/raft-distributed-storage"]
