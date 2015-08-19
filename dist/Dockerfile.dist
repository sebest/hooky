FROM scratch

ADD dist/ca-certificates.crt /etc/ssl/certs/
ADD hooky-build.tar.gz /

EXPOSE 8000

CMD ["/hookyd"]