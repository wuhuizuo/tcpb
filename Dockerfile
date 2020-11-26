# container for build
FROM golang:latest as go

COPY . /workspace
RUN cd /workspace && make

# containter for run
FROM debian:buster-slim

COPY --from=go /workspace/tcpbs /usr/bin/
COPY --from=go /workspace/tcpbc /usr/bin/

RUN chmod +x /usr/bin/tcpbs && tcpbs -h 
RUN chmod +x /usr/bin/tcpbc && tcpbc -h

EXPOSE 80

ENTRYPOINT [ "/usr/bin/tcpbs", "-port", "80" ]