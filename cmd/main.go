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
	data "github.com/hojin-kr/haru/cmd/data"
	ds "github.com/hojin-kr/haru/cmd/ds"
	pb "github.com/hojin-kr/haru/cmd/proto"
	"github.com/hojin-kr/haru/cmd/trace"
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
	is_production     = os.Getenv("IS_PRODUCTION")
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
	key := ds.Put(ctx, datastore.IncompleteKey("Account", nil), &pb.Account{RegisterTimestamp: tm})
	ret := &pb.AccountReply{Id: key.ID, RegisterTimestamp: tm}
	// profile update
	var profile = &pb.Profile{
		AccountId: key.ID,
		Name:      "골퍼" + strconv.Itoa(int(key.ID))[0:4],
	}
	ds.Put(ctx, datastore.IDKey("Profile", key.ID, nil), profile)
	tracer.Trace(time.Now().Unix(), ret)
	return ret, nil
}

func (s *server) GetProfile(ctx context.Context, in *pb.ProfileRequest) (*pb.ProfileReply, error) {
	tracer.Trace(time.Now().UTC(), in)
	key := datastore.IDKey("Profile", in.Profile.GetAccountId(), nil)
	ds.Get(ctx, key, in.Profile)
	ret := &pb.ProfileReply{Profile: in.GetProfile()}
	tracer.Trace(time.Now().UTC(), ret)
	return ret, nil
}

func (s *server) UpdateProfile(ctx context.Context, in *pb.ProfileRequest) (*pb.ProfileReply, error) {
	tracer.Trace(time.Now().UTC(), in)
	if in.Profile.GetAccountId() == 0 {
		tracer.Trace(time.Now().UTC(), in, "ID is 0")
		ret := &pb.ProfileReply{Profile: in.GetProfile()}
		return ret, nil
	}
	ds.Put(ctx, datastore.IDKey("Profile", in.Profile.GetAccountId(), nil), in.Profile)
	ret := &pb.ProfileReply{Profile: in.GetProfile()}
	tracer.Trace(time.Now().UTC(), ret)
	return ret, nil
}

func (s *server) CreateGame(ctx context.Context, in *pb.GameRequest) (*pb.GameReply, error) {
	tracer.Trace(time.Now().UTC(), in)
	// Game 생성
	var game = in.Game
	key := ds.Put(ctx, datastore.IncompleteKey("Game", nil), game)
	game.Id = key.ID
	game.Created = time.Now().UTC().Unix()
	_ = ds.Put(ctx, datastore.IDKey("Game", key.ID, nil), game)
	ds.Put(ctx, datastore.IDKey("GameList", key.ID, nil), game) // 리스트에 뿌려주는 용도이며 TTL 세팅해서 지워지게
	ret := &pb.GameReply{Game: game}
	tracer.Trace(time.Now().UTC(), ret)
	return ret, nil
}

func (s *server) GetGame(ctx context.Context, in *pb.GameRequest) (*pb.GameReply, error) {
	tracer.Trace(time.Now().UTC(), in)
	ds.Get(ctx, datastore.IDKey("Game", in.Game.GetId(), nil), in.Game)
	ret := &pb.GameReply{Game: in.GetGame()}
	tracer.Trace(time.Now().UTC(), ret)
	return ret, nil
}

func (s *server) GetGameMulti(ctx context.Context, in *pb.GameMultiRequest) (*pb.GameMultiReply, error) {
	tracer.Trace(time.Now().UTC(), in)
	keys := []*datastore.Key{}
	for i := 0; i < len(in.GameIds); i++ {
		keys = append(keys, datastore.IDKey("Game", in.GameIds[i], nil))
	}
	keys = append(keys)
	games := make([]*pb.Game, len(in.GameIds))
	ds.GetMulti(ctx, keys, games)
	ret := &pb.GameMultiReply{Games: games}
	tracer.Trace(time.Now().UTC(), ret)
	return ret, nil
}

func (s *server) UpdateGame(ctx context.Context, in *pb.GameRequest) (*pb.GameReply, error) {
	tracer.Trace(time.Now().UTC(), in)
	in.Game.Updated = time.Now().UTC().Unix()
	// 조인을 수락했습니다. 거절했습니다로 노티
	var gameBefore pb.Game
	IDKey := datastore.IDKey("Game", in.Game.GetId(), nil)
	ds.Get(ctx, IDKey, &gameBefore)
	_ctx := context.Background()
	go setJoinChangePush(_ctx, in, &gameBefore)
	_ = ds.Put(ctx, IDKey, in.Game)
	ret := &pb.GameReply{Game: in.GetGame()}
	tracer.Trace(time.Now().UTC(), ret)
	return ret, nil
}

