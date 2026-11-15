# Template Bedrock Proxy
Template For People That want to make MCBE Proxy

# Quick Start
```go
go mod init bedrockproxy
go mod tidy
go run . -remote ip.server:19132
```
# Packet
**Chat and commands** <br /> `Text`, `CommandRequest`, `CommandOutput`, `AvailableCommands` <br />
**Movement** <br /> `PlayerAuthInput` (modern client motion input), `MovePlayer`, `MoveActorAbsolute`, `MoveActorDelta`, `SetActorMotion`, `CorrectPlayerMovePrediction` <br />
**Combat and action** <br /> `InventoryTransaction` (hit entity, use item, place block), `Animate`, `Interact`, `PlayerAction`, `ActorEvent`, `UpdateAttributes`, `SetHealth`, `MobEffect`, `Respawn` <br />
**Entities and players** <br /> `AddPlayer`, `AddActor`, `AddItemActor`, `RemoveActor`, `SetActorData`, `PlayerList` 
**World** <br /> `LevelChunk`, `SubChunk`, `UpdateBlock`, `UpdateBlockSynced`, `NetworkChunkPublisherUpdate`, `BlockActorData`, `LevelEvent`, `LevelSoundEvent`, `SetTime`, `ChangeDimension` <br />
**Inventory** <br /> `InventoryContent`, `InventorySlot`, `MobEquipment`, `MobArmourEquipment` (British English spelling), `ItemStackRequest`, `ItemStackResponse`, `ContainerOpen`, `ContainerClose` <br />
**UI and effects** <br /> `SetTitle`, `ToastRequest`, `ModalFormRequest`, `ModalFormResponse`, `BossEvent`, `SetScore`, `SetDisplayObjective`, `PlaySound`, `SpawnParticleEffect`, `Transfer` <br />
**Handshake (it's best not to mess with it, it's already taken care of by Gophertunnel)** - `Login`, `NetworkSettings`, `ServerToClientHandshake`, `ResourcePacksInfo`, `ResourcePackStack`, `PlayStatus`, `StartGame`, `SetLocalPlayerAsInitialised`, `Disconnect`<br />
<br />
To use it in a proxy, implement a type switch within the `ReadPacket` loop:
```go
switch p := pk.(type) {
case *packet.PlayerAuthInput:
	// p.Position, p.Yaw, p.Pitch, p.InputData
case *packet.UpdateAttributes:
  // can be read or modified before being forwarded
}
```
Within a switch, you can read a packet, modify its fields, discard it using `continue`, or send a new packet via `WritePacket`. The most accurate way to check the field names for each packet is by referring to pkg.go.dev, as these may change between different versions of gophertunnel.
