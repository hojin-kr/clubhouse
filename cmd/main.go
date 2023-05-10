package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	b64 "encoding/base64"

	"cloud.google.com/go/datastore"
	apns "github.com/edganiukov/apns"
	data "github.com/hojin-kr/clubhouse/cmd/data"
	ds "github.com/hojin-kr/clubhouse/cmd/ds"
	pb "github.com/hojin-kr/clubhouse/cmd/proto"
	"github.com/hojin-kr/clubhouse/cmd/trace"
	util "github.com/hojin-kr/clubhouse/cmd/util"
	cache "github.com/patrickmn/go-cache"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc"
)

var (
	port              = flag.Int("port", 50051, "The server port")
	project_id        = os.Getenv("PROJECT_ID")
	tracer            trace.Tracer
	apple_team_id     = os.Getenv("APPLE_TEAM_ID")
	apple_bundle_id   = os.Getenv("APPLE_BUNDLE_ID")
	apple_apns_key_id = os.Getenv("APPLE_APNS_KEY_ID")
	apple_apns_key    = os.Getenv("APPLE_APNS_KEY")
	environment       = os.Getenv("APP_ENVIRONMENT")
	c                 = cache.New(10*time.Second, 10*time.Minute)
)

// server is used to implement UnimplementedServiceServer
type server struct {
	pb.UnimplementedVersion1Server
}

// Account account infomation

// CreateAccount implements CreateAccount
func (s *server) CreateAccount(ctx context.Context, in *pb.AccountRequest) (*pb.AccountReply, error) {
	tracer.Trace(time.Now().Unix(), in)
	tm := time.Now().Unix()
	// Putting an entity into the datastore under an incomplete key will cause a unique key to be generated for that entity, with a non-zero IntID.
	key := ds.Put(ctx, datastore.IncompleteKey(getDatastoreKind("Account"), nil), &pb.Account{RegisterTimestamp: tm})
	ret := &pb.AccountReply{Id: key.ID, RegisterTimestamp: tm}
	// profile update
	var profile = &pb.Profile{
		AccountId: key.ID,
		Name:      "골퍼" + strconv.Itoa(int(key.ID))[0:4],
	}
	ds.Put(ctx, datastore.IDKey(getDatastoreKind("Profile"), key.ID, nil), profile)
	tracer.Trace(time.Now().Unix(), ret)
	return ret, nil
}

func (s *server) GetProfile(ctx context.Context, in *pb.ProfileRequest) (*pb.ProfileReply, error) {
	tracer.Trace(in)
	key := datastore.IDKey(getDatastoreKind("Profile"), in.Profile.GetAccountId(), nil)
	cacheKey := util.GetCacheKeyOfDatastoreKey(*key)
	if x, found := c.Get(cacheKey); found {
		ret := &pb.ProfileReply{Profile: x.(*pb.Profile)}
		return ret, nil
	}
	ds.Get(ctx, key, in.Profile)
	ret := &pb.ProfileReply{Profile: in.GetProfile()}
	go c.Set(cacheKey, in.Profile, cache.DefaultExpiration)
	return ret, nil
}

func (s *server) UpdateProfile(ctx context.Context, in *pb.ProfileRequest) (*pb.ProfileReply, error) {
	tracer.Trace(in)
	if in.Profile.GetAccountId() == 0 {
		tracer.Trace(time.Now().UTC(), in, "ID is 0")
		ret := &pb.ProfileReply{Profile: in.GetProfile()}
		return ret, nil
	}
	key := datastore.IDKey(getDatastoreKind("Profile"), in.Profile.GetAccountId(), nil)
	ds.Put(ctx, key, in.Profile)
	go c.Set(util.GetCacheKeyOfDatastoreKey(*key), in.Profile, cache.DefaultExpiration)
	ret := &pb.ProfileReply{Profile: in.GetProfile()}

	return ret, nil
}

