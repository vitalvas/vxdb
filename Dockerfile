FROM golang:latest
WORKDIR /app
COPY . .
RUN go build -o vxdb ./...

FROM ubuntu:latest
RUN go build -o main .
COPY --from=0 /app/vxdb /bin/vxdb
ENV DB_PATH="/data"
CMD ["/bin/vxdb"]
