// Package netmgr lista e configura as interfaces de rede da plataforma que
// hospeda o gateway. A leitura funciona em qualquer SO (net.Interfaces); as
// informações extras (velocidade, duplex, contadores) e a escrita (up/down,
// endereços IP, gateway) usam sysfs e o comando `ip` e só existem em Linux.
package netmgr

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ErrUnsupported é devolvido quando a operação de escrita não é suportada no SO atual
var ErrUnsupported = errors.New("configuração de rede suportada apenas em Linux")

// Interface descreve uma interface de rede
type Interface struct {
	Name      string   `json:"name"`
	Kind      string   `json:"kind"` // "ethernet", "loopback", "virtual"
	MAC       string   `json:"mac"`
	MTU       int      `json:"mtu"`
	Up        bool     `json:"up"`        // administrativamente ligada (flag UP)
	Carrier   *bool    `json:"carrier"`   // cabo/link detectado (nil se desconhecido)
	OperState string   `json:"oper_state"` // up, down, unknown...
	SpeedMbps *int     `json:"speed_mbps"`
	Duplex    string   `json:"duplex"`
	Addresses []string `json:"addresses"` // CIDR
	RxBytes   *uint64  `json:"rx_bytes"`
	TxBytes   *uint64  `json:"tx_bytes"`
	Gateway   string   `json:"gateway"`
}

// Info agrega as interfaces e metadados da plataforma
type Info struct {
	OS         string      `json:"os"`
	Writable   bool        `json:"writable"` // true se a edição está disponível
	Interfaces []Interface `json:"interfaces"`
}

func sysRead(iface, file string) (string, bool) {
	b, err := os.ReadFile(filepath.Join("/sys/class/net", iface, file))
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(b)), true
}

func sysExists(iface, file string) bool {
	_, err := os.Stat(filepath.Join("/sys/class/net", iface, file))
	return err == nil
}

// defaultGateways devolve interface -> gateway padrão (Linux, via /proc/net/route)
func defaultGateways() map[string]string {
	out := map[string]string{}
	b, err := os.ReadFile("/proc/net/route")
	if err != nil {
		return out
	}
	lines := strings.Split(string(b), "\n")
	for _, l := range lines[1:] {
		f := strings.Fields(l)
		if len(f) < 4 || f[1] != "00000000" {
			continue
		}
		gw, err := strconv.ParseUint(f[2], 16, 32)
		if err != nil {
			continue
		}
		out[f[0]] = fmt.Sprintf("%d.%d.%d.%d", gw&0xff, (gw>>8)&0xff, (gw>>16)&0xff, (gw>>24)&0xff)
	}
	return out
}

// List devolve todas as interfaces da máquina
func List() (Info, error) {
	ifs, err := net.Interfaces()
	if err != nil {
		return Info{}, err
	}
	gws := defaultGateways()
	info := Info{OS: runtime.GOOS, Writable: runtime.GOOS == "linux", Interfaces: []Interface{}}

	for _, ni := range ifs {
		it := Interface{
			Name:      ni.Name,
			MAC:       ni.HardwareAddr.String(),
			MTU:       ni.MTU,
			Up:        ni.Flags&net.FlagUp != 0,
			Addresses: []string{},
			OperState: "unknown",
			Gateway:   gws[ni.Name],
		}
		if addrs, err := ni.Addrs(); err == nil {
			for _, a := range addrs {
				it.Addresses = append(it.Addresses, a.String())
			}
		}

		switch {
		case ni.Flags&net.FlagLoopback != 0:
			it.Kind = "loopback"
		case runtime.GOOS == "linux":
			t, _ := sysRead(ni.Name, "type")
			if t == "1" && sysExists(ni.Name, "device") && !sysExists(ni.Name, "wireless") {
				it.Kind = "ethernet"
			} else {
				it.Kind = "virtual"
			}
		default:
			it.Kind = "ethernet"
		}

		if runtime.GOOS == "linux" {
			if v, ok := sysRead(ni.Name, "operstate"); ok {
				it.OperState = v
			}
			if v, ok := sysRead(ni.Name, "carrier"); ok {
				c := v == "1"
				it.Carrier = &c
			}
			if v, ok := sysRead(ni.Name, "speed"); ok {
				if n, err := strconv.Atoi(v); err == nil && n > 0 {
					it.SpeedMbps = &n
				}
			}
			if v, ok := sysRead(ni.Name, "duplex"); ok {
				it.Duplex = v
			}
			if v, ok := sysRead(ni.Name, "statistics/rx_bytes"); ok {
				if n, err := strconv.ParseUint(v, 10, 64); err == nil {
					it.RxBytes = &n
				}
			}
			if v, ok := sysRead(ni.Name, "statistics/tx_bytes"); ok {
				if n, err := strconv.ParseUint(v, 10, 64); err == nil {
					it.TxBytes = &n
				}
			}
		} else {
			// Fora do Linux o estado vem apenas das flags
			if it.Up {
				it.OperState = "up"
			} else {
				it.OperState = "down"
			}
		}
		info.Interfaces = append(info.Interfaces, it)
	}
	sort.Slice(info.Interfaces, func(i, j int) bool { return info.Interfaces[i].Name < info.Interfaces[j].Name })
	return info, nil
}