func (s *server) CreateGame(ctx context.Context, in *pb.GameRequest) (*pb.GameReply, error) {
	tracer.Trace(in)
	// Game 생성
	var game = in.Game
	key := ds.Put(ctx, datastore.IncompleteKey(getDatastoreKind("Game"), nil), game)
	game.Id = key.ID
	game.Created = time.Now().UTC().Unix()
	ds.Put(ctx, datastore.IDKey(getDatastoreKind("Game"), key.ID, nil), game)
	ds.Put(ctx, datastore.IDKey(getDatastoreKind("GameList"), key.ID, nil), game) // 리스트에 뿌려주는 용도이며 TTL 세팅해서 지워지게
	ret := &pb.GameReply{Game: game}

	go c.Set(util.GetCacheKeyOfDatastoreKey(*key), game, cache.DefaultExpiration)
	return ret, nil
}

func (s *server) GetGame(ctx context.Context, in *pb.GameRequest) (*pb.GameReply, error) {
	tracer.Trace(in)
	key := datastore.IDKey(getDatastoreKind("Game"), in.Game.GetId(), nil)
	cacheKey := util.GetCacheKeyOfDatastoreKey(*key)
	if x, found := c.Get(cacheKey); found {
		ret := &pb.GameReply{Game: x.(*pb.Game)}
		return ret, nil
	}
	ds.Get(ctx, key, in.Game)
	ret := &pb.GameReply{Game: in.GetGame()}
	go c.Set(cacheKey, in.Game, cache.DefaultExpiration)
	return ret, nil
}

func (s *server) GetGameMulti(ctx context.Context, in *pb.GameMultiRequest) (*pb.GameMultiReply, error) {
	tracer.Trace(in)
	keys := []*datastore.Key{}
	for i := 0; i < len(in.GameIds); i++ {
		key := datastore.IDKey(getDatastoreKind("Game"), in.GameIds[i], nil)
		keys = append(keys, key)
	}
	games := make([]*pb.Game, len(in.GameIds))
	ds.GetMulti(ctx, keys, games)
	ret := &pb.GameMultiReply{Games: games}

	return ret, nil
}

func (s *server) UpdateGame(ctx context.Context, in *pb.GameRequest) (*pb.GameReply, error) {
	tracer.Trace(in)
	in.Game.Updated = time.Now().UTC().Unix()
	// 조인을 수락했습니다. 거절했습니다로 노티
	var gameBefore pb.Game
	IDKey := datastore.IDKey(getDatastoreKind("Game"), in.Game.GetId(), nil)
	ds.Get(ctx, IDKey, &gameBefore)
	_ = ds.Put(ctx, IDKey, in.Game)
	_ = ds.Put(ctx, datastore.IDKey(getDatastoreKind("GameList"), in.Game.GetId(), nil), in.Game)
	ret := &pb.GameReply{Game: in.GetGame()}
	go c.Set(util.GetCacheKeyOfDatastoreKey(*IDKey), in.Game, cache.DefaultExpiration)
	return ret, nil
}

func (s *server) DeleteGame(ctx context.Context, in *pb.GameRequest) (*pb.GameReply, error) {
	tracer.Trace(in)
	key := datastore.IDKey(getDatastoreKind("Game"), in.Game.GetId(), nil)
	ds.Delete(ctx, key)
	// GameList에서도 삭제
	ds.Delete(ctx, datastore.IDKey(getDatastoreKind("GameList"), in.Game.GetId(), nil))
	ret := &pb.GameReply{Game: in.GetGame()}
	go c.Delete(util.GetCacheKeyOfDatastoreKey(*key))
	return ret, nil
}