// filterdGames에서는 Game 목록만 반환하고 GetGame에서는 attend, place 부가 정보 반환
func (s *server) GetFilterdGames(ctx context.Context, in *pb.FilterdGamesRequest) (*pb.FilterdGamesReply, error) {
	tracer.Trace(time.Now().UTC(), in)
	client := ds.GetClient(ctx)
	cursorStr := in.Cursor
	const pageSize = 10
	var orderTypes = map[int64]string{
		0: "Created",
		1: "-Created",
		2: "Time",
		3: "-Time",
		4: "Price",
		5: "-Price",
	}
	var filterTypes = map[int64]string{
		0: "TypePlay",
		1: "TypePlay",
	}

	queryBase := datastore.NewQuery("GameList")
	query := queryBase.
		Order(orderTypes[in.TypeOrder]).
		Limit(pageSize)

	if _, ok := filterTypes[in.TypeFilter]; ok {
		query = queryBase.
			Filter(filterTypes[in.TypeFilter]+" =", in.TypeFilter).
			Order(filterTypes[in.TypeFilter]).
			Limit(pageSize)
	}

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
		log.Print(game.Price)
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
	for i := 0; i < len(games); i++ {
		_games = append(_games, &games[i])
	}
	ret := &pb.FilterdGamesReply{Games: _games, Cursor: nextCursor.String()}
	tracer.Trace(time.Now().UTC(), ret)
	return ret, nil
}

func (s *server) Join(ctx context.Context, in *pb.JoinRequest) (*pb.JoinReply, error) {
	tracer.Trace(time.Now().UTC(), in)
	var join = in.Join
	key := ds.Put(ctx, datastore.IncompleteKey("Join", nil), join)
	join.JoinId = key.ID
	join.Created = time.Now().UTC().Unix()
	_ = ds.Put(ctx, datastore.IDKey("Join", key.ID, nil), join)
	ret := &pb.JoinReply{Join: join}
	_ctx := context.Background()
	go setJoinRequestPush(_ctx, in)
	tracer.Trace(time.Now().UTC(), ret)
	return ret, nil
}

func (s *server) UpdateJoin(ctx context.Context, in *pb.JoinRequest) (*pb.JoinReply, error) {
	tracer.Trace(time.Now().UTC(), in)
	if in.Join.GetAccountId() == 0 {
		tracer.Trace(time.Now().UTC(), in, "ID is 0")
		ret := &pb.JoinReply{Join: in.GetJoin()}
		return ret, nil
	}
	in.Join.Updated = time.Now().Unix()
	ds.Put(ctx, datastore.IDKey("Join", in.Join.GetJoinId(), nil), in.Join)
	ret := &pb.JoinReply{Join: in.GetJoin()}
	tracer.Trace(time.Now().UTC(), ret)
	return ret, nil
}

const (
	StatusJoinDefault = 0
	StatusJoinCancel  = 1
)

func (s *server) GetMyJoins(ctx context.Context, in *pb.JoinRequest) (*pb.JoinReply, error) {
	tracer.Trace(time.Now().UTC(), in)
	client := ds.GetClient(ctx)
	cursorStr := in.Cursor
	const pageSize = 100
	query := datastore.NewQuery("Join").Filter("AccountId =", in.Join.GetAccountId()).Filter("Status =", StatusJoinDefault).Filter("Start >", time.Now().Unix()).Order("Start").Limit(pageSize)
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

	ret := &pb.JoinReply{Joins: _joins, Cursor: nextCursor.String()}
	tracer.Trace(time.Now().UTC(), ret)
	return ret, nil
}

func (s *server) GetMyBeforeJoins(ctx context.Context, in *pb.JoinRequest) (*pb.JoinReply, error) {
	tracer.Trace(time.Now().UTC(), in)
	client := ds.GetClient(ctx)
	cursorStr := in.Cursor
	const pageSize = 50
	query := datastore.NewQuery("Join").Filter("AccountId =", in.Join.GetAccountId()).Filter("Status =", StatusJoinDefault).Filter("Start <", time.Now().Unix()).Order("Start").Limit(pageSize)
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

	ret := &pb.JoinReply{Joins: _joins, Cursor: nextCursor.String()}
	tracer.Trace(time.Now().UTC(), ret)
	return ret, nil
}

