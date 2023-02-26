package ds

import (
	"context"
	"log"
	"os"
	"time"

	"cloud.google.com/go/datastore"
	util "github.com/hojin-kr/clubhouse/cmd/util"
	"github.com/patrickmn/go-cache"
)

var (
	project_id = os.Getenv("PROJECT_ID")
	c          = cache.New(10*time.Second, 10*time.Minute)
)

func GetClient(ctx context.Context) *datastore.Client {
	var client *datastore.Client
	client, err := datastore.NewClient(ctx, project_id)
	if err != nil {
		log.Printf("get ds client" + err.Error())
	}
	return client
}

func Get(ctx context.Context, key *datastore.Key, dst interface{}) (err error) {
	cacheKey := util.GetCacheKeyOfDatastoreKey(*key)
	if x, found := c.Get(cacheKey); found {
		log.Print("cache hit")
		dst = x.(interface{})
	} else {
		log.Print("cache none")
		client := GetClient(ctx)
		if err := client.Get(ctx, key, dst); err != nil {
			log.Printf("get ds " + err.Error())
		}
		c.Set(cacheKey, dst, cache.DefaultExpiration)
	}
	return err
}

func Put(ctx context.Context, key *datastore.Key, src interface{}) (_key *datastore.Key) {
	client := GetClient(ctx)
	key, err := client.Put(ctx, key, src)
	if err != nil {
		log.Printf("put ds" + err.Error())
	}
	cacheKey := util.GetCacheKeyOfDatastoreKey(*key)
	c.Set(cacheKey, src, cache.DefaultExpiration)
	return key
}

func GetAll(ctx context.Context, query *datastore.Query, dst interface{}) (keys []*datastore.Key, err error) {
	client := GetClient(ctx)
	keys, err = client.GetAll(ctx, query, dst)
	if err != nil {
		log.Printf("query ds" + err.Error())
	}
	return keys, err
}

func GetMulti(ctx context.Context, keys []*datastore.Key, dst interface{}) (err error) {
	client := GetClient(ctx)
	if err := client.GetMulti(ctx, keys, dst); err != nil {
		log.Printf("getmulti ds" + err.Error())
	}
	return err
}
