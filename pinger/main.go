package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

// Config содержит конфигурацию приложения
type Config struct {
	BackendURL     string
	PingInterval   time.Duration
	RetryAttempts  int
	RetryDelay     time.Duration
	PingTimeout    time.Duration
	DockerEndpoint string
}

// PingResult представляет результат пинга контейнера
type PingResult struct {
	IP          string    `json:"ip"`
	PingTime    float64   `json:"ping_time"`
	LastSuccess string    `json:"last_success"`
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	Created     string    `json:"created"`
	Error       string    `json:"error,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// Pinger управляет процессом пинга контейнеров
type Pinger struct {
	config     Config
	dockerCli  *client.Client
	httpClient *http.Client
}

// NewPinger создаёт новый экземпляр Pinger
func NewPinger(config Config) (*Pinger, error) {
	var cli *client.Client
	var err error

	if config.DockerEndpoint != "" {
		cli, err = client.NewClientWithOpts(
			client.WithHost(config.DockerEndpoint),
			client.WithAPIVersionNegotiation(),
		)
	} else {
		cli, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %v", err)
	}

	return &Pinger{
		config:    config,
		dockerCli: cli,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

// getContainers получает список всех контейнерров
func (p *Pinger) getContainers(ctx context.Context) ([]types.Container, error) {
	containers, err := p.dockerCli.ContainerList(ctx, types.ContainerListOptions{
		All: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %v", err)
	}
	return containers, nil
}

// getContainerIP получает IP-адрес контейнера через ContainerInspect, если он отсутствует в ContainerList
func (p *Pinger) getContainerIP(ctx context.Context, container types.Container) (string, error) {
	inspect, err := p.dockerCli.ContainerInspect(ctx, container.ID)
	if err != nil {
		return "", fmt.Errorf("failed to inspect container: %v", err)
	}
	for _, network := range inspect.NetworkSettings.Networks {
		if network.IPAddress != "" {
			return network.IPAddress, nil
		}
	}
	return "", fmt.Errorf("no IP address found for container %s", container.ID)
}

// pingContainer выполняет пинг отдельного контейнера
func (p *Pinger) pingContainer(ctx context.Context, container types.Container) *PingResult {
	name := "<unknown>"
	if len(container.Names) > 0 {
		name = container.Names[0]
	}
	result := &PingResult{
		Name:      name,
		Status:    container.Status,
		Created:   time.Unix(container.Created, 0).Format(time.RFC3339),
		Timestamp: time.Now(),
	}

	ip, err := p.getContainerIP(ctx, container)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.IP = ip

	pingCtx, cancel := context.WithTimeout(ctx, p.config.PingTimeout)
	defer cancel()

	start := time.Now()
	cmd := exec.CommandContext(pingCtx, "ping", "-c", "1", ip)
	if err := cmd.Run(); err != nil {
		result.Error = fmt.Sprintf("ping failed: %v", err)
	} else {
		result.LastSuccess = time.Now().Format(time.RFC3339)
	}
	result.PingTime = float64(time.Since(start).Milliseconds())

	return result
}

// sendResult отправляет результат на бэкенд с retry механизмом
func (p *Pinger) sendResult(ctx context.Context, result *PingResult) error {
	jsonData, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %v", err)
	}

	var lastErr error
	for attempt := 1; attempt <= p.config.RetryAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, "POST", p.config.BackendURL, bytes.NewBuffer(jsonData))
		if err != nil {
			return fmt.Errorf("failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := p.httpClient.Do(req)
		if err == nil {
			resp.Body.Close()
			return nil
		}

		lastErr = err
		log.Printf("Attempt %d failed: %v", attempt, err)
		if attempt < p.config.RetryAttempts {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(p.config.RetryDelay * time.Duration(attempt)):
			}
		}
	}
	return fmt.Errorf("failed after %d attempts: %v", p.config.RetryAttempts, lastErr)
}

// pingCycle выполняет один цикл пинга всех контейнеров
func (p *Pinger) pingCycle(ctx context.Context) error {
	containers, err := p.getContainers(ctx)
	if err != nil {
		return fmt.Errorf("failed to get containers: %v", err)
	}

	var wg sync.WaitGroup
	for _, container := range containers {
		wg.Add(1)
		go func(container types.Container) {
			defer wg.Done()
			result := p.pingContainer(ctx, container)
			if err := p.sendResult(ctx, result); err != nil {
				log.Printf("Failed to send result for container %s: %v", container.ID, err)
			}
		}(container)
	}
	wg.Wait()
	return nil
}

// Run запускает основной цикл пингера
func (p *Pinger) Run(ctx context.Context) error {
	log.Info("Starting pinger service...")
	defer log.Info("Pinger service stopped")

	ticker := time.NewTicker(p.config.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := p.pingCycle(ctx); err != nil {
				log.Printf("Ping cycle error: %v", err)
			}
		}
	}
}

func main() {
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetLevel(logrus.InfoLevel)

	config := Config{
		BackendURL:     "http://backend:8080/api/ping-results",
		PingInterval:   5 * time.Second,
		RetryAttempts:  3,
		RetryDelay:     time.Second,
		PingTimeout:    5 * time.Second,
		DockerEndpoint: "unix:///var/run/docker.sock",
	}

	pinger, err := NewPinger(config)
	if err != nil {
		log.Fatalf("Failed to create pinger: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		log.Printf("Received signal: %v", sig)
		cancel()
	}()

	log.Info("Starting pinger service...")
	if err := pinger.Run(ctx); err != nil && err != context.Canceled {
		log.Fatalf("Pinger error: %v", err)
	}
}
