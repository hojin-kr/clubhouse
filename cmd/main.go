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

	"cloud.google.com/go/datastore"
	ds "github.com/hojin-kr/haru/cmd/ds"
	pb "github.com/hojin-kr/haru/cmd/proto"
	"github.com/hojin-kr/haru/cmd/trace"
	"google.golang.org/grpc"
)

var (
	port       = flag.Int("port", 50051, "The server port")
	project_id = os.Getenv("PROJECT_ID")
	tracer     trace.Tracer
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
	key := ds.Put(ctx, datastore.IncompleteKey("Account", nil), &pb.AccountRequest{Account: &pb.Account{RegisterTimestamp: tm}})
	ret := &pb.AccountReply{Account: &pb.Account{Id: key.ID, RegisterTimestamp: tm}}
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

func (s *server) UpdateGame(ctx context.Context, in *pb.GameRequest) (*pb.GameReply, error) {
	tracer.Trace(time.Now().UTC(), in)
	in.Game.Updated = time.Now().UTC().Unix()
	_ = ds.Put(ctx, datastore.IDKey("Game", in.Game.GetId(), nil), in.Game)
	ret := &pb.GameReply{Game: in.GetGame()}
	tracer.Trace(time.Now().UTC(), ret)
	return ret, nil
}

// filterdGames에서는 Game 목록만 반환하고 GetGame에서는 attend, place 부가 정보 반환
func (s *server) GetFilterdGames(ctx context.Context, in *pb.FilterdGamesRequest) (*pb.FilterdGamesReply, error) {
	tracer.Trace(time.Now().UTC(), in)
	// q := datastore.NewQuery("Game").Filter("A =", 12).Limit(30)
	var games []*pb.Game
	q := datastore.NewQuery("Game").Limit(30)
	ds.GetAll(ctx, q, &games)
	ret := &pb.FilterdGamesReply{Games: games}
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

func (s *server) GetMyJoins(ctx context.Context, in *pb.JoinRequest) (*pb.JoinReply, error) {
	tracer.Trace(time.Now().UTC(), in)
	var joins []*pb.Join
	q := datastore.NewQuery("Join").Filter("AccountId =", in.Join.GetAccountId()).Limit(100)
	ds.GetAll(ctx, q, &joins)
	ret := &pb.JoinReply{Joins: joins}
	tracer.Trace(time.Now().UTC(), ret)
	return ret, nil
}

func (s *server) GetGameJoins(ctx context.Context, in *pb.JoinRequest) (*pb.JoinReply, error) {
	tracer.Trace(time.Now().UTC(), in)
	var joins []*pb.Join
	q := datastore.NewQuery("Join").Filter("GameId =", in.Join.GetGameId()).Limit(100)
	ds.GetAll(ctx, q, &joins)
	ret := &pb.JoinReply{Joins: joins}
	tracer.Trace(time.Now().UTC(), ret)
	return ret, nil
}

func (s *server) GetChat(ctx context.Context, in *pb.ChatRequest) (*pb.ChatReply, error) {
	tracer.Trace(time.Now().UTC(), in)
	var chats []*pb.Chat
	q := datastore.NewQuery("Chat").Filter("GameId =", in.Chat.GetGameId()).Limit(100)
	log.Printf(strconv.FormatInt(in.Chat.GetGameId(), 10))
	ds.GetAll(ctx, q, &chats)
	ret := &pb.ChatReply{Chats: chats}
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
	q := datastore.NewQuery("Chat").Filter("GameId =", in.GetGameId()).Limit(100)
	ds.GetAll(ctx, q, &chats)
	log.Printf(strconv.FormatInt(in.GetGameId(), 10))
	ret := &pb.ChatReply{Chats: chats}
	tracer.Trace(time.Now().UTC(), ret)
	return ret, nil
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
