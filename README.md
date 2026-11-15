# Template Bedrock Proxy
Template For People That want to make MCBE Proxy

# Quick Start
```go
go mod init bedrockproxy
go mod tidy
go run . -remote ip.server:19132
```
# Packet
**Chat and commands** - `Text`, `CommandRequest`, `CommandOutput`, `AvailableCommands` <br />
**Movement** - `PlayerAuthInput` (modern client motion input), `MovePlayer`, `MoveActorAbsolute`, `MoveActorDelta`, `SetActorMotion`, `CorrectPlayerMovePrediction` <br />
**Combat and action** - `InventoryTransaction` (hit entity, use item, place block), `Animate`, `Interact`, `PlayerAction`, `ActorEvent`, `UpdateAttributes`, `SetHealth`, `MobEffect`, `Respawn` <br />
**Entities and players** - `AddPlayer`, `AddActor`, `AddItemActor`, `RemoveActor`, `SetActorData`, `PlayerList` 
**World** - `LevelChunk`, `SubChunk`, `UpdateBlock`, `UpdateBlockSynced`, `NetworkChunkPublisherUpdate`, `BlockActorData`, `LevelEvent`, `LevelSoundEvent`, `SetTime`, `ChangeDimension` <br />
**Inventory** - `InventoryContent`, `InventorySlot`, `MobEquipment`, `MobArmourEquipment` (British English spelling), `ItemStackRequest`, `ItemStackResponse`, `ContainerOpen`, `ContainerClose` <br />
**UI and effects** - `SetTitle`, `ToastRequest`, `ModalFormRequest`, `ModalFormResponse`, `BossEvent`, `SetScore`, `SetDisplayObjective`, `PlaySound`, `SpawnParticleEffect`, `Transfer` <br />
**Handshake (it's best not to mess with it, it's already taken care of by Gophertunnel)** - `Login`, `NetworkSettings`, `ServerToClientHandshake`, `ResourcePacksInfo`, `ResourcePackStack`, `PlayStatus`, `StartGame`, `SetLocalPlayerAsInitialised`, `Disconnect`
