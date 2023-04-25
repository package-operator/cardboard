FROM scratch

WORKDIR /
COPY passwd /etc/passwd
COPY main /

USER "noroot"

ENTRYPOINT ["/main"]
