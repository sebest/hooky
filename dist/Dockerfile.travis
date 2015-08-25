FROM scratch

ADD dist/ca-certificates.crt /etc/ssl/certs/
ADD hooky /
ADD hookyd /

EXPOSE 8000

CMD ["/hookyd"]