// filterdGames에서는 Game 목록만 반환하고 GetGame에서는 attend, place 부가 정보 반환
func (s *server) GetFilterdGames(ctx context.Context, in *pb.FilterdGamesRequest) (*pb.FilterdGamesReply, error) {
	tracer.Trace(in)
	cacheKey := util.GetCacheKeyOfDatastoreQueryGameFilter("GameList", in.Filter, in.TypeOrder, in.Cursor)
	log.Print(cacheKey)
	if x, found := c.Get(cacheKey); found {
		_games := x.([]*pb.Game)
		ret := &pb.FilterdGamesReply{Games: _games, Cursor: in.Cursor}
		fmt.Printf(cacheKey)
		return ret, nil
	}
	client := ds.GetClient(ctx)
	cursorStr := in.Cursor
	const pageSize = 100
	var orderTypes = map[int64]string{
		0: "Created",
		1: "-Created",
		2: "Time",
		3: "-Time",
		4: "Price",
		5: "-Price",
	}

	queryBase := datastore.NewQuery(getDatastoreKind("GameList"))
	query := queryBase
	for i := 0; i < len(in.Filter); i++ {
		if in.Filter[i].Value != 0 || in.Filter[i].Key == "TypePlay" {
			if in.Filter[i].Key == "ShortAddress" {
				query = query.Filter(in.Filter[i].Key+" =", data.GetLocationTypeString(in.Filter[i].Value))
			} else {
				query = query.Filter(in.Filter[i].Key+" =", in.Filter[i].Value)
			}
		}
	}
	query = query.
		Order(orderTypes[in.TypeOrder]).
		Limit(pageSize)

	if cursorStr != "" {
		cursor, err := datastore.DecodeCursor(cursorStr)
		if err != nil {
			log.Fatalf("Bad cursor %q: %v", cursorStr, err)
		}
		query = query.Start(cursor)
	}
	// Read the games.
	var games []pb.Game
	var game pb.Game
	it := client.Run(ctx, query)
	_, err := it.Next(&game)
	for err == nil {
		games = append(games, game)
		game = pb.Game{}
		_, err = it.Next(&game)
	}
	if err != iterator.Done {
		log.Fatalf("Failed fetching results: %v", err)
	}

	// Get the cursor for the next page of results.
	// nextCursor.String can be used as the next page's token.
	nextCursor, err := it.Cursor()
	// [END datastore_cursor_paging]
	_ = err        // Check the error.
	_ = nextCursor // Use nextCursor.String as the next page's token.
	var _games []*pb.Game
	now := time.Now().UTC().Unix()
	for i := 0; i < len(games); i++ {
		if now > games[i].Time {
			// todo 모아서 del 요청
			continue
		}
		_games = append(_games, &games[i])
	}
	go c.Set(cacheKey, _games, cache.DefaultExpiration)
	ret := &pb.FilterdGamesReply{Games: _games, Cursor: nextCursor.String()}

	return ret, nil
}

func (s *server) Join(ctx context.Context, in *pb.JoinRequest) (*pb.JoinReply, error) {
	tracer.Trace(in)
	var join = in.Join
	key := ds.Put(ctx, datastore.IncompleteKey(getDatastoreKind("Join"), nil), join)
	join.JoinId = key.ID
	join.Created = time.Now().UTC().Unix()
	_ = ds.Put(ctx, datastore.IDKey(getDatastoreKind("Join"), key.ID, nil), join)
	ret := &pb.JoinReply{Join: join}
	_ctx := context.Background()
	go setJoinRequestPush(_ctx, in)
	return ret, nil
}

func (s *server) UpdateJoin(ctx context.Context, in *pb.JoinRequest) (*pb.JoinReply, error) {
	tracer.Trace(in)
	if in.Join.GetAccountId() == 0 {
		tracer.Trace(time.Now().UTC(), in, "ID is 0")
		ret := &pb.JoinReply{Join: in.GetJoin()}
		return ret, nil
	}
	in.Join.Updated = time.Now().Unix()
	ds.Put(ctx, datastore.IDKey(getDatastoreKind("Join"), in.Join.GetJoinId(), nil), in.Join)
	ret := &pb.JoinReply{Join: in.GetJoin()}
	_ctx := context.Background()
	go setJoinUpdatePush(_ctx, in)
	// del cache
	go c.Delete(util.GetCacheKeyOfDatastoreQuery("Join", in.Join.AccountId, "myJoins"))
	go c.Delete(util.GetCacheKeyOfDatastoreQuery("Join", in.Join.AccountId, "beforeJoins"))
	go c.Delete(util.GetCacheKeyOfDatastoreQuery("Join", in.Join.GameId, "gameJoins:"))

	return ret, nil
}

const (
	StatusJoinDefault = 0
	StatusJoinAccept  = 1
	StatusJoinReject  = 2
	StatusJoinCancel  = 3
)

