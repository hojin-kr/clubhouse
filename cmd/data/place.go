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
	locations    = []string{"", "종로구", "중구", "용산구", "성동구", "광진구", "동대문구", "중랑구", "성북구", "강북구", "도봉구", "노원구", "은평구", "서대문구", "마포구", "양천구", "강서구", "구로구", "금천구", "영등포구", "동작구", "관악구", "서초구", "강남구", "송파구", "강동구"}
)

func GetLocationTypeString(index int64) string {
	return locations[index]
}

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