func (s *server) GetGameJoins(ctx context.Context, in *pb.JoinRequest) (*pb.JoinReply, error) {
	tracer.Trace(time.Now().UTC(), in)
	client := ds.GetClient(ctx)
	cursorStr := in.Cursor
	const pageSize = 10
	query := datastore.NewQuery("Join").Filter("GameId =", in.Join.GetGameId()).Filter("Status =", StatusJoinDefault).Order("Created").Limit(pageSize)
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
	ret := &pb.JoinReply{Joins: _joins, Cursor: nextCursor.String()}
	tracer.Trace(time.Now().UTC(), ret)
	return ret, nil
}

func (s *server) GetChat(ctx context.Context, in *pb.ChatRequest) (*pb.ChatReply, error) {
	tracer.Trace(time.Now().UTC(), in)
	var chats []*pb.Chat
	const pageSize = 100
	q := datastore.NewQuery("Chat").Filter("GameId =", in.Chat.GetGameId()).Order("Created").Limit(pageSize)
	ds.GetAll(ctx, q, &chats)
	ret := &pb.ChatReply{Chats: chats, Cursor: ""}
	tracer.Trace(time.Now().UTC(), ret)
	return ret, nil
}

// update my chat
func (s *server) AddChatMessage(ctx context.Context, in *pb.ChatMessageRequest) (*pb.ChatReply, error) {
	tracer.Trace(time.Now().UTC(), in)
	var Chat pb.Chat
	// get
	key := datastore.NameKey("Chat", strconv.FormatInt(in.GetGameId()+in.GetAccountId(), 10), nil)
	ds.Get(ctx, key, &Chat)
	// append & put
	Chat.AccountId = in.GetAccountId()
	Chat.GameId = in.GetGameId()

	NowUnix := time.Now().UTC().Unix()
	Chat.Updated = NowUnix
	if Chat.GetCreated() == 0 {
		Chat.Created = NowUnix
	}
	in.ChatMessage.Created = NowUnix
	in.ChatMessage.AccountId = in.GetAccountId()
	Chat.ChatMessages = append(Chat.ChatMessages, in.ChatMessage)
	ds.Put(ctx, key, &Chat)
	// return all chats
	var chats []*pb.Chat
	const pageSize = 100
	q := datastore.NewQuery("Chat").Filter("GameId =", in.GetGameId()).Order("Created").Limit(pageSize)
	ds.GetAll(ctx, q, &chats)
	log.Printf(strconv.FormatInt(in.GetGameId(), 10))
	ret := &pb.ChatReply{Chats: chats}
	_ctx := context.Background()
	go setChatPush(_ctx, in.GameId, in.GetAccountId(), in.ChatMessage.Message)
	tracer.Trace(time.Now().UTC(), ret)
	return ret, nil
}

func (s *server) GetDataPlace(ctx context.Context, in *pb.DataPlaceRequest) (*pb.DataPlaceReply, error) {
	tracer.Trace(time.Now().UTC(), in)
	names, address, dataTime, serverPlaceVersion := data.Get(in.Version)
	ret := &pb.DataPlaceReply{Version: serverPlaceVersion, Names: names, Address: address, Time: dataTime}
	tracer.Trace(time.Now().UTC(), ret)
	return ret, nil
}

func setJoinRequestPush(ctx context.Context, in *pb.JoinRequest) {
	var game pb.Game
	var profile pb.Profile
	var apnsTokens []string
	ds.Get(ctx, datastore.IDKey("Game", in.Join.GetGameId(), nil), &game)
	ds.Get(ctx, datastore.IDKey("Profile", game.GetHostAccountId(), nil), &profile)
	if game.GetHostAccountId() != in.Join.AccountId {
		apnsTokens = append(apnsTokens, profile.ApnsToken)
		pushNotification(apnsTokens, "클럽하우스", game.PlaceName, "조인 신청이 도착했습니다.")
	} else {

	}
}

