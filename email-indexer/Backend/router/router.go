package main

import (
	"encoding/json"
	"fmt"
	"indexer/zincsearch"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

var waitRequest sync.WaitGroup

type RespHandler struct {
    name    string
}

var host = ":3000"

func main() {
    r := chi.NewRouter()
    r.Use(cors.Handler(cors.Options{
        // AllowedOrigins:   []string{"https://foo.com"}, // Use this to allow specific origin hosts
        AllowedOrigins:   []string{"https://*", "http://*"},
        // AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
        AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
        AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
        ExposedHeaders:   []string{"Link"},
        AllowCredentials: false,
        MaxAge:           300, // Maximum value not ignored by any of major browsers
      }))
    r.Get("/", verifyIndex)
    r.Post("/get_emails", getEmails)
    fmt.Printf("Api deployed on port %s", host)
    http.ListenAndServe(host, r)
}

func verifyIndex(w http.ResponseWriter, r *http.Request){
    promise := make(chan *http.Response)
    go func ()  {
        waitRequest.Add(1)
        answer := zinc.VerifyIndex("enron")
        promise <- answer
    }()
    answer := <- promise
    if answer.StatusCode != http.StatusOK{
        w.Write([]byte(fmt.Sprintf("Connection succesfully")))
    }
}

func getEmails(w http.ResponseWriter, r *http.Request){
    body, err := io.ReadAll(r.Body)
    if err != nil {
        w.WriteHeader(http.StatusBadRequest)
        return
    }
    
    var jsonQuery struct{
        Text string `json:"text"`
        OrderBy string `json:"order_by"`
        Step int `json:"step"`
        Start int `json:"start"`
    }
    err = json.Unmarshal(body, &jsonQuery)
    if jsonQuery.Step == 0 {
        jsonQuery.Step = 10
    }

    res, err := zinc.Query(jsonQuery.Text, jsonQuery.Start, jsonQuery.Step, jsonQuery.OrderBy)
	if err != nil {
		log.Fatal(err)
	}

    if err != nil {
        w.WriteHeader(http.StatusBadRequest)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(res.Hits)
}
