package models

// Tag representa uma variável dentro do CLP
type Tag struct {
	Name     string
	DBNumber int
	Offset   int
	DataType string // "real", "int", "bool"
	Value    interface{}
}

// Configuração básica de conexão de um CLP
type Device struct {
	IPAddress string
	Rack      int
	Slot      int
}
