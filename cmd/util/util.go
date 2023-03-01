package util

import (
	"fmt"
	"strconv"
	"strings"

	"cloud.google.com/go/datastore"
	pb "github.com/hojin-kr/clubhouse/cmd/proto"
)

func Difference(a, b []int64) int64 {
	mb := make(map[int64]struct{}, len(b))
	for _, x := range b {
		mb[x] = struct{}{}
	}
	var diff []int64
	for _, x := range a {
		if _, found := mb[x]; !found {
			diff = append(diff, x)
		}
	}
	return diff[0]
}

func GetCacheKeyOfDatastoreKey(dsKey datastore.Key) string {
	return fmt.Sprintf("%s:%d", dsKey.Kind, dsKey.ID)
}

func GetCacheKeyOfDatastoreQuery(dsKind string, dsKey int64, custom string) string {
	return fmt.Sprintf("%s:%d:%s", dsKind, dsKey, custom)
}

func GetCacheKeyOfDatastoreQueryGameFilter(dsKind string, filters []*pb.GameFilter, order int64, cursor string) string {
	var keys []string
	for i := 0; i < len(filters); i++ {
		keys = append(keys, strconv.FormatInt(filters[i].Value, 10))
	}
	return fmt.Sprintf("%s:%s:%d:%s", dsKind, strings.Join(keys, ":"), order, cursor)
}
