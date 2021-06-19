FROM ubuntu:latest
COPY vxdb /bin/vxdb
ENV DB_PATH="/data"
CMD ["/bin/vxdb"]
