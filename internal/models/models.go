package models

import "time"

// Tag representa uma variável unificada para qualquer marca de CLP
type Tag struct {
	ID        string      `json:"id"`
	DeviceID  string      `json:"device_id"`
	Name      string      `json:"name"`
	Address   string      `json:"address"`   // Útil para Rockwell (ex: "Motor_Speed")
	DBNumber  int         `json:"db_number"` // Específico para Siemens
	Offset    int         `json:"offset"`    // Específico para Siemens
	Bit       int         `json:"bit"`       // Específico para Siemens (Booleanos)
	DataType  string      `json:"data_type"` // "REAL", "INT", "BOOL"
	Value     interface{} `json:"value"`
	Quality   bool        `json:"quality"` // true se a última leitura foi bem sucedida
	Timestamp time.Time   `json:"timestamp"`
}

// Device representa o equipamento físico
type Device struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Protocol  string `json:"protocol"` // "s7" ou "cip"
	IPAddress string `json:"ip_address"`
	Rack      int    `json:"rack"`
	Slot      int    `json:"slot"`
}
