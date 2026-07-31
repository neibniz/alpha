package supervisor

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/neibniz/alpha/xray-agent/internal/config"
)

type Supervisor struct {
	cfg           config.XrayConfig
	logCfg        config.LogConfig
	mu            sync.Mutex
	proc          *process
	configVersion string
	versionOnce   sync.Once
	xrayVersion   string
}

type process struct {
	cmd     *exec.Cmd
	done    chan error
	logFile *os.File
}

func New(cfg config.XrayConfig, logCfg ...config.LogConfig) *Supervisor {
	s := &Supervisor{cfg: cfg}
	if len(logCfg) > 0 {
		s.logCfg = logCfg[0]
	}
	return s
}

func (s *Supervisor) Apply(ctx context.Context, configData []byte) error {
	if err := s.SaveConfig(ctx, configData); err != nil {
		return err
	}
	return s.Restart(ctx)
}

func (s *Supervisor) SaveConfig(ctx context.Context, configData []byte) error {
	if s.cfg.ValidateConfig {
		if err := s.validate(ctx, configData); err != nil {
			return err
		}
	}
	if err := os.WriteFile(s.cfg.ConfigFile, configData, 0o644); err != nil {
		return err
	}
	return os.WriteFile(s.cfg.LastGoodFile, configData, 0o644)
}

func (s *Supervisor) Restart(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.proc != nil && s.proc.cmd.Process != nil {
		_ = s.proc.cmd.Process.Signal(os.Interrupt)
		select {
		case <-s.proc.done:
		case <-time.After(5 * time.Second):
			_ = s.proc.cmd.Process.Kill()
		}
		s.proc = nil
	}

	cmd := exec.CommandContext(ctx, s.cfg.BinPath, "run", "-config", s.cfg.ConfigFile)
	xrayLog, err := openAppend(s.logCfg.XrayErrorPath)
	if err != nil {
		return fmt.Errorf("open xray log: %w", err)
	}
	if xrayLog != nil {
		cmd.Stdout = xrayLog
		cmd.Stderr = xrayLog
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	if err := cmd.Start(); err != nil {
		if xrayLog != nil {
			_ = xrayLog.Close()
		}
		return fmt.Errorf("start xray-core: %w", err)
	}
	proc := &process{cmd: cmd, done: make(chan error, 1), logFile: xrayLog}
	s.proc = proc
	go s.wait(proc)
	return nil
}

func (s *Supervisor) Running() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.proc != nil && s.proc.cmd.Process != nil && s.proc.cmd.ProcessState == nil
}

func (s *Supervisor) PID() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.proc == nil || s.proc.cmd.Process == nil || s.proc.cmd.ProcessState != nil {
		return 0
	}
	return s.proc.cmd.Process.Pid
}

func (s *Supervisor) SetConfigVersion(version string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.configVersion = version
}

func (s *Supervisor) ConfigVersion() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.configVersion
}

func (s *Supervisor) XrayVersion() string {
	s.versionOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		output, err := exec.CommandContext(ctx, s.cfg.BinPath, "version").CombinedOutput()
		if err != nil {
			return
		}
		line := strings.TrimSpace(strings.SplitN(string(output), "\n", 2)[0])
		s.xrayVersion = line
	})
	return s.xrayVersion
}

func (s *Supervisor) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.proc == nil || s.proc.cmd.Process == nil {
		return
	}
	_ = s.proc.cmd.Process.Signal(os.Interrupt)
	_ = s.proc.cmd.Process.Kill()
	s.proc = nil
}

func (s *Supervisor) validate(ctx context.Context, configData []byte) error {
	tmp, err := os.CreateTemp(s.cfg.WorkDir, "xray-config-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(configData); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	cmd := xrayTestCommand(ctx, s.cfg.BinPath, tmp.Name())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("xray config test failed: %w: %s", err, string(output))
	}
	return nil
}

func xrayTestCommand(ctx context.Context, binPath string, configPath string) *exec.Cmd {
	return exec.CommandContext(ctx, binPath, "run", "-test", "-config", configPath)
}

func (s *Supervisor) wait(proc *process) {
	err := proc.cmd.Wait()
	if proc.logFile != nil {
		_ = proc.logFile.Close()
	}
	proc.done <- err
	s.mu.Lock()
	if s.proc == proc {
		s.proc = nil
	}
	s.mu.Unlock()
	if err != nil {
		log.Printf("xray-core exited: %v", err)
	}
}

func openAppend(path string) (*os.File, error) {
	if path == "" {
		return nil, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	return os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
}
