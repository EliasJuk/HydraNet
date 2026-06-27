package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

type ScanData struct {
	IP        string    `json:"ip"`
	Port      int       `json:"port"`
	Timestamp time.Time `json:"timestamp"`
	Banner    string    `json:"banner"`
}

var blocosIP = []string{
	"200.9.0.0/30",  // Apenas 4 IPs
	"200.17.0.0/30", // Apenas 4 IPs
}

func main() {
	portsToScan := []int{21, 22, 80, 443, 554, 8080, 8081} // FTP - SSH - HTTP - HTTPS - RTSP
	outputFile := "scan_simples.jsonl"

	fmt.Println("[+] Gerando lista de IPs a partir dos blocos")
	listaIps := expandirCIDRs(blocosIP)
	fmt.Printf("[+] %d IPs gerados para testar.\n\n", len(listaIps))

	// Abre o arquivo de texto uma única vez no início
	file, err := os.OpenFile(outputFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("Erro ao abrir arquivo: %v\n", err)
		return
	}
	defer file.Close()

	// LOOP 1: Passa por cada IP, um por um
	for _, ip := range listaIps {
		// LOOP 2: Para o IP atual, passa por cada porta
		for _, port := range portsToScan {

			target := fmt.Sprintf("%s:%d", ip, port)
			fmt.Printf("[%s] Tentando -> %s\n", time.Now().Format("15:04:05"), target)

			// Tenta conectar (timeout de 1 segundo para não travar o código se estiver fechada)
			conn, err := net.DialTimeout("tcp", target, 1*time.Second)
			if err != nil {
				// Se deu erro, a porta está fechada ou o IP está offline. Pula para a próxima.
				continue
			}

			// Se chegou aqui, a conexão foi bem-sucedida! A porta está aberta.
			fmt.Printf("    [!] PORTA ABERTA encontrada em %s!\n", target)

			banner := grabBanner(conn, port)
			conn.Close() // Fecha a conexão de rede após pegar o banner

			// Cria a estrutura com os dados obtidos
			resultado := ScanData{
				IP:        ip,
				Port:      port,
				Timestamp: time.Now(),
				Banner:    banner,
			}

			// Converte para JSON e salva diretamente no arquivo
			jsonData, _ := json.Marshal(resultado)
			_, _ = file.Write(append(jsonData, '\n'))
		}
	}

	fmt.Println("\n[+] Varredura concluída! Verifique o arquivo:", outputFile)
}

func grabBanner(conn net.Conn, port int) string {
	// Define um limite de tempo para ler o banner (1.5 segundos)
	_ = conn.SetReadDeadline(time.Now().Add(1500 * time.Millisecond))

	// Se for HTTP (porta 80), precisamos enviar algo para o servidor responder
	if port == 80 {
		_, _ = conn.Write([]byte("HEAD / HTTP/1.1\r\nHost: localhost\r\n\r\n"))
	}

	reader := bufio.NewReader(conn)
	buffer := make([]byte, 256)
	n, err := reader.Read(buffer)
	if err != nil && n == 0 {
		return "[Porta aberta - Sem banner]"
	}

	// Limpa o texto recebido
	bannerClean := string(buffer[:n])
	bannerClean = strings.ReplaceAll(bannerClean, "\n", " ")
	bannerClean = strings.ReplaceAll(bannerClean, "\r", " ")
	return strings.TrimSpace(bannerClean)
}

func expandirCIDRs(cidrs []string) []string {
	var ips []string
	for _, cidr := range cidrs {
		ip, ipnet, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); incrementaIP(ip) {
			ips = append(ips, ip.String())
		}
	}
	return ips
}

func incrementaIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j] = ip[j] + 1
		if ip[j] > 0 {
			break
		}
	}
}
