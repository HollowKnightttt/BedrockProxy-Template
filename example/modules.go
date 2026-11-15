package main

import (
	"strings"
	"sync"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

const prefix = "." // chat messages starting with this are treated as commands

// commands maps a command name to its handler. Add your own commands here.
var commands = map[string]func(s *session, args []string){
	"test":    func(s *session, _ []string) { s.say("Hello, World!") },
	"fly":     func(s *session, _ []string) { s.toggleFly() },
	"airjump": func(s *session, _ []string) { s.toggleAirJump() },
}

// session holds the state of one connected player and the module toggles.
type session struct {
	client *minecraft.Conn
	server *minecraft.Conn

	mu        sync.Mutex
	fly       bool
	airJump   bool
	jumpHeld  bool                    // jump button state of the previous tick
	abilities *packet.UpdateAbilities // last unmodified abilities the server sent us
}

// fromClient is called for every packet the player sends. Return false to drop it.
func (s *session) fromClient(pk packet.Packet) bool {
	switch p := pk.(type) {
	case *packet.Text:
		if s.runCommand(p.Message) {
			return false
		}
	case *packet.PlayerAuthInput:
		s.airJumpTick(p)
	}
	return true
}

// fromServer is called for every packet the server sends. Return false to drop it.
func (s *session) fromServer(pk packet.Packet) bool {
	switch p := pk.(type) {
	case *packet.UpdateAbilities:
		s.onAbilities(p)
	}
	return true
}

// ---------------------------------------------------------------- fly

// Fly gives the player the "may fly" ability (double-tap jump to start flying),
// the same way creative mode does it. The server also sends its own abilities,
// so we remember the original and re-apply fly on top of it whenever it changes.

func (s *session) toggleFly() {
	s.mu.Lock()
	s.fly = !s.fly
	on := s.fly
	s.mu.Unlock()

	if !s.sendAbilities() {
		s.mu.Lock()
		s.fly = !on // undo, nothing was sent
		s.mu.Unlock()
		s.say("§cThe server hasn't sent your abilities yet, try again in a moment.")
		return
	}
	s.say("Fly: " + state(on))
}

// onAbilities handles the server's UpdateAbilities packet (server -> client).
func (s *session) onAbilities(p *packet.UpdateAbilities) {
	if !s.isSelf(p.AbilityData.EntityUniqueID) {
		return // abilities of another player
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.abilities = cloneAbilities(p) // keep the original
	if s.fly {
		enableFly(&p.AbilityData) // modify the packet that gets forwarded
	}
}

// sendAbilities pushes the current abilities (with or without fly) to the player.
func (s *session) sendAbilities() bool {
	s.mu.Lock()
	if s.abilities == nil {
		s.mu.Unlock()
		return false
	}
	pk := cloneAbilities(s.abilities)
	if s.fly {
		enableFly(&pk.AbilityData)
	}
	s.mu.Unlock()

	_ = s.client.WritePacket(pk)
	return true
}

func enableFly(d *protocol.AbilityData) {
	for i := range d.Layers {
		l := &d.Layers[i]
		if l.Type != protocol.AbilityLayerTypeBase {
			continue
		}
		l.Abilities |= protocol.AbilityMayFly // this layer defines the ability
		l.Values |= protocol.AbilityMayFly    // and it is switched on
	}
}

func cloneAbilities(p *packet.UpdateAbilities) *packet.UpdateAbilities {
	c := *p
	c.AbilityData.Layers = append([]protocol.AbilityLayer(nil), p.AbilityData.Layers...)
	return &c
}

// ------------------------------------------------------------ airjump

// AirJump watches the jump button in PlayerAuthInput. On every new press it
// sets the player's motion upwards, so you can jump again while in the air.

func (s *session) toggleAirJump() {
	s.mu.Lock()
	s.airJump = !s.airJump
	on := s.airJump
	s.mu.Unlock()
	s.say("AirJump: " + state(on))
}

func (s *session) airJumpTick(p *packet.PlayerAuthInput) {
	down := p.InputData.Load(packet.InputFlagJumpDown)

	s.mu.Lock()
	pressed := down && !s.jumpHeld // only the tick the button goes down
	s.jumpHeld = down
	enabled := s.airJump
	s.mu.Unlock()

	if !enabled || !pressed {
		return
	}
	_ = s.client.WritePacket(&packet.SetActorMotion{
		EntityRuntimeID: s.client.GameData().EntityRuntimeID,
		Motion:          mgl32.Vec3{p.Delta.X(), 0.42, p.Delta.Z()}, // keep horizontal speed, 0.42 = normal jump
		Tick:            p.Tick,
	})
}

// ------------------------------------------------------------ helpers

// runCommand returns true if msg was a proxy command (so it must not reach the server).
func (s *session) runCommand(msg string) bool {
	if !strings.HasPrefix(msg, prefix) {
		return false
	}
	fields := strings.Fields(strings.TrimPrefix(msg, prefix))
	if len(fields) == 0 {
		return false
	}
	handler, ok := commands[strings.ToLower(fields[0])]
	if !ok {
		s.say("§cUnknown command: " + fields[0])
		return true
	}
	handler(s, fields[1:])
	return true
}

// say shows a chat line to the player, sent by the proxy itself.
func (s *session) say(msg string) {
	_ = s.client.WritePacket(&packet.Text{
		TextType: packet.TextTypeRaw,
		Message:  msg,
	})
}

func (s *session) isSelf(id int64) bool {
	gd := s.client.GameData()
	return id == gd.EntityUniqueID || uint64(id) == gd.EntityRuntimeID
}

func state(on bool) string {
	if on {
		return "§aON"
	}
	return "§cOFF"
}
