FROM golang:1.9

RUN mkdir /app

ENV GOOGLE_API_KEY $GOOGLE_API_KEY
ENV MYSQL_HOST $MYSQL_HOST
ENV MYSQL_USER $MYSQL_USER
ENV MYSQL_PASSWORD $MYSQL_PASSWORD
ENV MYSQL_DATABASE $MYSQL_DATABASE

WORKDIR /app

ADD app /app/

RUN go get github.com/julienschmidt/httprouter
RUN go get github.com/satori/go.uuid
RUN go get -u github.com/go-sql-driver/mysql
RUN go get github.com/mattes/migrate
RUN go get github.com/mattes/migrate/database/mysql
RUN go get github.com/mattes/migrate/source/file
RUN go get googlemaps.github.io/maps
RUN go get golang.org/x/net/context

RUN go build -o shortest_route .

ENTRYPOINT ["./shortest_route"]
EXPOSE 8080