func (s *server) GetMyJoins(ctx context.Context, in *pb.JoinRequest) (*pb.JoinReply, error) {
	// 취소 제외 조인 목록 조회
	tracer.Trace(in)
	cacheKey := util.GetCacheKeyOfDatastoreQuery("Join", in.Join.AccountId, "myJoins")
	if x, found := c.Get(cacheKey); found {
		joins := x.([]*pb.Join)
		ret := &pb.JoinReply{Joins: joins, Cursor: in.Cursor}
		fmt.Printf(cacheKey)
		return ret, nil
	}
	client := ds.GetClient(ctx)
	cursorStr := in.Cursor
	const pageSize = 100
	query := datastore.NewQuery(getDatastoreKind("Join")).Filter("AccountId =", in.Join.GetAccountId()).Filter("Start >", time.Now().Unix()).Order("Start").Limit(pageSize)
	if cursorStr != "" {
		cursor, err := datastore.DecodeCursor(cursorStr)
		if err != nil {
			log.Fatalf("Bad cursor %q: %v", cursorStr, err)
		}
		query = query.Start(cursor)
	}
	// Read the join.
	var joins []pb.Join
	var join pb.Join
	it := client.Run(ctx, query)
	_, err := it.Next(&join)
	for err == nil {
		joins = append(joins, join)
		join = pb.Join{}
		_, err = it.Next(&join)
	}
	if err != iterator.Done {
		log.Fatalf("Failed fetching results: %v", err)
	}
	// Get the cursor for the next page of results.
	// nextCursor.String can be used as the next page's token.
	nextCursor, err := it.Cursor()
	// [END datastore_cursor_paging]
	_ = err        // Check the error.
	_ = nextCursor // Use nextCursor.String as the next page's token.
	var _joins []*pb.Join
	for i := 0; i < len(joins); i++ {
		_joins = append(_joins, &joins[i])
	}
	go c.Set(cacheKey, _joins, cache.DefaultExpiration)
	ret := &pb.JoinReply{Joins: _joins, Cursor: nextCursor.String()}

	return ret, nil
}

func (s *server) GetMyBeforeJoins(ctx context.Context, in *pb.JoinRequest) (*pb.JoinReply, error) {
	// 지난 Accept 조인 목록
	tracer.Trace(in)
	cacheKey := util.GetCacheKeyOfDatastoreQuery("Join", in.Join.AccountId, "beforeJoins")
	if x, found := c.Get(cacheKey); found {
		joins := x.([]*pb.Join)
		ret := &pb.JoinReply{Joins: joins, Cursor: in.Cursor}
		fmt.Printf(cacheKey)
		return ret, nil
	}
	client := ds.GetClient(ctx)
	cursorStr := in.Cursor
	const pageSize = 50
	query := datastore.NewQuery(getDatastoreKind("Join")).Filter("AccountId =", in.Join.GetAccountId()).Filter("Status =", StatusJoinAccept).Filter("Start <", time.Now().Unix()).Order("Start").Limit(pageSize)
	if cursorStr != "" {
		cursor, err := datastore.DecodeCursor(cursorStr)
		if err != nil {
			log.Fatalf("Bad cursor %q: %v", cursorStr, err)
		}
		query = query.Start(cursor)
	}
	// Read the join.
	var joins []pb.Join
	var join pb.Join
	it := client.Run(ctx, query)
	_, err := it.Next(&join)
	for err == nil {
		joins = append(joins, join)
		join = pb.Join{}
		_, err = it.Next(&join)
	}
	if err != iterator.Done {
		log.Fatalf("Failed fetching results: %v", err)
	}
	// Get the cursor for the next page of results.
	// nextCursor.String can be used as the next page's token.
	nextCursor, err := it.Cursor()
	// [END datastore_cursor_paging]
	_ = err        // Check the error.
	_ = nextCursor // Use nextCursor.String as the next page's token.
	var _joins []*pb.Join
	for i := 0; i < len(joins); i++ {
		_joins = append(_joins, &joins[i])
	}
	go c.Set(cacheKey, _joins, cache.DefaultExpiration)
	ret := &pb.JoinReply{Joins: _joins, Cursor: nextCursor.String()}

	return ret, nil
}