func exists(name string) error {
	if _, err := net.InterfaceByName(name); err != nil {
		return fmt.Errorf("interface '%s' não encontrada", name)
	}
	return nil
}

func ip(args ...string) error {
	if runtime.GOOS != "linux" {
		return ErrUnsupported
	}
	cmd := exec.Command("ip", args...)
	done := make(chan struct{})
	var out []byte
	var err error
	go func() {
		out, err = cmd.CombinedOutput()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		_ = cmd.Process.Kill()
		return errors.New("timeout executando 'ip'")
	}
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("ip %s: %s", strings.Join(args, " "), msg)
	}
	return nil
}

// SetState liga ou desliga a interface
func SetState(name string, up bool) error {
	if err := exists(name); err != nil {
		return err
	}
	state := "down"
	if up {
		state = "up"
	}
	return ip("link", "set", "dev", name, state)
}

// parseCIDR valida e normaliza um endereço IPv4 em formato CIDR
func parseCIDR(cidr string) (string, error) {
	ipAddr, n, err := net.ParseCIDR(strings.TrimSpace(cidr))
	if err != nil {
		return "", fmt.Errorf("endereço inválido (use o formato 192.168.0.10/24): %v", err)
	}
	if ipAddr.To4() == nil {
		return "", errors.New("apenas IPv4 é suportado")
	}
	ones, _ := n.Mask.Size()
	return fmt.Sprintf("%s/%d", ipAddr.String(), ones), nil
}

// AddAddress adiciona um IP (CIDR) à interface; gateway é opcional
func AddAddress(name, cidr, gateway string) error {
	if err := exists(name); err != nil {
		return err
	}
	c, err := parseCIDR(cidr)
	if err != nil {
		return err
	}
	gateway = strings.TrimSpace(gateway)
	if gateway != "" && net.ParseIP(gateway) == nil {
		return errors.New("gateway inválido")
	}
	if err := ip("addr", "add", c, "dev", name); err != nil {
		return err
	}
	if gateway != "" {
		return ip("route", "replace", "default", "via", gateway, "dev", name)
	}
	return nil
}

// RemoveAddress remove um IP (CIDR) da interface
func RemoveAddress(name, cidr string) error {
	if err := exists(name); err != nil {
		return err
	}
	c, err := parseCIDR(cidr)
	if err != nil {
		return err
	}
	return ip("addr", "del", c, "dev", name)
}

// EditAddress substitui um endereço IP específico por um novo (em tempo real)
func EditAddress(name, oldCIDR, newCIDR, gateway string) error {
	if err := exists(name); err != nil {
		return err
	}
	newC, err := parseCIDR(newCIDR)
	if err != nil {
		return err
	}
	oldC, err := parseCIDR(oldCIDR)
	if err != nil {
		return err
	}
	// Se for exatamente o mesmo endereço e apenas gateway mudou
	if oldC != newC {
		// Adiciona o novo endereço primeiro para garantir que a interface não fique sem IP se for o mesmo subnet
		if err := ip("addr", "add", newC, "dev", name); err != nil {
			return fmt.Errorf("falha ao adicionar novo endereço: %w", err)
		}
		// Remove o endereço antigo
		_ = ip("addr", "del", oldC, "dev", name)
	}

	gateway = strings.TrimSpace(gateway)
	if gateway != "" {
		if net.ParseIP(gateway) == nil {
			return errors.New("gateway inválido")
		}
		if err := ip("route", "replace", "default", "via", gateway, "dev", name); err != nil {
			return fmt.Errorf("falha ao atualizar gateway: %w", err)
		}
	}
	return nil
}

// SetStatic substitui todos os IPv4 da interface por um único endereço
func SetStatic(name, cidr, gateway string) error {
	if err := exists(name); err != nil {
		return err
	}
	if _, err := parseCIDR(cidr); err != nil {
		return err
	}
	if err := ip("-4", "addr", "flush", "dev", name); err != nil {
		return err
	}
	return AddAddress(name, cidr, gateway)
}

