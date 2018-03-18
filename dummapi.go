package main

import (
	//"bytes"
	"encoding/json"
	"fmt"
	"io"
	//"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
)

type CounterInfo struct {
	Current int `json:"current"`
	To      int `json:"to"`
}

var counterList map[string]*CounterInfo
var chanList map[string]chan bool
var ip string
var redisClient *redis.Client

func main() {

	addrs, _ := net.InterfaceAddrs()

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ip = ipnet.IP.String()
			}
		}
	}

	redisServer := os.Getenv("REDISSERVER")
	redisClient = redis.NewClient(&redis.Options{
		Addr			:redisServer+":6379",
		Password	: "",
		DB				: 0,
	})

	counterList = make(map[string]*CounterInfo)
	chanList = make(map[string]chan bool)

	urlParser := map[string]interface{}{
		"/": defaultHandler,
	}

	counterurlPrefix := "/counter/"
	counterurlParser := map[string]interface{}{
		"/":                  counterHandler,
		"/{counterID}/":      counterInfoHandler,
		"/{counterID}/stop/": counterStopHandler,
	}

	r := mux.NewRouter()
	s := r.PathPrefix(counterurlPrefix).Subrouter()

	for url, handler := range urlParser {
		r.HandleFunc(url, handler.(func(http.ResponseWriter, *http.Request)))
	}

	for counterurl, handler := range counterurlParser {
		s.HandleFunc(counterurl, handler.(func(http.ResponseWriter, *http.Request)))
	}

	http.ListenAndServe(":10000", r)
}

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		hostName, err := os.Hostname()

		if err == nil {
			w.Write([]byte(hostName))
			w.Write([]byte("\n"))
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func counterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		var counterIDs string
		keys, err := redisClient.Keys("*").Result()

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		for _, key := range keys {
			counterIDs += key + "\n"
		}

		w.Write([]byte(counterIDs))
	} else if r.Method == http.MethodPost {

		toQuery := r.URL.Query().Get("to")

		if toQuery == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		toValue, err := strconv.Atoi(toQuery)

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		counterID := counterGenerator(toValue) + "\n"
		w.Write([]byte(counterID))
	}
}

func counterInfoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method==http.MethodGet {
		v := mux.Vars(r)
		counterID := string(v["counterID"])
		counter := counterList[counterID]

		if counter == nil {
			host, err := redisClient.Get(counterID).Result()

			if err != nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			redirect(w, r, host)
			return
		}

		js, err := json.Marshal(counter)

		if err != nil {
			fmt.Println(err)
		}

		w.Write(js)
		w.Write([]byte("\n"))
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func counterStopHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method==http.MethodPost {
		v := mux.Vars(r)
		counterID := string(v["counterID"])
		q := chanList[counterID]

		if q == nil {
			host, err := redisClient.Get(counterID).Result()

			if err != nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			url := "http://" + host + ":10000/counter/"+counterID+"/stop/"
			res, err := http.Post(url, "text/plain", nil)

			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			w.WriteHeader(res.StatusCode)
			//redirect(w, r, host)
			return
		}
		q<-true
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func counterGenerator(to int) string {
	newCounter := new(CounterInfo)
	uuid := genUUID()

	for {

		// should be "TEST AND SET LOCK" atomic operation but it is NOT!!!!!!!!!!!!!!!!!
		ok, _ := redisClient.SetNX(uuid, ip, 0).Result()
		if ok == true {
			break
		}

		uuid = genUUID()
	}

	newCounter.Current = 0
	newCounter.To = to

	counterList[uuid] = newCounter
	q := make(chan bool)
	chanList[uuid] = q

	go counterIncreaser(uuid, q)

	return uuid
}

func genUUID() (uuid string) {

	dummy := make([]byte, 16)
	_, err := rand.Read(dummy)
	if err != nil {
		return
	}
	uuid = fmt.Sprintf("%X-%x-%x-%x-%x", dummy[0:4], dummy[4:6], dummy[6:8], dummy[8:10], dummy[10:])

	return
}

func counterIncreaser(counterID string, q chan bool) {
	var counter *CounterInfo

	ticker := time.NewTicker(1 * time.Second)

	defer ticker.Stop()
	defer delete(counterList, counterID)
	defer delete(chanList, counterID)
	defer redisClient.Del(counterID)

	for {
		select {
		case <-ticker.C:
			counter = counterList[counterID]
			counter.Current++

			if counter.Current == counter.To {
				return
			}

		case <-q:
			return
		}
	}
}

func redirect(w http.ResponseWriter, r *http.Request, host string) {
	/*
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	*/

	url := fmt.Sprintf("http://%s:10000/%s", host, r.RequestURI)
	//req, err := http.NewRequest(r.Method, url, bytes.NewReader(body))
	req, err := http.NewRequest(r.Method, url, nil)
	req.Header = r.Header

	httpClient := http.Client{}
	response, err := httpClient.Do(req)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	io.Copy(w, response.Body)
	w.WriteHeader(response.StatusCode)
	return

	defer response.Body.Close()
}