func (s *server) GetGameJoins(ctx context.Context, in *pb.JoinRequest) (*pb.JoinReply, error) {
	tracer.Trace(in)
	cacheKey := util.GetCacheKeyOfDatastoreQuery("Join", in.Join.GameId, "gameJoins:"+in.Cursor)
	if x, found := c.Get(cacheKey); found {
		joins := x.([]*pb.Join)
		ret := &pb.JoinReply{Joins: joins, Cursor: in.Cursor}
		fmt.Printf(cacheKey)
		return ret, nil
	}
	client := ds.GetClient(ctx)
	cursorStr := in.Cursor
	const pageSize = 100
	query := datastore.NewQuery(getDatastoreKind("Join")).Filter("GameId =", in.Join.GetGameId()).Order("Created").Limit(pageSize)
	if cursorStr != "" {
		cursor, err := datastore.DecodeCursor(cursorStr)
		if err != nil {
			log.Fatalf("Bad cursor %q: %v", cursorStr, err)
		}
		query = query.Start(cursor)
	}
	// Read the join.
	var joins []pb.Join
	var join pb.Join
	it := client.Run(ctx, query)
	_, err := it.Next(&join)
	for err == nil {
		joins = append(joins, join)
		join = pb.Join{}
		_, err = it.Next(&join)
	}
	if err != iterator.Done {
		log.Fatalf("Failed fetching results: %v", err)
	}
	// Get the cursor for the next page of results.
	// nextCursor.String can be used as the next page's token.
	nextCursor, err := it.Cursor()
	// [END datastore_cursor_paging]
	_ = err        // Check the error.
	_ = nextCursor // Use nextCursor.String as the next page's token.
	var _joins []*pb.Join
	for i := 0; i < len(joins); i++ {
		_joins = append(_joins, &joins[i])
	}
	go c.Set(cacheKey, _joins, cache.DefaultExpiration)
	ret := &pb.JoinReply{Joins: _joins, Cursor: nextCursor.String()}

	return ret, nil
}

func (s *server) GetChat(ctx context.Context, in *pb.ChatRequest) (*pb.ChatReply, error) {
	tracer.Trace(in)
	cacheKey := util.GetCacheKeyOfDatastoreQuery("Chat", in.Chat.ForeginId, "")
	if x, found := c.Get(cacheKey); found {
		chats := x.([]*pb.Chat)
		ret := &pb.ChatReply{Chats: chats, Cursor: ""}
		tracer.Trace(cacheKey + " getChatCacheHit")
		return ret, nil
	}
	var chats []*pb.Chat
	const pageSize = 100
	q := datastore.NewQuery(getDatastoreKind("Chat")).Filter("ForeginId =", in.Chat.GetForeginId()).Order("Created").Limit(pageSize)
	ds.GetAll(ctx, q, &chats)
	go c.Set(cacheKey, chats, cache.DefaultExpiration)
	ret := &pb.ChatReply{Chats: chats, Cursor: ""}

	return ret, nil
}

// update my chat
func (s *server) AddChatMessage(ctx context.Context, in *pb.ChatMessageRequest) (*pb.ChatReply, error) {
	tracer.Trace(in)
	cacheKey := util.GetCacheKeyOfDatastoreQuery("Chat", in.ForeginId, "")
	var Chat pb.Chat
	const chatSize = 500
	// get
	key := datastore.NameKey(getDatastoreKind("Chat"), strconv.FormatInt(in.GetForeginId()+in.GetAccountId(), 10), nil)
	ds.Get(ctx, key, &Chat)
	// append & put
	Chat.AccountId = in.GetAccountId()
	Chat.ForeginId = in.GetForeginId()

	NowUnix := time.Now().UTC().Unix()
	Chat.Updated = NowUnix
	if Chat.GetCreated() == 0 {
		Chat.Created = NowUnix
	}
	in.ChatMessage.Created = NowUnix
	in.ChatMessage.AccountId = in.GetAccountId()
	Chat.ChatMessages = append(Chat.ChatMessages, in.ChatMessage)
	if len(Chat.ChatMessages) > chatSize {
		Chat.ChatMessages = Chat.ChatMessages[1:]
	}
	ds.Put(ctx, key, &Chat)
	// return all chats
	var chats []*pb.Chat
	const pageSize = 100
	q := datastore.NewQuery(getDatastoreKind("Chat")).Filter("ForeginId =", in.GetForeginId()).Order("Created").Limit(pageSize)
	ds.GetAll(ctx, q, &chats)
	go c.Set(cacheKey, chats, cache.DefaultExpiration)
	ret := &pb.ChatReply{Chats: chats}
	_ctx := context.Background()
	go setChatPush(_ctx, in.ForeginId, in.GetAccountId(), in.ChatMessage.Message)

	return ret, nil
}

