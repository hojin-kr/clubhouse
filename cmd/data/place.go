package data

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"

	pb "github.com/hojin-kr/clubhouse/cmd/proto"
)

var (
	rest_api_key = os.Getenv("KAKAO_REST_API_KEY")
)

func QueryToKakaoPlace(query string, x string, y string, page string) pb.PlaceKakaoReply {
	params := url.Values{}
	params.Add("query", query)
	if len(y) > 0 {
		params.Add("y", y)
	}
	if len(x) > 0 {
		params.Add("x", x)
	}
	if len(page) > 0 {
		params.Add("page", page)
	}
	params.Add("radius", "20000")
	req, err := http.NewRequest("GET", "https://dapi.kakao.com/v2/local/search/keyword.json?"+params.Encode(), nil)
	if err != nil {
		panic(err)
	}
	req.Header.Add("Authorization", "KakaoAK "+rest_api_key)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	var data pb.PlaceKakaoReply
	err = decoder.Decode(&data)
	if err != nil {
		log.Printf("%T\n%s\n%#v\n", err, err, err)
	}
	return data
	// bytes, _ := ioutil.ReadAll(resp.Body)
	// str := string(bytes) //바이트를 문자열로
	// log.Print(str)
}
