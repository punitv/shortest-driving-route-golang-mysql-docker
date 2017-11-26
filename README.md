## Application: Shortest drinving route APIs using Google Maps APIs

## Docker: Golang web + MySQL db


### Run containers
```
docker-compose build
docker-compose up -d
docker run -it -p 8080:8080 -e "MYSQL_USER=webuser" -e "MYSQL_HOST=192.168.99.100" -e "MYSQL_DATABASE=go_app_db" -e "MYSQL_PASSWORD=somepass" -e "PWD=/app" -e "GOOGLE_API_KEY=YOUR_API_KEY" shortestdrivingroutegolangmysqldocker_web
```
### Container logs
```
docker-compose logs
```