func (s *server) GetPlaceKaKao(ctx context.Context, in *pb.PlaceKakaoRequest) (*pb.PlaceKakaoReply, error) {
	tracer.Trace(in)
	data := data.QueryToKakaoPlace(in.Query, in.X, in.Y, in.Page)
	ret := &pb.PlaceKakaoReply{Meta: data.Meta, Documents: data.Documents}

	return ret, nil
}

func setJoinRequestPush(ctx context.Context, in *pb.JoinRequest) {
	var game pb.Game
	var profile pb.Profile
	var apnsTokens []string
	dsKeyGame := datastore.IDKey(getDatastoreKind("Game"), in.Join.GetGameId(), nil)
	ds.Get(ctx, dsKeyGame, &game)
	if game.GetHostAccountId() != in.Join.AccountId {
		dsKeyProfile := datastore.IDKey(getDatastoreKind("Profile"), game.GetHostAccountId(), nil)
		ds.Get(ctx, dsKeyProfile, &profile)
		apnsTokens = append(apnsTokens, profile.ApnsToken)
		pushNotification(apnsTokens, "클럽하우스", game.PlaceName, "조인 신청이 도착했습니다.")
	} else {

	}
}

func setJoinUpdatePush(ctx context.Context, in *pb.JoinRequest) {
	var game pb.Game
	var profile pb.Profile
	var apnsTokens []string
	dsKeyGame := datastore.IDKey(getDatastoreKind("Game"), in.Join.GetGameId(), nil)
	ds.Get(ctx, dsKeyGame, &game)
	dsKeyProfile := datastore.IDKey(getDatastoreKind("Profile"), in.Join.AccountId, nil)
	ds.Get(ctx, dsKeyProfile, &profile)
	apnsTokens = append(apnsTokens, profile.ApnsToken)
	StringStatus := "수락"
	if in.Join.Status == StatusJoinReject {
		StringStatus = "거절"
	}
	if in.Join.Status == StatusJoinCancel {
		StringStatus = "취소"
	}
	pushNotification(apnsTokens, "클럽하우스", game.PlaceName, "조인이 "+StringStatus+" 되었습니다")
}

func setJoinChangePush(ctx context.Context, in *pb.GameRequest, before *pb.Game) {
	var accountID int64
	var changeStatus = ""
	if len(in.Game.AcceptAccountIds) > len(before.AcceptAccountIds) {
		accountID = util.Difference(in.Game.AcceptAccountIds, before.AcceptAccountIds)
		changeStatus = "수락"
	}
	if len(in.Game.RejectAccountIds) > len(before.RejectAccountIds) {
		accountID = util.Difference(in.Game.RejectAccountIds, before.RejectAccountIds)
		changeStatus = "거절"
	}
	if changeStatus != "" {
		var profile pb.Profile
		var apnsTokens []string
		dsKeyProfile := datastore.IDKey(getDatastoreKind("Profile"), accountID, nil)
		if x, found := c.Get(util.GetCacheKeyOfDatastoreKey(*dsKeyProfile)); found {
			profile = x.(pb.Profile)
		} else {
			ds.Get(ctx, dsKeyProfile, &profile)
		}
		apnsTokens = append(apnsTokens, profile.ApnsToken)
		pushNotification(apnsTokens, "클럽하우스", in.Game.PlaceName, "조인 신청이 "+changeStatus+"됐습니다.")
	}
}

