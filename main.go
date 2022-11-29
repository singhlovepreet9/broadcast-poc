package main

import (
	context "context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"os"
	"time"
	tx "tx-poc/txproto"

	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	port  = os.Getenv("PORT")
	ports = []string{
		"50001", "50002", "50003", "50004",
	}
	filename = func() string {
		return os.Getenv("FILENAME")
	}()
	transactions = make(map[string]int)
)

type Server struct {
	tx.UnimplementedTransactionsServer
}

func broadCast(payload string, p string) {
	if port == p {
		return
	}
	nodeURL := "localhost:" + p

	conn, err := grpc.Dial(nodeURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	c := tx.NewTransactionsClient(conn)

	ctxn, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, e := c.SendTx(ctxn, &tx.TxRequest{Payload: payload})

	if e != nil {
		print("errror", e)
	}

}

func writeToFile(payload string, txid string) (string, error) {
	data := txid + "," + payload + "\n"

	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := f.Write([]byte(data)); err != nil {
		f.Close() // ignore error; Write error takes precedence
		log.Fatal(err)
	}
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}
	return "Successful written to file", nil
}

// type TxPayload struct {
// 	From   string  `json:from`
// 	To     string  `json:to`
// 	Amount float64 `json:amount`
// }

func (s *Server) SendTx(ctx context.Context, in *tx.TxRequest) (*tx.TxReply, error) {

	// var payload TxPayload
	// nn, _ := json.Marshal()

	payloadString := in.GetPayload()
	// json.Unmarshal([]byte(payloadString), &payload)

	txHash := sha256.Sum256([]byte(payloadString))

	fmt.Println("request", payloadString)
	txid := hex.EncodeToString(txHash[:])
	log.Printf("Received: %v %v %+v", payloadString, txid, transactions)

	if _, ok := transactions[txid]; ok {
		writeToFile(payloadString, txid)
		return &tx.TxReply{Body: "Recieved " + txid}, nil
	}
	transactions[txid] = 1
	for _, nodePort := range ports {
		broadCast(payloadString, nodePort)
	}

	return &tx.TxReply{Body: "Recieved " + txid}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	if err != nil {
		panic(err)
	}

	s := grpc.NewServer()

	tx.RegisterTransactionsServer(s, &Server{})
	log.Printf("server listening at %v", port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

}
