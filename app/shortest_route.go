package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	"github.com/julienschmidt/httprouter"
	"github.com/satori/go.uuid"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/mattes/migrate"
	"github.com/mattes/migrate/database/mysql"
	_ "github.com/mattes/migrate/source/file"
	"googlemaps.github.io/maps"
	"golang.org/x/net/context"
)

var (
	port = flag.String("port", ":8080", "http service address")
	apiKey = os.Getenv("GOOGLE_API_KEY")
)

// Global sql.DB to access the database by all handlers
var db *sql.DB 
var err error

type routePath [][]string
type routePathPoint []string

type responseToken struct {
	Token string `json:"token"`
}

type shortestRoute struct {
	Status        string     `json:"status"`
	Path          [][]string `json:"path,omitempty"`
	TotalDistance int64      `json:"total_distance,omitempty"`
	TotalTime     int64      `json:"total_time,omitempty"`
	Error         string     `json:"error,omitempty"`
}

func main() {
	flag.Parse()

	db, err = sql.Open("mysql", os.Getenv("MYSQL_USER") + ":" + os.Getenv("MYSQL_PASSWORD") + "@tcp(" + os.Getenv("MYSQL_HOST") + ")/" + os.Getenv("MYSQL_DATABASE") + "?charset=utf8")
	if err != nil {
		log.Fatal(err.Error())
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatal(err.Error())
	}

	driver, _ := mysql.WithInstance(db, &mysql.Config{})
	m, err := migrate.NewWithDatabaseInstance(
		"file://" + os.Getenv("PWD") + "/migrations",
		"mysql", 
		driver,
	)

	m.Steps(2)
	if err != nil {
		log.Fatal(err.Error())
	}

	r := httprouter.New()
	r.GET("/route/:token", routeHandler)
	r.POST("/route", pathHandler)
	err = http.ListenAndServe(*port, r)
	if err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

func pathHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params){
	routePathArr := routePath{}
	routePathPointArr := routePathPoint{}

	routePathJson, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error while reading r.Body: %s\n", err)
		http.Error(w, "{\"error\":\""+err.Error()+"\"}", 200)
		return
	}
	err = json.Unmarshal(routePathJson, &routePathArr)
	if err != nil {
		log.Printf("Decoding error: %s\n", err)
		http.Error(w, "{\"error\":\""+err.Error()+"\"}", 200)
		return
	}

	if len(routePathArr) < 2 {
		log.Printf("Route start & dropoff required\n")
		http.Error(w, "{\"error\":\"Route start & dropoff required\"}", 200)
		return
	} else if len(routePathArr) > 26 {
		log.Printf("Max dropoff location is 25 (the limit can increased as required)\n")
		http.Error(w, "{\"error\":\"Max dropoff location is 25 (the limit can increased as required)\"}", 200)
		return
	}
	for _, loc := range routePathArr {
		if len(loc) != 2 {
			log.Printf("Incorrect latitude longitude\n")
			http.Error(w, "{\"error\":\"Incorrect latitude longitude\"}", 200)
			return
		}
		routePathPointArr = append(routePathPointArr, strings.Join(loc[:], ","))
	}

	token := uuid.NewV4().String()
	tokenObj := responseToken{
		Token: token,
	}

	tokenJson, err := json.Marshal(tokenObj)
	if err != nil {
		log.Printf("Token encoding error: %s\n", err)
		http.Error(w, "{\"error\":\""+err.Error()+"\"}", 200)
		return
	}

	_, err = db.Exec("INSERT INTO `travel_details`(path, token, status) VALUES (?, ?, ?)", routePathJson, token, 0)
	if err != nil {
		log.Printf("Error while inserting token row: %s\n", err)
		http.Error(w, "{\"error\":\""+err.Error()+"\"}", 200)
		return
	}

	go processRoute(token, routePathPointArr);

	w.Header().Set("Content-Type", "application/json")
	w.Write(tokenJson)
	return
}

func processRoute(token string, routePathPointArr routePathPoint) {
	log.Printf("In process: %s\n", token)

	var response_error string
	var total_distance, total_time int64 = 0, 0

	stmt, err := db.Prepare("UPDATE `travel_details` SET status=?, total_distance=?, total_time=?, response_error=? WHERE token=?")
	if err != nil {
		log.Printf("Error while updating token row: %s\n", err)
		return
	}

	c, _ := maps.NewClient(maps.WithAPIKey(apiKey))
	dmreq := &maps.DistanceMatrixRequest{
		Origins:       routePathPointArr[:len(routePathPointArr)-1],
		Destinations:  routePathPointArr[1:],
		Language:      `en`,
	}
	dmresp, err := c.DistanceMatrix(context.Background(), dmreq)
	if err != nil {
		response_error = string(err.Error())
		log.Printf("Distance Matrix Error: %s: %+v", token, err)
		_, err = stmt.Exec(-1, 0, 0, response_error, token)
		if err != nil {
			log.Printf("Error while updating token row: %s: %s\n", token, err)
		}
		return
	}

	for i, row := range dmresp.Rows {
		if row.Elements[i].Status == "OK" {
			total_distance += int64(row.Elements[i].Distance.Meters)
			total_time += int64(row.Elements[i].Duration / time.Second)
		} else {
			response_error = row.Elements[i].Status
			log.Printf("Distance Matrix Error: %s: %+v", token, err)
			_, err = stmt.Exec(-1, 0, 0, response_error, token)
			if err != nil {
				log.Printf("Error while updating row: %s: %s\n", token, err)
			}
			return
		}
	}

	_, err = stmt.Exec(1, total_distance, total_time, "", token)
	if err != nil {
		log.Printf("Error while updating token row: %s\n", err)
	}
	log.Printf("Process completed: %s\n", token)
}

func routeHandler(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	routeResult := shortestRoute{}
	routePathArr := routePath{}

	token := p.ByName("token")
	if token == "" {
		log.Printf("Token empty\n")
		routeResult = shortestRoute {
							Status: "failure",
							Error: "Invalid Token",
						}
	}
	log.Printf("token received: %s\n", token)

	var status int
	var routePathJson, response_error string
	var total_distance, total_time int64

	err := db.QueryRow("SELECT path, status, total_distance, total_time, response_error FROM `travel_details` WHERE token=?", token).Scan(&routePathJson, &status, &total_distance, &total_time, &response_error)
	if err != nil {
		log.Printf("No row found: %s\n", err.Error())
		routeResult = shortestRoute {
							Status: "failure",
							Error: "Invalid Token",
						}
	}

	if status == 0 {
		routeResult = shortestRoute {
							Status: "in progress",
						}
	} else if status == -1 {
		routeResult = shortestRoute {
							Status: "failure",
							Error: response_error,
						}
	} else if status == 1 {
		err = json.Unmarshal([]byte(routePathJson), &routePathArr)
		if err != nil {
			log.Printf("Decoding error: %s\n", err)
			routeResult = shortestRoute {
							Status: "failure",
							Error: err.Error(),
						}
		} else {
			routeResult = shortestRoute {
							Status: "success",
							Path: routePathArr,
							TotalDistance: total_distance,
							TotalTime: total_time,
						}
		}
	}

	routeJson, err := json.Marshal(routeResult)
	if err != nil {
		log.Printf("Error: %s: %s", token, err)
		http.Error(w, "{\"status\":\"failure\", \"error\":\""+err.Error()+"\"}", 200)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(routeJson)
	return
}