func setChatPush(ctx context.Context, gameID int64, accountID int64, message string) {
	if message != "" {
		// accept account all
		var game pb.Game
		var senderName string
		dsKeyGame := datastore.IDKey(getDatastoreKind("Game"), gameID, nil)
		ds.Get(ctx, dsKeyGame, &game)
		var apnsTokens []string
		var joins []*pb.Join
		q := datastore.NewQuery(getDatastoreKind("Join")).Filter("GameId =", gameID).Filter("Status =", StatusJoinAccept).Limit(10)
		ds.GetAll(ctx, q, &joins)
		keys := []*datastore.Key{}
		for _, x := range joins {
			dsKeyProfile := datastore.IDKey(getDatastoreKind("Profile"), x.AccountId, nil)
			keys = append(keys, dsKeyProfile)
		}
		profiles := make([]*pb.Profile, len(keys))
		ds.GetMulti(ctx, keys, profiles)
		for _, p := range profiles {
			// sender에게는 보내지 않음
			if p.AccountId != accountID {
				apnsTokens = append(apnsTokens, p.ApnsToken)
			}
			// sender name 포함해서 발송
			if p.AccountId == accountID {
				senderName = p.Name
			}
		}
		if len(apnsTokens) > 0 {
			// todo PlaceName 은 변하지 않는 값이라서 클라이언트에서 받아와서 보내는걸로 변경
			pushNotification(apnsTokens, "클럽하우스", game.PlaceName, senderName+" : "+message)
			go c.Set("game:chat:apnstokens", apnsTokens, cache.DefaultExpiration)
		}
	}
}

func pushNotification(apnsTokens []string, title string, subtitle string, body string) {
	const (
		DevelopmentGateway = "https://api.sandbox.push.apple.com"
		ProductionGateway  = "https://api.push.apple.com"
	)
	GateWay := DevelopmentGateway
	if environment == "production" {
		GateWay = ProductionGateway
	}
	_apple_apns_key, _ := b64.StdEncoding.DecodeString(apple_apns_key)
	c, err := apns.NewClient(
		apns.WithJWT(_apple_apns_key, apple_apns_key_id, apple_team_id),
		apns.WithBundleID(apple_bundle_id),
		apns.WithMaxIdleConnections(10),
		apns.WithTimeout(5*time.Second),
		apns.WithEndpoint(GateWay),
	)
	if err != nil {
		print(err)
		/* ... */
	}
	for i := 0; i < len(apnsTokens); i++ {
		resp, err := c.Send(apnsTokens[i],
			apns.Payload{
				APS: apns.APS{
					Alert: apns.Alert{
						Title:    title,
						Subtitle: subtitle,
						Body:     body,
					},
					Sound: "default",
				},
			},
			apns.WithExpiration(10),
			apns.WithPriority(5),
		)
		if err != nil {
			print(err)
			/* ... */
		}
		print(resp.Timestamp)
	}
}

func (s *server) CreateArticle(ctx context.Context, in *pb.ArticleRequest) (*pb.ArticleReply, error) {
	tracer.Trace(in)
	var article = in.Article
	article.Created = time.Now().UTC().Unix()
	put := ds.Put(ctx, datastore.IncompleteKey(getDatastoreKind("Article"), nil), article)
	article.Id = put.ID
	ds.Put(ctx, datastore.IDKey(getDatastoreKind("Article"), put.ID, nil), article)
	ret := &pb.ArticleReply{Article: article}

	return ret, nil
}

func (s *server) UpdateArticle(ctx context.Context, in *pb.ArticleRequest) (*pb.ArticleReply, error) {
	tracer.Trace(in)
	in.Article.Updated = time.Now().UTC().Unix()
	_ = ds.Put(ctx, datastore.IDKey(getDatastoreKind("Article"), in.Article.GetId(), nil), in.Article)
	ret := &pb.ArticleReply{Article: in.GetArticle()}

	return ret, nil
}

func (s *server) GetFilterdArticles(ctx context.Context, in *pb.FilterdArticlesRequest) (*pb.FilterdArticlesReply, error) {
	tracer.Trace(in)
	client := ds.GetClient(ctx)
	cursorStr := in.Cursor
	const pageSize = 10
	queryBase := datastore.NewQuery(getDatastoreKind("Article"))
	query := queryBase.Order("-Created").Filter("Category =", in.Category).Filter("Type = ", in.Type).Limit(pageSize)
	const typeAccount = 2
	if in.Type == typeAccount {
		query = queryBase.Order("-Created").Filter("AccountId =", in.AccountId).Limit(pageSize)
	}

	if cursorStr != "" {
		cursor, err := datastore.DecodeCursor(cursorStr)
		if err != nil {
			log.Fatalf("Bad cursor %q: %v", cursorStr, err)
		}
		query = query.Start(cursor)
	}
	var articles []pb.Article
	var article pb.Article
	it := client.Run(ctx, query)
	_, err := it.Next(&article)
	// todo multi get likes 카운팅할지 고민
	for err == nil {
		articles = append(articles, article)
		article = pb.Article{}
		_, err = it.Next(&article)
	}
	if err != iterator.Done {
		log.Fatalf("Failed fetching results: %v", err)
	}

	// Get the cursor for the next page of results.
	// nextCursor.String can be used as the next page's token.
	nextCursor, err := it.Cursor()
	// [END datastore_cursor_paging]
	_ = err        // Check the error.
	_ = nextCursor // Use nextCursor.String as the next page's token.
	var _articles []*pb.Article
	for i := 0; i < len(articles); i++ {
		_articles = append(_articles, &articles[i])
	}
	ret := &pb.FilterdArticlesReply{Articles: _articles, Cursor: nextCursor.String()}

	return ret, nil
}