func difference(a, b []int64) int64 {
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

func setJoinChangePush(ctx context.Context, in *pb.GameRequest, before *pb.Game) {
	var accountID int64
	var changeStatus = ""
	if len(in.Game.AcceptAccountIds) > len(before.AcceptAccountIds) {
		accountID = difference(in.Game.AcceptAccountIds, before.AcceptAccountIds)
		changeStatus = "수락"
	}
	if len(in.Game.RejectAccountIds) > len(before.RejectAccountIds) {
		accountID = difference(in.Game.RejectAccountIds, before.RejectAccountIds)
		changeStatus = "거절"
	}
	if changeStatus != "" {
		var profile pb.Profile
		var apnsTokens []string
		ds.Get(ctx, datastore.IDKey("Profile", accountID, nil), &profile)
		apnsTokens = append(apnsTokens, profile.ApnsToken)
		pushNotification(apnsTokens, "클럽하우스", in.Game.PlaceName, "조인 신청이 "+changeStatus+"됐습니다.")
	}
}

func setChatPush(ctx context.Context, gameID int64, accountID int64, message string) {
	if message != "" {
		// accept account all
		var game pb.Game
		ds.Get(ctx, datastore.IDKey("Game", gameID, nil), &game)
		var apnsTokens []string
		game.AcceptAccountIds = append(game.AcceptAccountIds, game.HostAccountId)
		for _, x := range game.AcceptAccountIds {
			var profile pb.Profile
			if x != accountID {
				ds.Get(ctx, datastore.IDKey("Profile", x, nil), &profile)
				apnsTokens = append(apnsTokens, profile.ApnsToken)
			}
		}
		if len(apnsTokens) > 0 {
			pushNotification(apnsTokens, "클럽하우스", game.PlaceName, message)
		}
	}
}

func pushNotification(apnsTokens []string, title string, subtitle string, body string) {
	const (
		DevelopmentGateway = "https://api.sandbox.push.apple.com"
		ProductionGateway  = "https://api.push.apple.com"
	)
	GateWay := DevelopmentGateway
	if is_production == "true" {
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
	tracer.Trace(time.Now().UTC(), in)
	var article = in.Article
	article.Created = time.Now().UTC().Unix()
	log.Print("cret")
	put := ds.Put(ctx, datastore.IncompleteKey("Article", nil), article)
	article.Id = put.ID
	ret := &pb.ArticleReply{Article: article}
	tracer.Trace(time.Now().UTC(), ret)
	return ret, nil
}

func (s *server) UpdateArticle(ctx context.Context, in *pb.ArticleRequest) (*pb.ArticleReply, error) {
	tracer.Trace(time.Now().UTC(), in)
	in.Article.Updated = time.Now().UTC().Unix()
	_ = ds.Put(ctx, datastore.IDKey("Article", in.Article.GetId(), nil), in.Article)
	ret := &pb.ArticleReply{Article: in.GetArticle()}
	tracer.Trace(time.Now().UTC(), ret)
	return ret, nil
}

func (s *server) GetFilterdArticles(ctx context.Context, in *pb.FilterdArticlesRequest) (*pb.FilterdArticlesReply, error) {
	tracer.Trace(time.Now().UTC(), in)
	client := ds.GetClient(ctx)
	cursorStr := in.Cursor
	const pageSize = 10
	queryBase := datastore.NewQuery("Article")
	// todo re article
	query := queryBase.Order("Created").Filter("Category =", in.Category).Filter("Type = ", in.Type).Limit(pageSize)
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
	tracer.Trace(time.Now().UTC(), ret)
	return ret, nil
}

func (s *server) GetCount(ctx context.Context, in *pb.Count) (*pb.Count, error) {
	tracer.Trace(time.Now().UTC(), in)
	var count pb.Count
	ds.Get(ctx, datastore.IDKey(in.Kind, in.ForeginId, nil), &count)
	ret := &pb.Count{Count: count.Count, ForeginId: in.ForeginId, Kind: in.Kind}
	tracer.Trace(time.Now().UTC(), ret)
	return ret, nil
}

func (s *server) CreateLike(ctx context.Context, in *pb.LikeRequest) (*pb.LikeReply, error) {
	tracer.Trace(time.Now().UTC(), in)
	in.Like.Created = time.Now().UTC().Unix()
	log.Print(in.Like.GetForeginAccountId())
	_ = ds.Put(ctx, datastore.IncompleteKey("Like", nil), in.Like)
	// just counting
	go CountIncr(context.Background(), in.Like.GetForeginId(), "Like")
	go setAccountIdPush(context.Background(), in.Like.GetForeginAccountId(), in.Like.GetTitle(), "+1 좋아합니다.")
	ret := &pb.LikeReply{Like: in.Like}
	tracer.Trace(time.Now().UTC(), ret)
	return ret, nil
}

// 계정 아이디로 푸시 발송
func setAccountIdPush(ctx context.Context, id int64, title string, body string) {
	var profile pb.Profile
	var apnsTokens []string
	ds.Get(ctx, datastore.IDKey("Profile", id, nil), &profile)
	apnsTokens = append(apnsTokens, profile.ApnsToken)
	pushNotification(apnsTokens, "클럽하우스", title, body)
}

func (s *server) UpdateLike(ctx context.Context, in *pb.LikeRequest) (*pb.LikeReply, error) {
	tracer.Trace(time.Now().UTC(), in)
	in.Like.Created = time.Now().UTC().Unix()
	_ = ds.Put(ctx, datastore.IncompleteKey("Like", nil), in.Like)
	ret := &pb.LikeReply{Like: in.Like}
	tracer.Trace(time.Now().UTC(), ret)
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
