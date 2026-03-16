package ports

import "context"

type ServerRecord struct {
	ID         string
	Name       string
	Address    string
	Port       int
	Status     string
	NodeID     string
	Runtime    string
	RuntimeRef string
}

type ContainerRuntime interface {
	Provision(ctx context.Context, name string, payload ProvisionPayload) (*ServerRecord, error)
}

type ProvisionPayload struct {
	ServerName   string
	Type         string
	Version      string
	MaxPlayers   int
	Difficulty   string
	Gamemode     string
	ViewDistance int
	WorldSeed    string
	OnlineMode   bool
	Memory       string
	Storage      string
}