func (s *server) GetCount(ctx context.Context, in *pb.Count) (*pb.Count, error) {
	tracer.Trace(in)
	var count pb.Count
	ds.Get(ctx, datastore.IDKey(getDatastoreKind(in.Kind), in.ForeginId, nil), &count)
	ret := &pb.Count{Count: count.Count, ForeginId: in.ForeginId, Kind: in.Kind}

	return ret, nil
}

var likeTypes = map[int64]string{
	0: "LikeArticle",
	1: "LikeGame",
}

func (s *server) CreateLike(ctx context.Context, in *pb.LikeRequest) (*pb.LikeReply, error) {
	tracer.Trace(in)
	in.Like.Created = time.Now().UTC().Unix()
	log.Print(in.Like.GetForeginAccountId())
	_ = ds.Put(ctx, datastore.IDKey(getDatastoreKind(likeTypes[in.Like.Type]), in.Like.Id, nil), in.Like)
	// just counting
	go CountIncr(context.Background(), in.Like.Id, likeTypes[in.Like.Type])
	go setAccountIdPush(context.Background(), in.Like.GetForeginAccountId(), in.Like.GetTitle(), "+1 좋아합니다.")
	ret := &pb.LikeReply{Like: in.Like}

	return ret, nil
}

// 계정 아이디로 푸시 발송
func setAccountIdPush(ctx context.Context, id int64, title string, body string) {
	var profile *pb.Profile
	var apnsTokens []string
	key := datastore.IDKey(getDatastoreKind(getDatastoreKind("Profile")), id, nil)
	if x, found := c.Get(util.GetCacheKeyOfDatastoreKey(*key)); found {
		profile = x.(*pb.Profile)
	} else {
		ds.Get(ctx, key, profile)
	}
	apnsTokens = append(apnsTokens, profile.ApnsToken)
	pushNotification(apnsTokens, "클럽하우스", title, body)
}

func (s *server) UpdateLike(ctx context.Context, in *pb.LikeRequest) (*pb.LikeReply, error) {
	tracer.Trace(in)
	in.Like.Created = time.Now().UTC().Unix()
	_ = ds.Put(ctx, datastore.IDKey(likeTypes[in.Like.Type], in.Like.Id, nil), in.Like)
	ret := &pb.LikeReply{Like: in.Like}

	return ret, nil
}

func (s *server) GetEtcd(ctx context.Context, in *pb.EtcdRequest) (*pb.EtcdReply, error) {
	tracer.Trace(in)
	switch in.Key {
	case "ETCD_IS_AD":
		in.Value = os.Getenv("ETCD_IS_AD")
	case "ETCD_INTERVAL_AD":
		in.Value = os.Getenv("ETCD_INTERVAL_AD")
	}
	ret := &pb.EtcdReply{Key: in.Key, Value: in.Value}

	return ret, nil
}

func CountIncr(ctx context.Context, id int64, kind string) {
	var count pb.Count
	IDKey := datastore.IDKey(kind, id, nil)
	err := ds.Get(ctx, IDKey, &count)
	if err != nil {
		count.Count = 1
	} else {
		count.Count = count.Count + 1
	}
	_ = ds.Put(ctx, IDKey, &count)
}

func getDatastoreKind(kind string) string {
	return getEnv() + kind
}
func getEnv() string {
	return environment
}

func main() {
	flag.Parse()
	tracer = trace.New(os.Stdout)
	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", *port))
	if err != nil {
		log.Printf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterVersion1Server(s, &server{})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Printf("failed to serve: %v", err)
	}
}
