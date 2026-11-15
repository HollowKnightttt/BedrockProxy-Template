package main

import (
	"encoding/json"
	"errors"
	"flag"
	"log"
	"os"
	"sync"

	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/auth"
	"golang.org/x/oauth2"
)

const tokenFile = "token.tok" // cached Xbox login

func main() {
	local := flag.String("local", "0.0.0.0:19132", "address the proxy listens on")
	remote := flag.String("remote", "", "real server address (host:port)")
	flag.Parse()
	if *remote == "" {
		log.Fatal("usage: proxy -remote host:port [-local 0.0.0.0:19132]")
	}

	src := tokenSource()

	status, err := minecraft.NewForeignStatusProvider(*remote)
	if err != nil {
		log.Fatalf("status provider: %v", err)
	}
	listener, err := minecraft.ListenConfig{StatusProvider: status}.Listen("raknet", *local)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	defer listener.Close()
	log.Printf("proxy listening on %s -> %s", *local, *remote)

	for {
		c, err := listener.Accept()
		if err != nil {
			log.Printf("accept: %v", err)
			return
		}
		go handleConn(c.(*minecraft.Conn), listener, *remote, src)
	}
}

func handleConn(client *minecraft.Conn, listener *minecraft.Listener, remote string, src oauth2.TokenSource) {
	server, err := minecraft.Dialer{
		TokenSource: src,
		ClientData:  client.ClientData(),
	}.Dial("raknet", remote)
	if err != nil {
		log.Printf("dial server: %v", err)
		_ = listener.Disconnect(client, "could not connect to the server")
		return
	}

	var (
		wg                 sync.WaitGroup
		startErr, spawnErr error
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		startErr = client.StartGame(server.GameData())
	}()
	go func() {
		defer wg.Done()
		spawnErr = server.DoSpawn()
	}()
	wg.Wait()
	if startErr != nil || spawnErr != nil {
		log.Printf("spawn failed: client=%v server=%v", startErr, spawnErr)
		_ = server.Close()
		_ = listener.Disconnect(client, "spawn failed")
		return
	}

	s := &session{client: client, server: server}

	// client -> server
	go func() {
		defer server.Close()
		defer listener.Disconnect(client, "connection lost")
		for {
			pk, err := client.ReadPacket()
			if err != nil {
				return
			}
			if !s.fromClient(pk) {
				continue // dropped by a module, don't forward
			}
			if err := server.WritePacket(pk); err != nil {
				var disc minecraft.DisconnectError
				if errors.As(err, &disc) {
					_ = listener.Disconnect(client, disc.Error())
				}
				return
			}
		}
	}()

	// server -> client
	go func() {
		defer server.Close()
		defer listener.Disconnect(client, "connection lost")
		for {
			pk, err := server.ReadPacket()
			if err != nil {
				var disc minecraft.DisconnectError
				if errors.As(err, &disc) {
					_ = listener.Disconnect(client, disc.Error())
				}
				return
			}
			if !s.fromServer(pk) {
				continue
			}
			if err := client.WritePacket(pk); err != nil {
				return
			}
		}
	}()
}

func tokenSource() oauth2.TokenSource {
	token := new(oauth2.Token)
	if data, err := os.ReadFile(tokenFile); err == nil {
		_ = json.Unmarshal(data, token)
	} else {
		token = login()
	}
	src := auth.RefreshTokenSource(token)
	if _, err := src.Token(); err != nil {
		// cached token expired, log in again
		token = login()
		src = auth.RefreshTokenSource(token)
	}
	if tok, err := src.Token(); err == nil {
		if data, err := json.Marshal(tok); err == nil {
			_ = os.WriteFile(tokenFile, data, 0o600)
		}
	}
	return src
}

func login() *oauth2.Token {
	token, err := auth.RequestLiveToken()
	if err != nil {
		log.Fatalf("xbox login failed: %v", err)
	}
	return token
}
