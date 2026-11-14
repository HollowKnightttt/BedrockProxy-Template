package main

import (
	"encoding/json"
	"errors"
	"flag"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/auth"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"golang.org/x/oauth2"
)

const (
	prefix    = "."         // chat messages starting with this are treated as commands
	tokenFile = "token.tok" // cached Xbox login
)

// commands maps a command name to its handler. Add your own commands here.
var commands = map[string]func(client *minecraft.Conn, args []string){
	"test": func(client *minecraft.Conn, args []string) {
		sendMessage(client, "Hello, World!")
	},
}

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

	// client -> server
	go func() {
		defer server.Close()
		defer listener.Disconnect(client, "connection lost")
		for {
			pk, err := client.ReadPacket()
			if err != nil {
				return
			}
			if text, ok := pk.(*packet.Text); ok && runCommand(client, text.Message) {
				continue // command handled by the proxy, don't forward it
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
			if err := client.WritePacket(pk); err != nil {
				return
			}
		}
	}()
}

// runCommand returns true if msg was a proxy command (so it must not reach the server).
func runCommand(client *minecraft.Conn, msg string) bool {
	if !strings.HasPrefix(msg, prefix) {
		return false
	}
	fields := strings.Fields(strings.TrimPrefix(msg, prefix))
	if len(fields) == 0 {
		return false
	}
	handler, ok := commands[strings.ToLower(fields[0])]
	if !ok {
		sendMessage(client, "§cUnknown command: "+fields[0])
		return true
	}
	handler(client, fields[1:])
	return true
}

// sendMessage shows a chat line to the player, sent by the proxy itself.
func sendMessage(client *minecraft.Conn, msg string) {
	_ = client.WritePacket(&packet.Text{
		TextType: packet.TextTypeRaw,
		Message:  msg,
	})
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
