package zinc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

const(
  nameIndexer = "enron"
  configIndexer = 
  `{
      "name": "enron",
      "storage_type": "disk",
      "shard_num": 2,
      "mappings": {
        "properties": {
			"date":{
				"type": "date",
				"index": true,
				"store": false
			},
			"directory":{
				"type": "text",
				"index": false,
				"store": false
			},
			"from": {
				"type": "text",
				"index": true,
				"store": false
			},
			"to": {
				"type": "text",
				"index": true,
				"store": false
			},
			"subject": {
				"type": "text",
				"index": true,
				"store": false,
				"highlightable": true
			},
			"content": {
				"type": "text",
				"index": true,
				"store": false,
				"highlightable": true
			}
        }
      }
    }`
    host = "localhost:4080"
)

type (
	Shit struct {
		Date	string
		Id      string `json:"_id"`
		Directory string
		Content string
		From    string
		To      string
		Subject string
	}
	Ssource struct {
		Source Shit `json:"_source"`
	}
	SQty struct {
		Value int
	}
	Shits struct {
		Total SQty
		Hits  []Ssource
	}
	Chits struct {
		Hits Shits
	}
	serializerResponse struct {
		Count int `json:"record_count"`
	}
)

func configRequest(request *http.Request){
	request.SetBasicAuth("admin", "Complexpass#123")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/81.0.4044.138 Safari/537.36")
}

func CreateIndex() (err error) {
	fmt.Println("Creating index...")
	request, err := http.NewRequest("POST", "http://"+host+"/api/index", strings.NewReader(configIndexer))
	if err != nil {
		return err
	}
  	configRequest(request)
	request.Close = true
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	fmt.Println(response.StatusCode)
	if response.StatusCode != http.StatusOK {
		body, err := io.ReadAll(response.Body)
		if err != nil {
			return err
		}
		return errors.New(string(body))
	}
	return nil
}

func DeleteIndex() (err error) {
	fmt.Println("Deleting Index...'")
	request, err := http.NewRequest("DELETE",fmt.Sprintf("http://"+host+"/api/index/%s",nameIndexer), nil)
	if err != nil {
		return err
	}
 	configRequest(request)
	request.Close = true
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	fmt.Println((response.StatusCode))
	if response.StatusCode != http.StatusOK {
		body, err := io.ReadAll(response.Body)
		if err != nil {
			return err
		}
		return errors.New(string(body))
	}
	return nil
}

func VerifyIndex(indexerName string) (*http.Response){
	request, err := http.NewRequest("HEAD", fmt.Sprintf("http://"+host+"/api/index"+nameIndexer), nil)
	if err != nil {
		panic(err)
	}
	configRequest(request)
	request.Close = true
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		panic(err)
	}
	return response
}

func CreateData(emailsResume string) (int, error) {
	request, err := http.NewRequest("POST", fmt.Sprintf("http://"+host+"/api/%s/_multi", nameIndexer), strings.NewReader(emailsResume))
	if err != nil {
		return 0, err
	}
  configRequest(request)
	request.Close = true
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return 0, err
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return 0, err
	}
	res := serializerResponse{}
	err = json.Unmarshal(body, &res)
	if err != nil {
		return 0, err
	}
	return res.Count, nil
}

func Query(text string, start, step int, sort string) (res Chits, err error) {
	var query_text string
	if text != "" {
		query_text = `{
						"query_string": {
											"query": "`+text+`"
										}
						}`
	}else{
		query_text = `{
						"match_all": {}
					  }`
	}
	query := `{
				"query": {
					"bool": {
						"must": [
							`+query_text+`
						]
					}
				},
				"sort": [
					"`+sort+`"
				],
				"from": `+fmt.Sprint(start)+`,
				"size": `+fmt.Sprint(step)+`,
				"aggs": {
					"histogram": {
						"auto_date_histogram": {
							"field": "@timestamp",
							"buckets": 100
						}
					}
				}
}`
	request, err := http.NewRequest("POST", "http://"+host+"/es/enron/_search", strings.NewReader(query))
	if err != nil {
		return
	}
	configRequest(request)
	request.Close = true
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	log.Println(resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	err = json.Unmarshal(body, &res)
	return
}
